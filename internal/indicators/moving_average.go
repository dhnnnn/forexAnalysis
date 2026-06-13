package indicators

// CalcSMA computes Simple Moving Average over the last `period` elements.
// If len(closes) < period, it computes the average over all available elements.
func CalcSMA(closes []float64, period int) float64 {
	n := len(closes)
	if n == 0 {
		return 0
	}

	if period > n {
		period = n
	}

	sum := 0.0
	for i := n - period; i < n; i++ {
		sum += closes[i]
	}

	return sum / float64(period)
}

// CalcEMA computes Exponential Moving Average for the given period.
// Returns the final EMA value (last element of the series).
// Multiplier = 2 / (period + 1).
// Seed value: SMA of first `period` elements.
// If len(closes) < period, it returns the SMA of all available elements.
func CalcEMA(closes []float64, period int) float64 {
	n := len(closes)
	if n == 0 {
		return 0
	}

	if n < period {
		// Graceful degradation: compute SMA of all available data
		return CalcSMA(closes, n)
	}

	// Seed with SMA of first `period` elements
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	ema := sum / float64(period)

	// Multiplier
	multiplier := 2.0 / float64(period+1)

	// Apply EMA formula from index `period` onward
	for i := period; i < n; i++ {
		ema = (closes[i]-ema)*multiplier + ema
	}

	return ema
}

// CalcEMASeries computes the full EMA series (used internally by MACD).
// Returns a slice of the same length as closes.
// First `period-1` entries are 0, then EMA from index `period-1` onward
// (seed = SMA of first `period` elements).
func CalcEMASeries(closes []float64, period int) []float64 {
	n := len(closes)
	result := make([]float64, n)

	if n < period {
		return result
	}

	// Compute SMA of first `period` elements as seed
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	sma := sum / float64(period)

	// Set seed at index period-1
	result[period-1] = sma

	// Multiplier
	multiplier := 2.0 / float64(period+1)

	// Compute EMA from index `period` onward
	for i := period; i < n; i++ {
		result[i] = (closes[i]-result[i-1])*multiplier + result[i-1]
	}

	return result
}
