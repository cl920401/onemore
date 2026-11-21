package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"onemore/internal/secret"
	"onemore/internal/store"
	"onemore/internal/upstream"
)

// TestAuditRequestIDIsolation needs a disposable MySQL database. It is skipped
// unless ONEMORE_TEST_DSN points at one, so `go test ./...` stays DB-free.
func TestAuditRequestIDIsolation(t *testing.T) {
	dsn := os.Getenv("ONEMORE_TEST_DSN")
	if dsn == "" {
		t.Skip("set ONEMORE_TEST_DSN to a disposable MySQL database to run this integration test")
	}
	ctx := context.Background()
	db, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	resetTables(t, ctx, db)
	repo := store.NewRepo(db)
	cipher, err := secret.New("integration-test-master-key")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	var upstreamCalls atomic.Int64
	upstreamSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := upstreamCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      fmt.Sprintf("upstream-%d", n),
			"object":  "chat.completion",
			"model":   "gpt-4o-mini",
			"choices": []any{map[string]any{"message": map[string]string{"role": "assistant", "content": "ok"}}},
			"usage":   map[string]int64{"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8},
		})
	}))
	defer upstreamSrv.Close()

	adminToken := "integration-admin-token"
	server := New(repo, cipher, upstream.New(), adminToken, t.TempDir(), logger)
	apiSrv := httptest.NewServer(server.Routes())
	defer apiSrv.Close()

	status, _, _ := doJSON(t, http.MethodPost, apiSrv.URL+"/api/channels", adminToken, map[string]any{
		"name":     "fake",
		"base_url": upstreamSrv.URL,
		"api_key":  "fake-key",
		"models":   []string{"gpt-4o-mini"},
	})
	if status != http.StatusCreated {
		t.Fatalf("create channel status = %d, want %d", status, http.StatusCreated)
	}

	status, raw, _ := doJSON(t, http.MethodPost, apiSrv.URL+"/api/members", adminToken, map[string]any{
		"name":             "alice",
		"enterprise_limit": 10000,
	})
	if status != http.StatusCreated {
		t.Fatalf("create member status = %d body=%s", status, raw)
	}
	var created struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(raw, &created); err != nil || created.Token == "" {
		t.Fatalf("decode member token: %v body=%s", err, raw)
	}

	call := func(content string) (platformID, modelID string) {
		status, body, header := doJSON(t, http.MethodPost, apiSrv.URL+"/v1/chat/completions", created.Token, map[string]any{
			"model":    "gpt-4o-mini",
			"messages": []any{map[string]string{"role": "user", "content": content}},
		})
		if status != http.StatusOK {
			t.Fatalf("chat status = %d body=%s", status, body)
		}
		var parsed struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			t.Fatalf("decode chat response: %v body=%s", err, body)
		}
		return header.Get("X-Request-ID"), parsed.ID
	}

	firstPlatformID, firstModelID := call("first call")
	secondPlatformID, secondModelID := call("second call")

	if firstPlatformID == "" || secondPlatformID == "" {
		t.Fatalf("missing X-Request-ID: %q %q", firstPlatformID, secondPlatformID)
	}
	if firstPlatformID == secondPlatformID {
		t.Fatalf("platform IDs must differ, both = %q", firstPlatformID)
	}
	if firstModelID != "upstream-1" || secondModelID != "upstream-2" {
		t.Fatalf("model response ids not preserved: %q %q", firstModelID, secondModelID)
	}

	byRequestID := func(id string) []map[string]any {
		status, body, _ := doJSON(t, http.MethodGet, apiSrv.URL+"/api/audits?request_id="+url.QueryEscape(id), adminToken, nil)
		if status != http.StatusOK {
			t.Fatalf("audit query status = %d body=%s", status, body)
		}
		var out []map[string]any
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("decode audits: %v body=%s", err, body)
		}
		return out
	}

	first := byRequestID(firstPlatformID)
	if len(first) != 1 || first[0]["request_id"] != firstPlatformID {
		t.Fatalf("first ID should match exactly one audit, got %v", first)
	}
	second := byRequestID(secondPlatformID)
	if len(second) != 1 || second[0]["request_id"] != secondPlatformID {
		t.Fatalf("second ID should match exactly one audit, got %v", second)
	}
	if first[0]["member_name"] != "alice" || second[0]["member_name"] != "alice" {
		t.Fatalf("unexpected audit attribution: %v / %v", first[0], second[0])
	}
	if missing := byRequestID("00000000000000000000000000000000"); len(missing) != 0 {
		t.Fatalf("unknown ID should return empty, got %v", missing)
	}
	if prefix := byRequestID(firstPlatformID[:8]); len(prefix) != 0 {
		t.Fatalf("request_id filter must be exact, prefix matched %v", prefix)
	}
}

func resetTables(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	for _, table := range []string{"audits", "config_audits", "members", "channels"} {
		if _, err := db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("reset %s: %v", table, err)
		}
	}
}

func doJSON(t *testing.T, method, target, token string, body any) (int, []byte, http.Header) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, target, reader)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return resp.StatusCode, raw, resp.Header
}
