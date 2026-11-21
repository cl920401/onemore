package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrRateLimited = errors.New("upstream rate limited")

// UnknownError means the upstream call may have executed but the result cannot
// be verified. The caller must keep the reservation and must not retry.
type UnknownError struct {
	Status int
	Reason string
}

func (e *UnknownError) Error() string {
	return fmt.Sprintf("upstream result unknown: status=%d reason=%s", e.Status, e.Reason)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
	TotalTokens      int64 `json:"total_tokens"`
}

type ChatResponse struct {
	ID      string
	Model   string
	Content string
	Usage   Usage
}

type Client struct {
	http *http.Client
}

func New() *Client {
	return &Client{http: &http.Client{Timeout: 30 * time.Second}}
}

// Ping checks connectivity to an admin-configured upstream without consuming quota.
func (c *Client) Ping(ctx context.Context, baseURL, apiKey string) error {
	url := strings.TrimRight(baseURL, "/") + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build ping request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ping upstream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ping upstream: status %d", resp.StatusCode)
	}
	return nil
}

// Chat performs a non-streaming OpenAI-compatible completion. baseURL is an
// admin-configured destination; callers must run egress gates beforehand.
func (c *Client) Chat(ctx context.Context, baseURL, apiKey string, req ChatRequest) (*ChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream request: %w", err)
	}
	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, &UnknownError{Status: http.StatusGatewayTimeout, Reason: "timeout"}
		}
		return nil, &UnknownError{Status: http.StatusBadGateway, Reason: "disconnected"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &UnknownError{Status: http.StatusBadGateway, Reason: fmt.Sprintf("status %d", resp.StatusCode)}
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, &UnknownError{Status: http.StatusBadGateway, Reason: "read body"}
	}

	var decoded struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, &UnknownError{Status: http.StatusBadGateway, Reason: "unparseable response"}
	}
	out := &ChatResponse{ID: decoded.ID, Model: decoded.Model, Usage: decoded.Usage}
	if len(decoded.Choices) > 0 {
		out.Content = decoded.Choices[0].Message.Content
	}
	if out.Usage.TotalTokens == 0 {
		out.Usage.TotalTokens = out.Usage.PromptTokens + out.Usage.CompletionTokens
	}
	return out, nil
}
