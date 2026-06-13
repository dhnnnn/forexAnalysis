package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dhnnnn/forex-agent/internal/agents"
	"github.com/dhnnnn/forex-agent/internal/feed"
)

func main() {
	// ── Setup structured logging ──────────────────────────────────────
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("═══════════════════════════════════════════════")
	slog.Info("  🤖 Forex Multi-Agent Bot — Starting...")
	slog.Info("═══════════════════════════════════════════════")

	// ── Konfigurasi (hardcoded dulu, nanti pindah ke config.yaml) ────
	pairs := []string{"EUR_USD", "GBP_USD"}
	timeframes := []string{"1h"}

	// ── Context dengan graceful shutdown ──────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Tangkap sinyal OS untuk graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ── Inisialisasi Feed Layer ───────────────────────────────────────
	wsFeed := feed.NewWebSocketFeed(
		"wss://stream-fxtrade.oanda.com", // URL (mock mode dulu)
		"",                                // API key (kosong = mock)
		pairs,
	)

	restPoller := feed.NewRESTPoller(
		"https://api.twelvedata.com",
		"", // API key (kosong = belum dipakai)
		pairs,
	)

	// ── Inisialisasi Agent 1: MarketDataAgent ─────────────────────────
	marketAgent := agents.NewMarketDataAgent(pairs, timeframes, wsFeed, restPoller)

	slog.Info("Agent initialized",
		"agent", marketAgent.Name(),
		"pairs", pairs,
		"timeframes", timeframes,
	)

	// ── Mulai collecting data di background ──────────────────────────
	marketAgent.StartCollecting(ctx)
	slog.Info("MarketDataAgent: collecting candles in background...")

	// ── Pipeline loop (cek readiness setiap 10 detik) ────────────────
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, pair := range pairs {
					// Cek apakah Agent 1 sudah punya cukup data
					output := marketAgent.Run(ctx, agents.AgentInput{Pair: pair})

					if output.Success {
						candles := marketAgent.GetCandles(pair, timeframes[0])
						latest := candles[len(candles)-1]
						slog.Info("✅ MarketDataAgent READY — pipeline bisa dimulai",
							"pair", pair,
							"candles", len(candles),
							"latest_close", fmt.Sprintf("%.5f", latest.Close),
							"latest_time", latest.Timestamp.Format("15:04:05"),
						)
						// TODO: di sini nanti panggil Agent 2 (TechnicalAgent), dst.
					} else {
						bufSize := marketAgent.BufferSize(pair, timeframes[0])
						slog.Info("⏳ MarketDataAgent collecting...",
							"pair", pair,
							"buffer", fmt.Sprintf("%d/%d", bufSize, agents.MinCandlesRequired),
						)
					}
				}
			}
		}
	}()

	// ── Tunggu shutdown signal ────────────────────────────────────────
	sig := <-sigChan
	slog.Info("Shutdown signal received", "signal", sig)
	cancel()

	// Beri waktu goroutine untuk cleanup
	time.Sleep(1 * time.Second)
	slog.Info("🛑 Forex Multi-Agent Bot stopped. Bye!")
}
