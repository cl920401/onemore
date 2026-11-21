package quota

import "testing"

func TestEstimate(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		maxTokens int
		wantMin   int64
	}{
		{"defaults max tokens", "hello", 0, DefaultMaxTokens},
		{"uses max tokens", "hello", 100, 100},
		{"counts prompt chars", "abcdefgh", 10, 12},
		{"never below one", "", -5, DefaultMaxTokens},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Estimate(tc.prompt, tc.maxTokens); got < tc.wantMin {
				t.Fatalf("Estimate(%q, %d) = %d, want >= %d", tc.prompt, tc.maxTokens, got, tc.wantMin)
			}
		})
	}
}
