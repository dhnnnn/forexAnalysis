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
	"github.com/dhnnnn/forex-agent/internal/sentiment"
	"github.com/redis/go-redis/v9"
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

	// ── Inisialisasi Agent 3: FundamentalAgent ────────────────────────

	// Redis client
	redisAddr := os.Getenv("REDIS_ADDRESS")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	// NewsFetcher with API keys
	alphaVantageKey := os.Getenv("ALPHA_VANTAGE_KEY")
	twelveDataKey := os.Getenv("TWELVE_DATA_KEY")
	rssURLs := []string{
		"https://www.forexfactory.com/rss",
	}
	newsFetcher := sentiment.NewNewsFetcher(alphaVantageKey, twelveDataKey, rssURLs)

	// GeminiClient with API key, model, and 2s timeout
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-1.5-flash"
	}
	geminiClient := sentiment.NewGeminiClient(geminiAPIKey, geminiModel, 2*time.Second)

	// SentimentCache with 5-minute TTL
	sentimentCache := sentiment.NewSentimentCache(redisClient, 5*time.Minute)

	// FundamentalAgent with all dependencies injected
	fundamentalAgent := agents.NewFundamentalAgent(geminiClient, newsFetcher, sentimentCache)

	slog.Info("Agent initialized",
		"agent", fundamentalAgent.Name(),
		"gemini_model", geminiModel,
		"cache_ttl", "5m",
	)

	// ── Inisialisasi Agent 5: DecisionAgent ───────────────────────────
	signalCfg := agents.SignalConfig{
		BuyThreshold:  0.65,
		SellThreshold: 0.35,
		TechWeight:    0.60,
		FundWeight:    0.40,
		MLBoostWeight: 0.20,
	}

	// ML service disabled → pass nil for mlClient
	decisionAgent := agents.NewDecisionAgent(signalCfg, nil)

	slog.Info("Agent initialized", "agent", decisionAgent.Name())

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

						// Run FundamentalAgent for sentiment analysis
						fundOutput := fundamentalAgent.Run(ctx, agents.AgentInput{Pair: pair})
						if fundOutput.Success && fundOutput.Fundamental != nil {
							slog.Info("✅ FundamentalAgent completed",
								"pair", pair,
								"sentiment", fundOutput.Fundamental.Sentiment,
								"confidence", fmt.Sprintf("%.2f", fundOutput.Fundamental.Confidence),
								"score", fmt.Sprintf("%.3f", fundOutput.Fundamental.Score),
								"from_cache", fundOutput.Fundamental.FromCache,
							)
						} else {
							slog.Warn("⚠️ FundamentalAgent failed",
								"pair", pair,
								"error", fundOutput.Error,
							)
						}

						// Run DecisionAgent for final trading signal
						decisionInput := agents.AgentInput{
							Pair:        pair,
							Candles:     candles,
							Technical:   nil, // TechnicalAgent belum aktif
							Fundamental: fundOutput.Fundamental,
							Risk:        nil, // RiskAgent belum aktif
						}
						decisionOutput := decisionAgent.Run(ctx, decisionInput)
						if decisionOutput.Success && decisionOutput.Decision != nil {
							slog.Info("✅ DecisionAgent completed",
								"pair", pair,
								"signal", decisionOutput.Decision.Signal,
								"confidence", fmt.Sprintf("%.2f", decisionOutput.Decision.Confidence),
								"risk_level", decisionOutput.Decision.RiskLevel,
							)
						} else {
							slog.Warn("⚠️ DecisionAgent failed",
								"pair", pair,
								"error", decisionOutput.Error,
							)
						}
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
