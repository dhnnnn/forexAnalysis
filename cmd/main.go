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
	"github.com/dhnnnn/forex-agent/internal/config"
	"github.com/dhnnnn/forex-agent/internal/feed"
	"github.com/dhnnnn/forex-agent/internal/sentiment"
	"github.com/redis/go-redis/v9"
)

func main() {
	// ── Setup structured logging ──────────────────────────────────────
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("═══════════════════════════════════════════════")
	slog.Info("  🤖 Forex Multi-Agent Bot — Starting...")
	slog.Info("═══════════════════════════════════════════════")

	// ── Load configuration from YAML ──────────────────────────────────
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("Config loaded", "pairs", cfg.Pairs, "timeframes", cfg.Scheduler.Timeframes)

	// ── Context with graceful shutdown ────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// ── Initialize Feed Layer ─────────────────────────────────────────
	wsFeed := feed.NewWebSocketFeed(
		cfg.Oanda.WebSocketURL,
		cfg.Oanda.APIKey,
		cfg.Pairs,
	)

	restPoller := feed.NewRESTPoller(
		cfg.TwelveData.BaseURL,
		cfg.TwelveData.APIKey,
		cfg.Pairs,
	)

	// ── Initialize Agent 1: MarketDataAgent ───────────────────────────
	marketAgent := agents.NewMarketDataAgent(cfg.Pairs, cfg.Scheduler.Timeframes, wsFeed, restPoller)
	slog.Info("Agent initialized",
		"agent", marketAgent.Name(),
		"pairs", cfg.Pairs,
		"timeframes", cfg.Scheduler.Timeframes,
	)

	// ── Initialize Agent 2: TechnicalAgent ────────────────────────────
	technicalAgent := agents.NewTechnicalAgent()
	slog.Info("Agent initialized", "agent", technicalAgent.Name())

	// ── Initialize Agent 3: FundamentalAgent ──────────────────────────
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
	})

	newsFetcher := sentiment.NewNewsFetcher(
		cfg.AlphaVantage.APIKey,
		cfg.TwelveData.APIKey,
		cfg.RSSFeeds.URLs,
	)

	geminiTimeout := time.Duration(cfg.Gemini.TimeoutMs) * time.Millisecond
	geminiClient := sentiment.NewGeminiClient(cfg.Gemini.APIKey, cfg.Gemini.Model, geminiTimeout)

	sentimentTTL := time.Duration(cfg.Redis.SentimentTTLMin) * time.Minute
	sentimentCache := sentiment.NewSentimentCache(redisClient, sentimentTTL)

	fundamentalAgent := agents.NewFundamentalAgent(geminiClient, newsFetcher, sentimentCache)
	slog.Info("Agent initialized",
		"agent", fundamentalAgent.Name(),
		"gemini_model", cfg.Gemini.Model,
		"cache_ttl", sentimentTTL.String(),
	)

	// ── Initialize Agent 4: RiskAgent ─────────────────────────────────
	riskAgent := agents.NewRiskAgent()
	slog.Info("Agent initialized", "agent", riskAgent.Name())

	// ── Initialize Agent 5: DecisionAgent ─────────────────────────────
	signalCfg := agents.SignalConfig{
		BuyThreshold:  cfg.Signal.BuyThreshold,
		SellThreshold: cfg.Signal.SellThreshold,
		TechWeight:    cfg.Signal.Weights.Technical,
		FundWeight:    cfg.Signal.Weights.Fundamental,
		MLBoostWeight: cfg.Signal.MLBoostWeight,
	}

	// ML service disabled → pass nil for mlClient
	decisionAgent := agents.NewDecisionAgent(signalCfg, nil)
	slog.Info("Agent initialized", "agent", decisionAgent.Name())

	// ── Start collecting data in background ───────────────────────────
	marketAgent.StartCollecting(ctx)
	slog.Info("MarketDataAgent: collecting candles in background...")

	// ── Pipeline loop (check readiness every 10 seconds) ──────────────
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, pair := range cfg.Pairs {
					// Agent 1: Check if MarketDataAgent has enough data
					output := marketAgent.Run(ctx, agents.AgentInput{Pair: pair})
					if !output.Success {
						bufSize := marketAgent.BufferSize(pair, cfg.Scheduler.Timeframes[0])
						slog.Debug("⏳ MarketDataAgent collecting...",
							"pair", pair,
							"buffer", fmt.Sprintf("%d/%d", bufSize, agents.MinCandlesRequired),
						)
						continue
					}

					candles := marketAgent.GetCandles(pair, cfg.Scheduler.Timeframes[0])

					// Agent 2: Technical Analysis
					techOutput := technicalAgent.Run(ctx, agents.AgentInput{
						Pair:    pair,
						Candles: candles,
					})
					if techOutput.Success && techOutput.Technical != nil {
						slog.Debug("✅ TechnicalAgent completed",
							"pair", pair,
							"signal", techOutput.Technical.Signal,
							"confidence", fmt.Sprintf("%.2f", techOutput.Technical.Confidence),
							"tech_score", fmt.Sprintf("%.3f", techOutput.Technical.TechScore),
						)
					} else {
						slog.Debug("⚠️ TechnicalAgent failed",
							"pair", pair,
							"error", techOutput.Error,
						)
					}

					// Agent 3: Fundamental Analysis
					fundOutput := fundamentalAgent.Run(ctx, agents.AgentInput{Pair: pair})
					if fundOutput.Success && fundOutput.Fundamental != nil {
						slog.Debug("✅ FundamentalAgent completed",
							"pair", pair,
							"sentiment", fundOutput.Fundamental.Sentiment,
							"confidence", fmt.Sprintf("%.2f", fundOutput.Fundamental.Confidence),
							"score", fmt.Sprintf("%.3f", fundOutput.Fundamental.Score),
							"from_cache", fundOutput.Fundamental.FromCache,
						)
					} else {
						slog.Debug("⚠️ FundamentalAgent failed",
							"pair", pair,
							"error", fundOutput.Error,
						)
					}

					// Agent 4: Risk Management (needs technical signal)
					riskInput := agents.AgentInput{
						Pair:           pair,
						Candles:        candles,
						Technical:      techOutput.Technical,
						AccountBalance: cfg.Account.Balance,
						RiskPercent:    cfg.Account.RiskPercent,
					}
					riskOutput := riskAgent.Run(ctx, riskInput)
					if riskOutput.Success && riskOutput.Risk != nil {
						slog.Debug("✅ RiskAgent completed",
							"pair", pair,
							"lot_size", fmt.Sprintf("%.2f", riskOutput.Risk.LotSize),
							"sl", fmt.Sprintf("%.5f", riskOutput.Risk.StopLoss),
							"tp", fmt.Sprintf("%.5f", riskOutput.Risk.TakeProfit),
						)
					} else {
						slog.Debug("⚠️ RiskAgent failed",
							"pair", pair,
							"error", riskOutput.Error,
						)
					}

					// Agent 5: Decision (aggregate all)
					decisionInput := agents.AgentInput{
						Pair:           pair,
						Candles:        candles,
						Technical:      techOutput.Technical,
						Fundamental:    fundOutput.Fundamental,
						Risk:           riskOutput.Risk,
						AccountBalance: cfg.Account.Balance,
						RiskPercent:    cfg.Account.RiskPercent,
					}
					decisionOutput := decisionAgent.Run(ctx, decisionInput)

					// Log final pipeline result at Info level
					if decisionOutput.Success && decisionOutput.Decision != nil {
						d := decisionOutput.Decision
						slog.Info("📊 Pipeline completed",
							"pair", pair,
							"signal", d.Signal,
							"confidence", fmt.Sprintf("%.0f%%", d.Confidence*100),
							"risk_level", d.RiskLevel,
							"tech_signal", d.TechSignal,
							"fund_sentiment", d.FundSentiment,
						)
						if d.Signal != "HOLD" {
							slog.Info("💰 Trade Signal",
								"pair", pair,
								"signal", d.Signal,
								"entry", fmt.Sprintf("%.5f", d.Entry),
								"sl", fmt.Sprintf("%.5f", d.StopLoss),
								"tp", fmt.Sprintf("%.5f", d.TakeProfit),
								"lot", fmt.Sprintf("%.2f", d.LotSize),
							)
						}
					} else {
						slog.Warn("⚠️ DecisionAgent failed",
							"pair", pair,
							"error", decisionOutput.Error,
						)
					}
				}
			}
		}
	}()

	// ── Wait for shutdown signal ──────────────────────────────────────
	sig := <-sigChan
	slog.Info("Shutdown signal received", "signal", sig)
	cancel()

	// Give goroutines time to cleanup
	time.Sleep(1 * time.Second)
	slog.Info("🛑 Forex Multi-Agent Bot stopped. Bye!")
}
