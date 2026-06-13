package indicators

// ScoreResult holds the aggregated technical score and derived signal.
type ScoreResult struct {
	RSIScore   float64 // individual RSI score (0.0–1.0)
	RSIDir     string  // "BUY" | "SELL" | "HOLD"
	MACDScore  float64 // individual MACD score
	MACDDir    string
	BBScore    float64 // individual BB score
	BBDir      string
	TechScore  float64 // weighted aggregate
	Signal     string  // final signal
	Confidence float64 // == TechScore
}

// Weights for aggregation.
const (
	RSIWeight  = 0.40
	MACDWeight = 0.40
	BBWeight   = 0.20
)

// ScoreRSI converts an RSI value to a directional score.
//
// Threshold table:
//
//	RSI ≤ 30       → (0.85, BUY)
//	30 < RSI ≤ 40  → (0.65, BUY)
//	60 ≤ RSI < 70  → (0.65, SELL)
//	RSI ≥ 70       → (0.85, SELL)
//	40 < RSI < 60  → (0.50, HOLD)
func ScoreRSI(rsi float64) (score float64, direction string) {
	switch {
	case rsi <= 30:
		return 0.85, "BUY"
	case rsi <= 40:
		return 0.65, "BUY"
	case rsi >= 70:
		return 0.85, "SELL"
	case rsi >= 60:
		return 0.65, "SELL"
	default:
		return 0.50, "HOLD"
	}
}

// ScoreMACD converts a MACDResult to a directional score.
//
// Rules:
//
//	Bullish crossover              → (0.80, BUY)
//	Bearish crossover              → (0.80, SELL)
//	No crossover, histogram > 0   → (0.60, BUY)
//	No crossover, histogram < 0   → (0.60, SELL)
//	No crossover, histogram == 0  → (0.50, HOLD)
func ScoreMACD(macd MACDResult) (score float64, direction string) {
	switch macd.Crossover {
	case "bullish":
		return 0.80, "BUY"
	case "bearish":
		return 0.80, "SELL"
	default:
		// No crossover — use histogram direction
		if macd.Histogram > 0 {
			return 0.60, "BUY"
		} else if macd.Histogram < 0 {
			return 0.60, "SELL"
		}
		return 0.50, "HOLD"
	}
}

// ScoreBB converts a BBPosition value to a directional score.
//
// Threshold rules:
//
//	BBPosition ≤ 0.10              → (0.80, BUY)
//	BBPosition ≥ 0.90              → (0.80, SELL)
//	0.10 < BBPosition < 0.90      → (0.50, HOLD)
func ScoreBB(bbPosition float64) (score float64, direction string) {
	switch {
	case bbPosition <= 0.10:
		return 0.80, "BUY"
	case bbPosition >= 0.90:
		return 0.80, "SELL"
	default:
		return 0.50, "HOLD"
	}
}

// ComputeScore aggregates individual indicator scores into a TechnicalScore
// and determines the final signal.
//
// Formula: TechScore = (RSIScore × 0.40) + (MACDScore × 0.40) + (BBScore × 0.20)
// Signal thresholds: >= 0.65 → BUY, <= 0.35 → SELL, else → HOLD.
// Confidence = TechScore.
func ComputeScore(rsi float64, macd MACDResult, bbPosition float64) ScoreResult {
	rsiScore, rsiDir := ScoreRSI(rsi)
	macdScore, macdDir := ScoreMACD(macd)
	bbScore, bbDir := ScoreBB(bbPosition)

	techScore := (rsiScore * RSIWeight) + (macdScore * MACDWeight) + (bbScore * BBWeight)

	var signal string
	switch {
	case techScore >= 0.65:
		signal = "BUY"
	case techScore <= 0.35:
		signal = "SELL"
	default:
		signal = "HOLD"
	}

	return ScoreResult{
		RSIScore:   rsiScore,
		RSIDir:     rsiDir,
		MACDScore:  macdScore,
		MACDDir:    macdDir,
		BBScore:    bbScore,
		BBDir:      bbDir,
		TechScore:  techScore,
		Signal:     signal,
		Confidence: techScore,
	}
}
