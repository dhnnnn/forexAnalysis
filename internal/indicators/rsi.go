package indicators

// CalcRSI computes RSI using Wilder's smoothing method.
// Requires len(closes) >= period + 1.
// Returns RSI value in range [0.0, 100.0].
// If avgLoss == 0 (no losses in the data), returns 100.0.
func CalcRSI(closes []float64, period int) float64 {
	n := len(closes)
	if n < period+1 {
		return 0
	}

	// Step 1: Compute price changes and separate gains/losses
	gains := make([]float64, n-1)
	losses := make([]float64, n-1)

	for i := 1; i < n; i++ {
		delta := closes[i] - closes[i-1]
		if delta > 0 {
			gains[i-1] = delta
		} else {
			losses[i-1] = -delta
		}
	}

	// Step 2: First average — SMA of first `period` gains and losses
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Step 3: Wilder's smoothing for subsequent values
	for i := period; i < n-1; i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
	}

	// Step 4: Compute RS and RSI
	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))

	return rsi
}
