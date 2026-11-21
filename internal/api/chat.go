package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"onemore/internal/domain"
	"onemore/internal/egress"
	"onemore/internal/quota"
	"onemore/internal/upstream"
)

func (s *Server) chatCompletions(w http.ResponseWriter, r *http.Request) {
	member, ok := currentMember(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "member token required")
		return
	}

	// The platform audit ID is returned as a header; the model's own response id
	// stays untouched in the JSON body.
	requestID := newRequestID()
	w.Header().Set("X-Request-ID", requestID)

	var req upstream.ChatRequest
	if err := decode(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	req.Model = strings.TrimSpace(req.Model)
	if req.Model == "" || len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "model and messages are required")
		return
	}

	content := joinContent(req.Messages)

	channels, err := s.repo.EnabledChannels(r.Context())
	if err != nil {
		s.log.Error("load channels", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to route request")
		return
	}
	var chosen *channelChoice
	for _, ch := range channels {
		if egress.ModelAllowed(req.Model, ch.Models) {
			chosen = &channelChoice{id: ch.ID, baseURL: ch.BaseURL, keyEnc: ch.APIKeyEnc}
			break
		}
	}
	if chosen == nil {
		s.recordAudit(r, member, domain.Audit{
			RequestID: requestID, Model: req.Model, QuotaSource: "enterprise",
			Result: "policy_rejected", ErrorCode: "POLICY_BLOCKED", HTTPStatus: http.StatusForbidden,
		})
		writeError(w, http.StatusForbidden, "POLICY_BLOCKED", "model is not allowed by any enabled channel")
		return
	}

	terms, err := s.repo.BlockedTerms(r.Context())
	if err != nil {
		s.log.Error("load policy", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to evaluate policy")
		return
	}
	if _, hit := egress.ContentViolation(content, terms); hit {
		s.recordAudit(r, member, domain.Audit{
			RequestID: requestID, ChannelID: chosen.id, Model: req.Model, QuotaSource: "enterprise",
			Result: "policy_rejected", ErrorCode: "POLICY_BLOCKED", HTTPStatus: http.StatusForbidden,
		})
		writeError(w, http.StatusForbidden, "POLICY_BLOCKED", "request blocked by egress policy")
		return
	}

	estimate := quota.Estimate(content, req.MaxTokens)
	reserved, err := s.repo.ReserveQuota(r.Context(), member.ID, estimate)
	if err != nil {
		s.log.Error("reserve quota", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to reserve quota")
		return
	}
	if !reserved {
		s.recordAudit(r, member, domain.Audit{
			RequestID: requestID, ChannelID: chosen.id, Model: req.Model, QuotaSource: "enterprise",
			Result: "quota_exceeded", ErrorCode: "ORG_QUOTA_EXCEEDED", HTTPStatus: http.StatusTooManyRequests,
		})
		writeError(w, http.StatusTooManyRequests, "ORG_QUOTA_EXCEEDED", "enterprise quota exhausted")
		return
	}

	key, err := s.cipher.Decrypt(chosen.keyEnc)
	if err != nil {
		_ = s.repo.ReleaseQuota(r.Context(), member.ID, estimate)
		s.log.Error("decrypt channel key", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to load credential")
		return
	}

	start := time.Now()
	resp, callErr := s.upstream.Chat(r.Context(), chosen.baseURL, key, req)
	latency := time.Since(start).Milliseconds()

	if callErr == nil {
		actual := resp.Usage.TotalTokens
		if actual <= 0 {
			actual = estimate
		}
		if err := s.repo.SettleQuota(r.Context(), member.ID, estimate, actual); err != nil {
			s.log.Error("settle quota", "err", err)
		}
		s.recordAudit(r, member, domain.Audit{
			RequestID: requestID, ChannelID: chosen.id, Model: req.Model, QuotaSource: "enterprise",
			Result: "success", HTTPStatus: http.StatusOK,
			PromptTokens: resp.Usage.PromptTokens, CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens: actual, LatencyMS: latency,
		})
		writeJSON(w, http.StatusOK, map[string]any{
			"id":      resp.ID,
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []any{map[string]any{
				"index":         0,
				"message":       map[string]string{"role": "assistant", "content": resp.Content},
				"finish_reason": "stop",
			}},
			"usage": map[string]int64{
				"prompt_tokens":     resp.Usage.PromptTokens,
				"completion_tokens": resp.Usage.CompletionTokens,
				"total_tokens":      actual,
			},
		})
		return
	}

	if errors.Is(callErr, upstream.ErrRateLimited) {
		_ = s.repo.ReleaseQuota(r.Context(), member.ID, estimate)
		s.recordAudit(r, member, domain.Audit{
			RequestID: requestID, ChannelID: chosen.id, Model: req.Model, QuotaSource: "enterprise",
			Result: "rate_limited", ErrorCode: "UPSTREAM_RATE_LIMITED", HTTPStatus: http.StatusTooManyRequests,
			LatencyMS: latency,
		})
		writeError(w, http.StatusTooManyRequests, "UPSTREAM_RATE_LIMITED", "upstream rate limited")
		return
	}

	status := http.StatusBadGateway
	var unknown *upstream.UnknownError
	if errors.As(callErr, &unknown) {
		status = unknown.Status
	}
	// Keep the reservation: the upstream call may have executed and must be reviewed.
	s.recordAudit(r, member, domain.Audit{
		RequestID: requestID, ChannelID: chosen.id, Model: req.Model, QuotaSource: "enterprise",
		Result: "unknown", ErrorCode: "UPSTREAM_RESULT_UNKNOWN", HTTPStatus: status,
		LatencyMS: latency,
	})
	writeError(w, status, "UPSTREAM_RESULT_UNKNOWN", "upstream result unknown; reservation kept for review")
}

type channelChoice struct {
	id      int64
	baseURL string
	keyEnc  []byte
}

func joinContent(messages []upstream.Message) string {
	parts := make([]string, 0, len(messages))
	for _, m := range messages {
		parts = append(parts, m.Content)
	}
	return strings.Join(parts, "\n")
}

func (s *Server) recordAudit(r *http.Request, member domain.Member, a domain.Audit) {
	if a.MemberID == 0 {
		a.MemberID = member.ID
	}
	if a.MemberName == "" {
		a.MemberName = member.Name
	}
	if a.QuotaSource == "" {
		a.QuotaSource = "enterprise"
	}
	if err := s.repo.InsertAudit(r.Context(), a); err != nil {
		s.log.Error("insert audit", "err", err)
	}
}
