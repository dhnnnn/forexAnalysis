package indicators

// MACDResult holds the output of MACD computation.
type MACDResult struct {
	MACDLine   float64 // EMA(fast) - EMA(slow)
	SignalLine float64 // EMA of MACDLine
	Histogram  float64 // MACDLine - SignalLine
	Crossover  string  // "bullish" | "bearish" | "none"
}

// CalcMACD computes MACD(fast, slow, signal) from close prices.
// Requires len(closes) >= slow + signal for valid computation.
// Crossover is detected by comparing current and previous histogram signs.
func CalcMACD(closes []float64, fast, slow, signal int) MACDResult {
	n := len(closes)
	if n < slow+signal {
		return MACDResult{Crossover: "none"}
	}

	// Step 1: Compute EMA(fast) and EMA(slow) series over all closes
	emaFast := CalcEMASeries(closes, fast)
	emaSlow := CalcEMASeries(closes, slow)

	// Step 2: Compute MACDLine series (valid from index slow-1 onward,
	// since EMA(slow) first becomes valid at index slow-1)
	macdStart := slow - 1
	macdSeriesLen := n - macdStart
	macdSeries := make([]float64, macdSeriesLen)
	for i := 0; i < macdSeriesLen; i++ {
		macdSeries[i] = emaFast[macdStart+i] - emaSlow[macdStart+i]
	}

	// Step 3: Compute signal line as EMA(signal) of the MACD series
	signalSeries := CalcEMASeries(macdSeries, signal)

	// Step 4: Current values (last element)
	lastIdx := macdSeriesLen - 1
	macdLine := macdSeries[lastIdx]
	signalLine := signalSeries[lastIdx]
	histogram := macdLine - signalLine

	// Step 5: Crossover detection — compare current and previous histogram signs
	crossover := "none"
	if lastIdx >= 1 {
		prevSignal := signalSeries[lastIdx-1]
		prevMACD := macdSeries[lastIdx-1]
		prevHistogram := prevMACD - prevSignal

		if prevHistogram < 0 && histogram > 0 {
			crossover = "bullish"
		} else if prevHistogram > 0 && histogram < 0 {
			crossover = "bearish"
		}
	}

	return MACDResult{
		MACDLine:   macdLine,
		SignalLine:  signalLine,
		Histogram:  histogram,
		Crossover:  crossover,
	}
}
