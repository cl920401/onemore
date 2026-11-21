package quota

const DefaultMaxTokens = 256

// Estimate returns the amount to reserve before an upstream call. It is a
// conservative floor, not a billing figure; the actual amount is settled from
// upstream usage afterwards.
func Estimate(prompt string, maxTokens int) int64 {
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}
	estimate := int64(len(prompt)/4) + int64(maxTokens)
	if estimate < 1 {
		estimate = 1
	}
	return estimate
}
