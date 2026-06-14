package agents

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/feed"
)

const (
	// MaxBufferSize adalah jumlah maksimum candle dalam rolling buffer per pair.
	MaxBufferSize = 200

	// MinCandlesRequired adalah jumlah minimum candle yang dibutuhkan
	// sebelum agent lain bisa mulai analisa (MACD butuh minimal 26).
	MinCandlesRequired = 26
)

// MarketDataAgent (Agent 1) bertanggung jawab mengambil data OHLCV
// dari sumber eksternal, memvalidasi, dan menyimpan di rolling buffer.
type MarketDataAgent struct {
	pairs      []string
	timeframes []string
	buffers    map[string][]Candle // key: "PAIR:TIMEFRAME" → rolling buffer
	mu         sync.RWMutex

	wsFeed     *feed.WebSocketFeed
	restPoller *feed.RESTPoller

	onCandleIngested func(Candle) // Callback for real-time publishing
}

// NewMarketDataAgent membuat MarketDataAgent baru dengan pre-population data historis.
func NewMarketDataAgent(
	pairs, timeframes []string,
	wsFeed *feed.WebSocketFeed,
	restPoller *feed.RESTPoller,
) *MarketDataAgent {
	a := &MarketDataAgent{
		pairs:      pairs,
		timeframes: timeframes,
		buffers:    make(map[string][]Candle),
		wsFeed:     wsFeed,
		restPoller: restPoller,
	}

	// Inisialisasi buffer kosong per pair:timeframe
	for _, p := range pairs {
		for _, tf := range timeframes {
			key := bufferKey(p, tf)
			a.buffers[key] = make([]Candle, 0, MaxBufferSize)
		}
	}

	// Pre-populate dengan historical mock candles agar sistem bisa langsung berjalan
	basePrice := map[string]float64{
		"EUR_USD": 1.08450,
		"GBP_USD": 1.27230,
		"USD_JPY": 149.850,
		"AUD_USD": 0.66780,
	}

	now := time.Now()
	for _, p := range pairs {
		price, ok := basePrice[p]
		if !ok {
			price = 1.00000
		}
		for _, tf := range timeframes {
			key := bufferKey(p, tf)
			duration := parseTimeframeDuration(tf)

			// Generate 100 historical candles
			for i := 100; i >= 1; i-- {
				t := now.Add(-time.Duration(i) * duration).Truncate(duration)
				seconds := float64(t.Unix())
				trend := math.Sin(seconds/150.0) * 0.00080
				// Use i instead of t.UnixNano() since truncated timestamps always yield a modulo 100 of 0.
				noise := float64((i*17)%100-50) * 0.00002

				openPrice := price + trend - noise
				closePrice := price + trend + noise
				high := math.Max(openPrice, closePrice) + 0.00020
				low := math.Min(openPrice, closePrice) - 0.00015

				a.buffers[key] = append(a.buffers[key], Candle{
					Pair:      p,
					Open:      openPrice,
					High:      high,
					Low:       low,
					Close:     closePrice,
					Volume:    float64(t.Unix() % 10000),
					Spread:    1.2,
					Timeframe: tf,
					Timestamp: t,
				})
			}
		}
	}

	return a
}

// SetOnCandleIngested mendaftarkan callback untuk menerima candle baru yang di-ingest.
func (a *MarketDataAgent) SetOnCandleIngested(cb func(Candle)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onCandleIngested = cb
}

// Name mengembalikan nama agent.
func (a *MarketDataAgent) Name() string { return "MarketDataAgent" }

// Run mengecek apakah buffer sudah cukup data untuk pair & timeframe tertentu.
// Agent ini tidak fetch data di Run() — data dikumpulkan via StartCollecting().
func (a *MarketDataAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	key := bufferKey(input.Pair, a.defaultTimeframe())

	a.mu.RLock()
	candles := a.buffers[key]
	count := len(candles)
	a.mu.RUnlock()

	if count < MinCandlesRequired {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   false,
			Error: fmt.Errorf("insufficient candle data for %s: need min %d, have %d",
				input.Pair, MinCandlesRequired, count),
			Timestamp: time.Now(),
		}
	}

	slog.Info("MarketDataAgent ready",
		"pair", input.Pair,
		"candles", count,
		"latest_close", candles[count-1].Close,
		"latest_time", candles[count-1].Timestamp.Format("15:04:05"),
	)

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Timestamp: time.Now(),
	}
}

// ════════════════════════════════════════════════════════════════════════
// Data Collection — berjalan di background goroutine
// ════════════════════════════════════════════════════════════════════════

// StartCollecting memulai goroutine background yang mendengarkan
// candle dari WebSocket feed dan menyimpannya ke rolling buffer.
func (a *MarketDataAgent) StartCollecting(ctx context.Context) {
	if a.wsFeed == nil {
		slog.Warn("MarketDataAgent: no WebSocket feed configured, skipping collection")
		return
	}

	// Start WebSocket feed
	a.wsFeed.Start(ctx)

	// Goroutine consumer: baca dari feed channel, simpan ke buffer
	go func() {
		slog.Info("MarketDataAgent: collecting candles from feed...",
			"pairs", a.pairs)

		for {
			select {
			case <-ctx.Done():
				slog.Info("MarketDataAgent: collection stopped")
				return
			case raw, ok := <-a.wsFeed.Output:
				if !ok {
					slog.Info("MarketDataAgent: feed channel closed")
					return
				}
				a.ingestCandle(raw)
			}
		}
	}()
}

// ingestCandle mereplikasi dan mengupdate/menambahkan candle ke semua buffer timeframe yang terdaftar.
func (a *MarketDataAgent) ingestCandle(raw feed.OHLCVCandle) {
	// Replikasi ticks ke semua timeframe yang terdaftar
	for _, tf := range a.timeframes {
		duration := parseTimeframeDuration(tf)
		truncatedTime := raw.Timestamp.Truncate(duration)
		key := bufferKey(raw.Pair, tf)

		a.mu.Lock()
		buf := a.buffers[key]
		var updated Candle

		if len(buf) > 0 && buf[len(buf)-1].Timestamp.Equal(truncatedTime) {
			// Update candle yang sedang berjalan (current period)
			last := &buf[len(buf)-1]
			last.Close = raw.Close
			if raw.High > last.High {
				last.High = raw.High
			}
			if raw.Low < last.Low {
				last.Low = raw.Low
			}
			last.Volume += raw.Volume
			last.Spread = raw.Spread
			updated = *last
		} else {
			// Buat candle baru untuk period berikutnya
			candle := Candle{
				Pair:      raw.Pair,
				Open:      raw.Open,
				High:      raw.High,
				Low:       raw.Low,
				Close:     raw.Close,
				Volume:    raw.Volume,
				Spread:    raw.Spread,
				Timeframe: tf,
				Timestamp: truncatedTime,
			}
			if len(buf) >= MaxBufferSize {
				buf = buf[1:]
			}
			a.buffers[key] = append(buf, candle)
			updated = candle
		}

		cb := a.onCandleIngested
		a.mu.Unlock()

		// Trigger callback jika terdaftar untuk real-time update
		if cb != nil {
			cb(updated)
		}
	}
}

// ════════════════════════════════════════════════════════════════════════
// Public Getters
// ════════════════════════════════════════════════════════════════════════

// GetCandles mengembalikan salinan rolling buffer untuk pair + timeframe tertentu.
func (a *MarketDataAgent) GetCandles(pair, timeframe string) []Candle {
	key := bufferKey(pair, timeframe)

	a.mu.RLock()
	defer a.mu.RUnlock()

	buf := a.buffers[key]
	if len(buf) == 0 {
		return nil
	}

	// Return copy agar caller tidak bisa modify internal buffer
	result := make([]Candle, len(buf))
	copy(result, buf)
	return result
}

// GetLatestCandle mengembalikan candle terakhir dari buffer.
func (a *MarketDataAgent) GetLatestCandle(pair, timeframe string) *Candle {
	key := bufferKey(pair, timeframe)

	a.mu.RLock()
	defer a.mu.RUnlock()

	buf := a.buffers[key]
	if len(buf) == 0 {
		return nil
	}

	c := buf[len(buf)-1]
	return &c
}

// BufferSize mengembalikan jumlah candle dalam buffer.
func (a *MarketDataAgent) BufferSize(pair, timeframe string) int {
	key := bufferKey(pair, timeframe)

	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.buffers[key])
}

// defaultTimeframe mengembalikan timeframe default (pertama dari daftar).
func (a *MarketDataAgent) defaultTimeframe() string {
	if len(a.timeframes) > 0 {
		return a.timeframes[0]
	}
	return "1h"
}

// bufferKey menghasilkan key unik untuk buffer map.
func bufferKey(pair, timeframe string) string {
	return pair + ":" + timeframe
}

// parseTimeframeDuration mengonversi string timeframe ke time.Duration.
func parseTimeframeDuration(tf string) time.Duration {
	switch tf {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	default:
		return time.Hour
	}
}
