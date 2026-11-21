package egress

import (
	"errors"
	"strings"
)

// ErrPolicyBlocked is returned when either egress gate rejects a request.
// A blocked request must never reach the upstream.
var ErrPolicyBlocked = errors.New("policy blocked")

// ModelAllowed is the first gate: it decides where a request may go by matching
// the requested model against the admin-configured allowlist of a channel.
func ModelAllowed(model string, allowlist []string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	for _, allowed := range allowlist {
		if strings.TrimSpace(allowed) == model {
			return true
		}
	}
	return false
}

// ContentViolation is the second gate: it decides what may leave by matching
// configured deterministic terms against the outbound content.
func ContentViolation(content string, blockedTerms []string) (string, bool) {
	lower := strings.ToLower(content)
	for _, term := range blockedTerms {
		t := strings.TrimSpace(term)
		if t == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(t)) {
			return t, true
		}
	}
	return "", false
}

// Evaluate runs both gates. Both must pass for the request to leave.
func Evaluate(model string, allowlist []string, content string, blockedTerms []string) error {
	if !ModelAllowed(model, allowlist) {
		return ErrPolicyBlocked
	}
	if _, hit := ContentViolation(content, blockedTerms); hit {
		return ErrPolicyBlocked
	}
	return nil
}
