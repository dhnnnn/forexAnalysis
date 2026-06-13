package feed

import (
	"fmt"
	"math"
	"time"
)

// OHLCVCandle adalah representasi raw candle dari data feed sebelum dinormalisasi.
type OHLCVCandle struct {
	Pair      string    `json:"pair"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Spread    float64   `json:"spread"`
	Timeframe string    `json:"timeframe"`
	Timestamp time.Time `json:"timestamp"`
}

// Validate memvalidasi candle sebelum diteruskan ke pipeline.
// Return error jika data invalid — jangan panic.
func Validate(c OHLCVCandle) error {
	// Cek nilai positif dan bukan NaN/Inf
	for _, v := range []struct {
		name string
		val  float64
	}{
		{"Open", c.Open},
		{"High", c.High},
		{"Low", c.Low},
		{"Close", c.Close},
	} {
		if v.val <= 0 {
			return fmt.Errorf("normalizer: %s harus > 0, got %.5f", v.name, v.val)
		}
		if math.IsNaN(v.val) || math.IsInf(v.val, 0) {
			return fmt.Errorf("normalizer: %s is NaN or Inf", v.name)
		}
	}

	// Cek relasi harga OHLC
	if c.High < c.Low {
		return fmt.Errorf("normalizer: High (%.5f) < Low (%.5f)", c.High, c.Low)
	}
	if c.High < c.Open {
		return fmt.Errorf("normalizer: High (%.5f) < Open (%.5f)", c.High, c.Open)
	}
	if c.High < c.Close {
		return fmt.Errorf("normalizer: High (%.5f) < Close (%.5f)", c.High, c.Close)
	}
	if c.Low > c.Open {
		return fmt.Errorf("normalizer: Low (%.5f) > Open (%.5f)", c.Low, c.Open)
	}
	if c.Low > c.Close {
		return fmt.Errorf("normalizer: Low (%.5f) > Close (%.5f)", c.Low, c.Close)
	}

	// Cek timestamp
	if c.Timestamp.IsZero() {
		return fmt.Errorf("normalizer: timestamp is zero")
	}

	// Cek pair
	if c.Pair == "" {
		return fmt.Errorf("normalizer: pair is empty")
	}

	return nil
}

// Normalize memvalidasi dan menormalisasi candle ke 5 desimal (pip precision).
// Return candle yang sudah bersih, atau error jika data invalid.
func Normalize(c OHLCVCandle) (OHLCVCandle, error) {
	if err := Validate(c); err != nil {
		return OHLCVCandle{}, err
	}

	// Bulatkan ke 5 desimal (pip precision forex)
	c.Open = roundTo5(c.Open)
	c.High = roundTo5(c.High)
	c.Low = roundTo5(c.Low)
	c.Close = roundTo5(c.Close)
	c.Spread = roundTo5(c.Spread)

	return c, nil
}

// roundTo5 membulatkan float ke 5 desimal.
func roundTo5(v float64) float64 {
	return math.Round(v*100000) / 100000
}
