package indicators

import "math"

// BollingerResult holds the output of Bollinger Bands computation.
type BollingerResult struct {
	Upper      float64 // SMA + (stddev × multiplier)
	Middle     float64 // SMA(period)
	Lower      float64 // SMA - (stddev × multiplier)
	BBPosition float64 // (close - lower) / (upper - lower), clamped [0,1]
}

// CalcBollingerBands computes Bollinger Bands(period, multiplier).
// Requires len(closes) >= period.
// If upper == lower (zero bandwidth), BBPosition = 0.50.
func CalcBollingerBands(closes []float64, period int, multiplier float64) BollingerResult {
	n := len(closes)
	if n < period {
		return BollingerResult{}
	}

	// Step 1: Middle = SMA of last `period` closes
	middle := CalcSMA(closes, period)

	// Step 2: Standard deviation of last `period` closes (population stddev)
	start := n - period
	sumSqDiff := 0.0
	for i := start; i < n; i++ {
		diff := closes[i] - middle
		sumSqDiff += diff * diff
	}
	stddev := math.Sqrt(sumSqDiff / float64(period))

	// Step 3: Upper and Lower bands
	upper := middle + (multiplier * stddev)
	lower := middle - (multiplier * stddev)

	// Step 4: BBPosition
	var bbPosition float64
	if upper == lower {
		// Zero bandwidth: all prices in the window are identical
		bbPosition = 0.50
	} else {
		lastClose := closes[n-1]
		bbPosition = (lastClose - lower) / (upper - lower)

		// Clamp to [0.0, 1.0]
		if bbPosition < 0.0 {
			bbPosition = 0.0
		}
		if bbPosition > 1.0 {
			bbPosition = 1.0
		}
	}

	return BollingerResult{
		Upper:      upper,
		Middle:     middle,
		Lower:      lower,
		BBPosition: bbPosition,
	}
}
