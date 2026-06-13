package agents

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dhnnnn/forex-agent/internal/feed"
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
}

// NewMarketDataAgent membuat MarketDataAgent baru.
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

	return a
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

// ingestCandle mengkonversi OHLCVCandle → Candle dan menambahkan ke rolling buffer.
func (a *MarketDataAgent) ingestCandle(raw feed.OHLCVCandle) {
	candle := Candle{
		Pair:      raw.Pair,
		Open:      raw.Open,
		High:      raw.High,
		Low:       raw.Low,
		Close:     raw.Close,
		Volume:    raw.Volume,
		Spread:    raw.Spread,
		Timeframe: raw.Timeframe,
		Timestamp: raw.Timestamp,
	}

	key := bufferKey(raw.Pair, raw.Timeframe)

	a.mu.Lock()
	defer a.mu.Unlock()

	buf, exists := a.buffers[key]
	if !exists {
		// Auto-create buffer untuk pair/timeframe yang belum terdaftar
		buf = make([]Candle, 0, MaxBufferSize)
	}

	// Rolling buffer: buang candle tertua jika sudah penuh
	if len(buf) >= MaxBufferSize {
		buf = buf[1:]
	}

	a.buffers[key] = append(buf, candle)

	bufLen := len(a.buffers[key])
	if bufLen%10 == 0 { // Log setiap 10 candle biar tidak spam
		slog.Info("MarketDataAgent: buffer updated",
			"pair", raw.Pair,
			"timeframe", raw.Timeframe,
			"buffer_size", bufLen,
			"close", candle.Close,
		)
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
