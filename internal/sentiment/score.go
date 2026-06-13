package sentiment

// ComputeScore normalizes a sentiment direction and confidence level into a
// single score value suitable for downstream decision making.
//
// The confidence is clamped to [0.0, 1.0] before the formula is applied:
//   - bullish:  0.5 + (confidence × 0.5) → range [0.5, 1.0]
//   - bearish:  0.5 - (confidence × 0.5) → range [0.0, 0.5]
//   - neutral:  0.5
//
// Any unrecognized sentiment value is treated as neutral.
func ComputeScore(sentiment string, confidence float64) float64 {
	// Clamp confidence to [0.0, 1.0]
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	switch sentiment {
	case "bullish":
		return 0.5 + (confidence * 0.5)
	case "bearish":
		return 0.5 - (confidence * 0.5)
	default:
		return 0.5
	}
}
