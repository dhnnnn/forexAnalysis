package feed

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"
)

// WebSocketFeed mewakili koneksi WebSocket ke broker (e.g. OANDA).
// Untuk MVP ini masih stub — akan diimplementasi penuh di Minggu 2.
type WebSocketFeed struct {
	Output chan OHLCVCandle // Channel output candle yang sudah dinormalisasi

	url    string
	apiKey string
	pairs  []string
	done   chan struct{}
	mu     sync.Mutex
}

// NewWebSocketFeed membuat feed WebSocket baru.
func NewWebSocketFeed(url, apiKey string, pairs []string) *WebSocketFeed {
	return &WebSocketFeed{
		Output: make(chan OHLCVCandle, 100), // buffered channel kapasitas 100
		url:    url,
		apiKey: apiKey,
		pairs:  pairs,
		done:   make(chan struct{}),
	}
}

// Start memulai koneksi WebSocket dan mulai mengirim candle ke channel Output.
// Untuk MVP: menghasilkan mock candle setiap interval.
// TODO Minggu 2: ganti dengan koneksi WebSocket nyata ke OANDA.
func (f *WebSocketFeed) Start(ctx context.Context) {
	slog.Info("WebSocketFeed starting (mock mode)", "pairs", f.pairs)

	go func() {
		ticker := time.NewTicker(5 * time.Second) // mock: setiap 5 detik
		defer ticker.Stop()

		basePrice := map[string]float64{
			"EUR_USD": 1.08450,
			"GBP_USD": 1.27230,
			"USD_JPY": 149.850,
			"AUD_USD": 0.66780,
		}

		for {
			select {
			case <-ctx.Done():
				slog.Info("WebSocketFeed stopping")
				close(f.done)
				return
			case t := <-ticker.C:
				for _, pair := range f.pairs {
					price, ok := basePrice[pair]
					if !ok {
						price = 1.00000
					}

					// Simulasi pergerakan harga dengan tren naik-turun (sinusoidal + noise)
					// Ini bikin indikator teknikal kadang trigger BUY/SELL, bukan cuma HOLD
					seconds := float64(t.Unix())
					// Tren sinusoidal: cycle ~60 candle (5 menit = 300 detik di mock)
					trend := math.Sin(seconds/150.0) * 0.00080
					// Noise random kecil
					noise := float64(t.UnixNano()%100-50) * 0.00001

					closePrice := price + trend + noise
					high := math.Max(price+trend, closePrice) + 0.00020
					low := math.Min(price+trend, closePrice) - 0.00015

					candle := OHLCVCandle{
						Pair:      pair,
						Open:      price + trend - noise,
						High:      high,
						Low:       low,
						Close:     closePrice,
						Volume:    float64(t.Unix() % 10000),
						Spread:    1.2,
						Timeframe: "1h",
						Timestamp: t,
					}

					// Normalisasi sebelum publish
					normalized, err := Normalize(candle)
					if err != nil {
						slog.Warn("WebSocketFeed: invalid candle dropped",
							"pair", pair, "error", err)
						continue
					}

					// Non-blocking send ke channel
					select {
					case f.Output <- normalized:
					default:
						slog.Warn("WebSocketFeed: output channel full, dropping candle",
							"pair", pair)
					}
				}
			}
		}
	}()
}

// Stop menghentikan feed.
func (f *WebSocketFeed) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case <-f.done:
		// already stopped
	default:
		// done will be closed when the goroutine exits via ctx.Done()
	}
}
