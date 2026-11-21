package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Fake upstream used by the M1 experiment. It never calls a real model and
// records how many requests it actually received so tests can assert zero
// egress on blocked requests.
var requestCount atomic.Int64

func main() {
	addr := os.Getenv("FAKE_UPSTREAM_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9099"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", handleChat)
	mux.HandleFunc("GET /v1/models", handleModels)
	mux.HandleFunc("GET /stats", handleStats)
	mux.HandleFunc("POST /stats/reset", func(w http.ResponseWriter, r *http.Request) {
		requestCount.Store(0)
		writeJSON(w, map[string]int64{"requests": 0})
	})

	log.Printf("fake upstream listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	requestCount.Add(1)

	if strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") == "fake-enterprise-429" {
		writeJSONStatus(w, http.StatusTooManyRequests, map[string]any{
			"error": map[string]string{"message": "rate limited"},
		})
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	writeJSON(w, map[string]any{
		"id":      "chatcmpl-fake",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   req.Model,
		"choices": []any{map[string]any{
			"index":         0,
			"message":       map[string]string{"role": "assistant", "content": "fake upstream response"},
			"finish_reason": "stop",
		}},
		"usage": map[string]int64{
			"prompt_tokens":     8,
			"completion_tokens": 4,
			"total_tokens":      12,
		},
	})
}

func handleModels(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"object": "list",
		"data":   []any{map[string]string{"id": "gpt-4o-mini", "object": "model"}},
	})
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]int64{"requests": requestCount.Load()})
}

func writeJSON(w http.ResponseWriter, v any) { writeJSONStatus(w, http.StatusOK, v) }

func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
