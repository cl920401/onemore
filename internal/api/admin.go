package api

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"onemore/internal/domain"
	"onemore/internal/store"
)

type channelView struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	BaseURL      string   `json:"base_url"`
	APIKeyMasked string   `json:"api_key_masked"`
	Models       []string `json:"models"`
	Enabled      bool     `json:"enabled"`
	CreatedAt    string   `json:"created_at"`
}

func (s *Server) toChannelView(rec store.ChannelRecord) channelView {
	return channelView{
		ID:           rec.ID,
		Name:         rec.Name,
		BaseURL:      rec.BaseURL,
		APIKeyMasked: s.maskKey(rec.APIKeyEnc),
		Models:       rec.Models,
		Enabled:      rec.Enabled,
		CreatedAt:    rec.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *Server) maskKey(enc []byte) string {
	plain, err := s.cipher.Decrypt(enc)
	if err != nil || len(plain) < 8 {
		return "****"
	}
	return "****" + plain[len(plain)-4:]
}

func (s *Server) createChannel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string   `json:"name"`
		BaseURL string   `json:"base_url"`
		APIKey  string   `json:"api_key"`
		Models  []string `json:"models"`
		Enabled *bool    `json:"enabled"`
	}
	if err := decode(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.BaseURL = strings.TrimSpace(req.BaseURL)
	if req.Name == "" || req.APIKey == "" || len(req.Models) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name, api_key and models are required")
		return
	}
	u, err := url.Parse(req.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "base_url must be an absolute http(s) URL")
		return
	}
	enc, err := s.cipher.Encrypt(req.APIKey)
	if err != nil {
		s.log.Error("encrypt channel key", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to store credential")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	id, err := s.repo.CreateChannel(r.Context(), req.Name, req.BaseURL, enc, req.Models, enabled)
	if err != nil {
		s.log.Error("create channel", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to create channel")
		return
	}
	_ = s.repo.InsertConfigAudit(r.Context(), "admin", "channel.create", req.Name, req.BaseURL)
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) listChannels(w http.ResponseWriter, r *http.Request) {
	records, err := s.repo.ListChannels(r.Context())
	if err != nil {
		s.log.Error("list channels", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to list channels")
		return
	}
	out := make([]channelView, 0, len(records))
	for _, rec := range records {
		out = append(out, s.toChannelView(rec))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) updateChannel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid channel id")
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := decode(w, r, &req); err != nil || req.Enabled == nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "enabled is required")
		return
	}
	if err := s.repo.SetChannelEnabled(r.Context(), id, *req.Enabled); err != nil {
		if notFound(w, err) {
			return
		}
		s.log.Error("update channel", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to update channel")
		return
	}
	_ = s.repo.InsertConfigAudit(r.Context(), "admin", "channel.update", idString(id), "enabled="+boolString(*req.Enabled))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) testChannel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid channel id")
		return
	}
	rec, err := s.repo.GetChannel(r.Context(), id)
	if err != nil {
		if notFound(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to load channel")
		return
	}
	key, err := s.cipher.Decrypt(rec.APIKeyEnc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to decrypt credential")
		return
	}
	if err := s.upstream.Ping(r.Context(), rec.BaseURL, key); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "message": "upstream unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type memberView struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	EnterpriseLimit    int64  `json:"enterprise_limit"`
	EnterpriseUsed     int64  `json:"enterprise_used"`
	EnterpriseReserved int64  `json:"enterprise_reserved"`
	Remaining          int64  `json:"remaining"`
	CreatedAt          string `json:"created_at"`
}

func toMemberView(m domain.Member) memberView {
	return memberView{
		ID:                 m.ID,
		Name:               m.Name,
		Role:               m.Role,
		Status:             m.Status,
		EnterpriseLimit:    m.EnterpriseLimit,
		EnterpriseUsed:     m.EnterpriseUsed,
		EnterpriseReserved: m.EnterpriseReserved,
		Remaining:          m.EnterpriseLimit - m.EnterpriseUsed - m.EnterpriseReserved,
		CreatedAt:          m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (s *Server) createMember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string `json:"name"`
		EnterpriseLimit int64  `json:"enterprise_limit"`
	}
	if err := decode(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.EnterpriseLimit < 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required and enterprise_limit must be >= 0")
		return
	}
	token, err := newToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to generate token")
		return
	}
	id, err := s.repo.CreateMember(r.Context(), req.Name, hashToken(token), req.EnterpriseLimit)
	if err != nil {
		s.log.Error("create member", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to create member")
		return
	}
	_ = s.repo.InsertConfigAudit(r.Context(), "admin", "member.create", req.Name, "enterprise_limit="+intString(req.EnterpriseLimit))
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":               id,
		"name":             req.Name,
		"token":            token,
		"enterprise_limit": req.EnterpriseLimit,
	})
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) {
	members, err := s.repo.ListMembers(r.Context())
	if err != nil {
		s.log.Error("list members", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to list members")
		return
	}
	out := make([]memberView, 0, len(members))
	for _, m := range members {
		out = append(out, toMemberView(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) setMemberQuota(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid member id")
		return
	}
	var req struct {
		Limit int64 `json:"limit"`
	}
	if err := decode(w, r, &req); err != nil || req.Limit < 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "limit must be >= 0")
		return
	}
	m, err := s.repo.MemberByID(r.Context(), id)
	if err != nil {
		if notFound(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to load member")
		return
	}
	if req.Limit < m.EnterpriseUsed+m.EnterpriseReserved {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "limit below used and reserved quota")
		return
	}
	if err := s.repo.SetMemberLimit(r.Context(), id, req.Limit); err != nil {
		if notFound(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to update quota")
		return
	}
	_ = s.repo.InsertConfigAudit(r.Context(), "admin", "member.quota", idString(id), "limit="+intString(req.Limit))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) setMemberStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid member id")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := decode(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	if req.Status != "active" && req.Status != "disabled" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "status must be active or disabled")
		return
	}
	if err := s.repo.SetMemberStatus(r.Context(), id, req.Status); err != nil {
		if notFound(w, err) {
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to update status")
		return
	}
	_ = s.repo.InsertConfigAudit(r.Context(), "admin", "member.status", idString(id), "status="+req.Status)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getPolicy(w http.ResponseWriter, r *http.Request) {
	terms, err := s.repo.BlockedTerms(r.Context())
	if err != nil {
		s.log.Error("get policy", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to load policy")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocked_terms": terms})
}

func (s *Server) setPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BlockedTerms []string `json:"blocked_terms"`
	}
	if err := decode(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON body")
		return
	}
	clean := make([]string, 0, len(req.BlockedTerms))
	for _, t := range req.BlockedTerms {
		if t = strings.TrimSpace(t); t != "" {
			clean = append(clean, t)
		}
	}
	if err := s.repo.SetBlockedTerms(r.Context(), clean); err != nil {
		s.log.Error("set policy", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to save policy")
		return
	}
	_ = s.repo.InsertConfigAudit(r.Context(), "admin", "policy.update", "egress", strings.Join(clean, ","))
	writeJSON(w, http.StatusOK, map[string]any{"blocked_terms": clean})
}

type auditView struct {
	RequestID        string `json:"request_id"`
	CreatedAt        string `json:"created_at"`
	MemberID         int64  `json:"member_id"`
	MemberName       string `json:"member_name"`
	ChannelID        int64  `json:"channel_id"`
	QuotaSource      string `json:"quota_source"`
	Model            string `json:"model"`
	Result           string `json:"result"`
	ErrorCode        string `json:"error_code"`
	HTTPStatus       int    `json:"http_status"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	TotalTokens      int64  `json:"total_tokens"`
	LatencyMS        int64  `json:"latency_ms"`
}

func (s *Server) listAudits(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := parsePositiveInt(raw); err == nil {
			limit = n
		}
	}
	audits, err := s.repo.ListAudits(r.Context(), limit, strings.TrimSpace(r.URL.Query().Get("request_id")))
	if err != nil {
		s.log.Error("list audits", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to list audits")
		return
	}
	out := make([]auditView, 0, len(audits))
	for _, a := range audits {
		out = append(out, auditView{
			RequestID:        a.RequestID,
			CreatedAt:        a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			MemberID:         a.MemberID,
			MemberName:       a.MemberName,
			ChannelID:        a.ChannelID,
			QuotaSource:      a.QuotaSource,
			Model:            a.Model,
			Result:           a.Result,
			ErrorCode:        a.ErrorCode,
			HTTPStatus:       a.HTTPStatus,
			PromptTokens:     a.PromptTokens,
			CompletionTokens: a.CompletionTokens,
			TotalTokens:      a.TotalTokens,
			LatencyMS:        a.LatencyMS,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func notFoundMember(w http.ResponseWriter, err error) bool {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "member not found")
		return true
	}
	return false
}
