package egress

import "testing"

func TestModelAllowed(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		allowlist []string
		want      bool
	}{
		{"exact match", "gpt-4o-mini", []string{"gpt-4o-mini", "gpt-4o"}, true},
		{"not listed", "gpt-4o-mini", []string{"claude-3"}, false},
		{"empty model", "", []string{"gpt-4o-mini"}, false},
		{"empty allowlist", "gpt-4o-mini", nil, false},
		{"blank entries", "gpt-4o-mini", []string{" ", "gpt-4o-mini"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ModelAllowed(tc.model, tc.allowlist); got != tc.want {
				t.Fatalf("ModelAllowed(%q, %v) = %v, want %v", tc.model, tc.allowlist, got, tc.want)
			}
		})
	}
}

func TestContentViolation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		terms   []string
		hit     bool
		term    string
	}{
		{"matches case-insensitively", "please leak secret_marker now", []string{"SECRET_MARKER"}, true, "SECRET_MARKER"},
		{"no match", "ordinary text", []string{"SECRET_MARKER"}, false, ""},
		{"blank term ignored", "ordinary text", []string{"", "  "}, false, ""},
		{"no terms", "anything", nil, false, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term, hit := ContentViolation(tc.content, tc.terms)
			if hit != tc.hit || term != tc.term {
				t.Fatalf("ContentViolation() = (%q, %v), want (%q, %v)", term, hit, tc.term, tc.hit)
			}
		})
	}
}

func TestEvaluateBothGates(t *testing.T) {
	allow := []string{"gpt-4o-mini"}
	terms := []string{"SECRET_MARKER"}

	tests := []struct {
		name      string
		model     string
		content   string
		wantBlock bool
	}{
		{"both pass", "gpt-4o-mini", "hello", false},
		{"allowlist fails", "unknown-model", "hello", true},
		{"content gate fails", "gpt-4o-mini", "send SECRET_MARKER", true},
		{"both fail", "unknown-model", "send SECRET_MARKER", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := Evaluate(tc.model, allow, tc.content, terms)
			if blocked := err != nil; blocked != tc.wantBlock {
				t.Fatalf("Evaluate() blocked = %v, want %v (err=%v)", blocked, tc.wantBlock, err)
			}
		})
	}
}
