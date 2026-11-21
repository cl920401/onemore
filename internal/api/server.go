package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"onemore/internal/domain"
	"onemore/internal/secret"
	"onemore/internal/store"
	"onemore/internal/upstream"
)

type Server struct {
	repo       *store.Repo
	cipher     *secret.Cipher
	upstream   *upstream.Client
	adminToken string
	webDir     string
	log        *slog.Logger
}

func New(repo *store.Repo, cipher *secret.Cipher, upstreamClient *upstream.Client, adminToken, webDir string, log *slog.Logger) *Server {
	return &Server{
		repo:       repo,
		cipher:     cipher,
		upstream:   upstreamClient,
		adminToken: adminToken,
		webDir:     webDir,
		log:        log,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /api/channels", s.admin(s.listChannels))
	mux.HandleFunc("POST /api/channels", s.admin(s.createChannel))
	mux.HandleFunc("PATCH /api/channels/{id}", s.admin(s.updateChannel))
	mux.HandleFunc("POST /api/channels/{id}/test", s.admin(s.testChannel))

	mux.HandleFunc("GET /api/members", s.admin(s.listMembers))
	mux.HandleFunc("POST /api/members", s.admin(s.createMember))
	mux.HandleFunc("PUT /api/members/{id}/quota", s.admin(s.setMemberQuota))
	mux.HandleFunc("PUT /api/members/{id}/status", s.admin(s.setMemberStatus))

	mux.HandleFunc("GET /api/policy", s.admin(s.getPolicy))
	mux.HandleFunc("PUT /api/policy", s.admin(s.setPolicy))
	mux.HandleFunc("GET /api/audits", s.admin(s.listAudits))

	mux.HandleFunc("POST /v1/chat/completions", s.member(s.chatCompletions))

	mux.HandleFunc("/", s.serveWeb)
	return mux
}

type ctxKey struct{}

func (s *Server) admin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.adminToken)) != 1 {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "admin token required")
			return
		}
		next(w, r)
	}
}

func (s *Server) member(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "member token required")
			return
		}
		m, err := s.repo.MemberByTokenHash(r.Context(), hashToken(token))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			return
		}
		if m.Status != "active" {
			writeError(w, http.StatusForbidden, "POLICY_BLOCKED", "member disabled")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey{}, m)
		next(w, r.WithContext(ctx))
	}
}

func currentMember(ctx context.Context) (domain.Member, bool) {
	m, ok := ctx.Value(ctxKey{}).(domain.Member)
	return m, ok
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func decode(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	return dec.Decode(v)
}

func pathID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func notFound(w http.ResponseWriter, err error) bool {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
		return true
	}
	return false
}

func (s *Server) serveWeb(w http.ResponseWriter, r *http.Request) {
	clean := filepath.Clean("/" + r.URL.Path)
	full := filepath.Join(s.webDir, clean)
	if fi, err := os.Stat(full); err == nil && !fi.IsDir() {
		http.ServeFile(w, r, full)
		return
	}
	index := filepath.Join(s.webDir, "index.html")
	if _, err := os.Stat(index); err == nil {
		http.ServeFile(w, r, index)
		return
	}
	writeError(w, http.StatusNotFound, "NOT_FOUND", "console not built; run npm run build in web/")
}
