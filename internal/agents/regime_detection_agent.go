package agents

import (
	"context"
	"math"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
)

// ════════════════════════════════════════════════════════════════════════
// RegimeDetectionAgent — deteksi kondisi pasar (Trending/Ranging/Breakout/dll)
// ════════════════════════════════════════════════════════════════════════

// RegimeDetectionAgent menganalisis candle buffer dan mengklasifikasikan
// kondisi pasar menggunakan ADX (kekuatan trend), ATR (volatilitas),
// dan Bollinger Band width.
type RegimeDetectionAgent struct {
	adxPeriod    int
	atrPeriod    int
	adxThreshold float64 // di atas ini = trending
	volThreshold float64 // ATR/price relatif, di atas ini = high volatility
}

// RegimeConfig menyimpan konfigurasi dari config.yaml untuk RegimeDetectionAgent.
type RegimeConfig struct {
	ADXPeriod    int
	ATRPeriod    int
	ADXThreshold float64
	VolThreshold float64
}

// NewRegimeDetectionAgent membuat instance baru dengan parameter default industri.
func NewRegimeDetectionAgent() *RegimeDetectionAgent {
	return &RegimeDetectionAgent{
		adxPeriod:    14,
		atrPeriod:    14,
		adxThreshold: 25.0,  // ADX > 25 = trending (standar industri)
		volThreshold: 0.015, // ATR/price > 1.5% = high volatility
	}
}

// NewRegimeDetectionAgentWithConfig membuat instance dengan konfigurasi custom.
func NewRegimeDetectionAgentWithConfig(cfg RegimeConfig) *RegimeDetectionAgent {
	agent := NewRegimeDetectionAgent()
	if cfg.ADXPeriod > 0 {
		agent.adxPeriod = cfg.ADXPeriod
	}
	if cfg.ATRPeriod > 0 {
		agent.atrPeriod = cfg.ATRPeriod
	}
	if cfg.ADXThreshold > 0 {
		agent.adxThreshold = cfg.ADXThreshold
	}
	if cfg.VolThreshold > 0 {
		agent.volThreshold = cfg.VolThreshold
	}
	return agent
}

// Name mengembalikan identifier agent.
func (r *RegimeDetectionAgent) Name() string {
	return "RegimeDetectionAgent"
}

// Detect mengklasifikasikan regime dari slice candle OHLCV.
// Input: candles []Candle dari MarketDataAgent (minimal 30 candle).
// Output: RegimeContext yang dikirim ke seluruh pipeline.
func (r *RegimeDetectionAgent) Detect(ctx context.Context, pair string, candles []Candle) knowledge.RegimeContext {
	if len(candles) < 30 {
		return knowledge.RegimeContext{
			Pair:       pair,
			Regime:     knowledge.RegimeUnknown,
			DetectedAt: time.Now(),
		}
	}

	adx := r.calculateADX(candles)
	atr := r.calculateATR(candles)
	lastPrice := candles[len(candles)-1].Close
	relVol := atr / lastPrice

	bbWidth := r.calculateBBWidth(candles, 20, 2.0)
	trendStrength := math.Min(adx/50.0, 1.0) // normalisasi 0–1

	regime := r.classify(adx, relVol, bbWidth)

	return knowledge.RegimeContext{
		Pair:          pair,
		Regime:        regime,
		ADX:           adx,
		ATR:           atr,
		Volatility:    relVol,
		TrendStrength: trendStrength,
		DetectedAt:    time.Now(),
	}
}

// classify menentukan MarketRegime berdasarkan kombinasi ADX, volatilitas relatif,
// dan Bollinger Band width.
func (r *RegimeDetectionAgent) classify(adx, relVol, bbWidth float64) knowledge.MarketRegime {
	switch {
	case relVol > r.volThreshold*1.8 && bbWidth > 0.03:
		return knowledge.RegimeBreakout
	case adx > r.adxThreshold && relVol > r.volThreshold:
		return knowledge.RegimeTrending
	case adx > r.adxThreshold && relVol <= r.volThreshold:
		return knowledge.RegimeLowVolatility
	case adx <= r.adxThreshold && relVol > r.volThreshold*1.5:
		return knowledge.RegimeHighVolatility
	default:
		return knowledge.RegimeRanging
	}
}

// ════════════════════════════════════════════════════════════════════════
// ADX Calculation — Average Directional Index (Wilder's smoothing)
// ════════════════════════════════════════════════════════════════════════

// calculateADX menghitung Average Directional Index (14 periode).
// Menggunakan formula Wilder's smoothed DX.
func (r *RegimeDetectionAgent) calculateADX(candles []Candle) float64 {
	period := r.adxPeriod
	if len(candles) < period*2 {
		return 0
	}

	// Hitung +DM, -DM, dan TR untuk setiap candle
	trValues := make([]float64, 0, len(candles)-1)
	plusDMValues := make([]float64, 0, len(candles)-1)
	minusDMValues := make([]float64, 0, len(candles)-1)

	for i := 1; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevHigh := candles[i-1].High
		prevLow := candles[i-1].Low
		prevClose := candles[i-1].Close

		// True Range
		tr := math.Max(high-low, math.Max(
			math.Abs(high-prevClose),
			math.Abs(low-prevClose),
		))
		trValues = append(trValues, tr)

		// Directional Movement
		plusDM := 0.0
		if high-prevHigh > prevLow-low && high-prevHigh > 0 {
			plusDM = high - prevHigh
		}
		plusDMValues = append(plusDMValues, plusDM)

		minusDM := 0.0
		if prevLow-low > high-prevHigh && prevLow-low > 0 {
			minusDM = prevLow - low
		}
		minusDMValues = append(minusDMValues, minusDM)
	}

	if len(trValues) < period {
		return 0
	}

	// Wilder smoothing: pertama = sum of first N, lalu smoothed = prev - (prev/N) + current
	smoothedTR := sum(trValues[:period])
	smoothedPlusDM := sum(plusDMValues[:period])
	smoothedMinusDM := sum(minusDMValues[:period])

	dxValues := make([]float64, 0)

	// Hitung DX pertama
	if smoothedTR > 0 {
		diPlus := (smoothedPlusDM / smoothedTR) * 100
		diMinus := (smoothedMinusDM / smoothedTR) * 100
		diSum := diPlus + diMinus
		if diSum > 0 {
			dx := (math.Abs(diPlus-diMinus) / diSum) * 100
			dxValues = append(dxValues, dx)
		}
	}

	// Smoothed values untuk periode berikutnya
	for i := period; i < len(trValues); i++ {
		smoothedTR = smoothedTR - (smoothedTR / float64(period)) + trValues[i]
		smoothedPlusDM = smoothedPlusDM - (smoothedPlusDM / float64(period)) + plusDMValues[i]
		smoothedMinusDM = smoothedMinusDM - (smoothedMinusDM / float64(period)) + minusDMValues[i]

		if smoothedTR == 0 {
			continue
		}
		diPlus := (smoothedPlusDM / smoothedTR) * 100
		diMinus := (smoothedMinusDM / smoothedTR) * 100
		diSum := diPlus + diMinus
		if diSum == 0 {
			continue
		}
		dx := (math.Abs(diPlus-diMinus) / diSum) * 100
		dxValues = append(dxValues, dx)
	}

	if len(dxValues) == 0 {
		return 0
	}

	// ADX = Wilder smoothed average dari DX values
	if len(dxValues) < period {
		return avg(dxValues)
	}

	// First ADX = simple average of first N DX values
	adx := avg(dxValues[:period])

	// Smooth ADX selanjutnya
	for i := period; i < len(dxValues); i++ {
		adx = (adx*float64(period-1) + dxValues[i]) / float64(period)
	}

	return adx
}

// ════════════════════════════════════════════════════════════════════════
// ATR Calculation — Average True Range
// ════════════════════════════════════════════════════════════════════════

// calculateATR menghitung Average True Range untuk N periode terakhir.
func (r *RegimeDetectionAgent) calculateATR(candles []Candle) float64 {
	if len(candles) < 2 {
		return 0
	}

	period := r.atrPeriod
	start := len(candles) - period
	if start < 1 {
		start = 1
	}

	total := 0.0
	count := 0
	for i := start; i < len(candles); i++ {
		tr := math.Max(candles[i].High-candles[i].Low, math.Max(
			math.Abs(candles[i].High-candles[i-1].Close),
			math.Abs(candles[i].Low-candles[i-1].Close),
		))
		total += tr
		count++
	}

	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// ════════════════════════════════════════════════════════════════════════
// Bollinger Band Width — lebar relatif terhadap harga
// ════════════════════════════════════════════════════════════════════════

// calculateBBWidth menghitung lebar Bollinger Band relatif terhadap harga tengah.
func (r *RegimeDetectionAgent) calculateBBWidth(candles []Candle, period int, multiplier float64) float64 {
	if len(candles) < period {
		return 0
	}

	recent := candles[len(candles)-period:]

	// Simple Moving Average
	total := 0.0
	for _, c := range recent {
		total += c.Close
	}
	mean := total / float64(period)

	// Standard Deviation
	variance := 0.0
	for _, c := range recent {
		diff := c.Close - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(period))

	upper := mean + multiplier*stddev
	lower := mean - multiplier*stddev

	if mean == 0 {
		return 0
	}
	return (upper - lower) / mean
}

// ════════════════════════════════════════════════════════════════════════
// Helper functions
// ════════════════════════════════════════════════════════════════════════

func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	return sum(values) / float64(len(values))
}
