package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/agents"
	"github.com/dhnnnn/forexAnalysis/internal/chatbot"
	"github.com/dhnnnn/forexAnalysis/internal/config"
	"github.com/dhnnnn/forexAnalysis/internal/feed"
	"github.com/dhnnnn/forexAnalysis/internal/sentiment"
	"github.com/dhnnnn/forexAnalysis/internal/storage"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
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

	// ── Initialize Storage (TimescaleDB) ──────────────────────────────
	store, err := storage.New(ctx, cfg.TimescaleDB.DSN)
	if err != nil {
		slog.Warn("⚠️ TimescaleDB not available — signals won't be persisted", "error", err)
		store = nil
	} else {
		defer store.Close()
	}

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

	// ── Initialize Agent 6: WhatsAppAgent ─────────────────────────────
	whatsAppAgent := agents.NewWhatsAppAgent(agents.WhatsAppConfig{
		ServiceURL:           cfg.WhatsApp.ServiceURL,
		TargetPhone:          cfg.WhatsApp.TargetPhone,
		MinConfidenceToAlert: cfg.WhatsApp.MinConfidenceToAlert,
		RateLimitSeconds:     cfg.WhatsApp.RateLimitSeconds,
	})
	slog.Info("Agent initialized", "agent", whatsAppAgent.Name(),
		"service_url", cfg.WhatsApp.ServiceURL,
		"rate_limit", fmt.Sprintf("%ds", cfg.WhatsApp.RateLimitSeconds),
	)

	// ── Start collecting data in background ───────────────────────────
	marketAgent.StartCollecting(ctx)
	slog.Info("MarketDataAgent: collecting candles in background...")

	// ── Initialize ChatBot Handler ────────────────────────────────────
	chatHandler := chatbot.NewHandler()
	chatHandler.UpdateFromConfig(cfg.Account.Balance, cfg.Account.RiskPercent)
	chatHandler.SetPairs(cfg.Pairs)
	chatHandler.SetStatusFunc(func() string {
		// Cek apakah ada pair yang sudah ready
		for _, pair := range cfg.Pairs {
			bufSize := marketAgent.BufferSize(pair, cfg.Scheduler.Timeframes[0])
			if bufSize >= agents.MinCandlesRequired {
				return "🟢 Running — pipeline active"
			}
		}
		return "🟡 Warming up — collecting candle data"
	})

	// Initialize AI Chat (Gemini primary + Groq fallback)
	geminiChat := chatbot.NewGeminiChat(cfg.Gemini.APIKey, cfg.Gemini.Model, geminiTimeout)
	if cfg.Groq.APIKey != "" {
		geminiChat.SetGroqFallback(cfg.Groq.APIKey, cfg.Groq.Model)
		slog.Info("Groq fallback configured", "model", cfg.Groq.Model)
	}
	chatHandler.SetGeminiChat(geminiChat)
	slog.Info("AI Chat initialized", "primary", cfg.Gemini.Model, "fallback", cfg.Groq.Model)

	// ── Start HTTP server for chat ────────────────────────────────────
	mux := http.NewServeMux()
	mux.Handle("/chat", chatHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		slog.Info("HTTP server starting", "addr", ":8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// ── Pipeline loop (concurrent execution) ──────────────────────────
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Cek setiap 5 menit (hemat API quota)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Process all pairs concurrently
				var wg sync.WaitGroup
				for _, pair := range cfg.Pairs {
					wg.Add(1)
					go func(pair string) {
						defer wg.Done()
						runPipeline(ctx, pair, cfg, marketAgent, technicalAgent, fundamentalAgent,
							riskAgent, decisionAgent, whatsAppAgent, chatHandler, store)
					}(pair)
				}
				wg.Wait()
			}
		}
	}()

	// ── Wait for shutdown signal ──────────────────────────────────────
	sig := <-sigChan
	slog.Info("Shutdown signal received", "signal", sig)
	cancel()

	// Gracefully shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	httpServer.Shutdown(shutdownCtx)

	// Give goroutines time to cleanup
	time.Sleep(1 * time.Second)
	slog.Info("🛑 Forex Multi-Agent Bot stopped. Bye!")
}

// ════════════════════════════════════════════════════════════════════════════════
// runPipeline — proses satu pair dengan concurrent TechnicalAgent + FundamentalAgent
// ════════════════════════════════════════════════════════════════════════════════

func runPipeline(
	ctx context.Context,
	pair string,
	cfg *config.Config,
	marketAgent *agents.MarketDataAgent,
	technicalAgent *agents.TechnicalAgent,
	fundamentalAgent *agents.FundamentalAgent,
	riskAgent *agents.RiskAgent,
	decisionAgent *agents.DecisionAgent,
	whatsAppAgent *agents.WhatsAppAgent,
	chatHandler *chatbot.Handler,
	store *storage.Store,
) {
	// Agent 1: Check if MarketDataAgent has enough data
	output := marketAgent.Run(ctx, agents.AgentInput{Pair: pair})
	if !output.Success {
		bufSize := marketAgent.BufferSize(pair, cfg.Scheduler.Timeframes[0])
		slog.Debug("⏳ MarketDataAgent collecting...",
			"pair", pair,
			"buffer", fmt.Sprintf("%d/%d", bufSize, agents.MinCandlesRequired),
		)
		return
	}

	candles := marketAgent.GetCandles(pair, cfg.Scheduler.Timeframes[0])

	// Persist candles to TimescaleDB (non-blocking, best-effort)
	if store != nil {
		go func() {
			if err := store.InsertCandles(ctx, candles); err != nil {
				slog.Debug("⚠️ Failed to persist candles", "pair", pair, "error", err)
			}
		}()
	}

	// ── Agent 2 + 3: Technical & Fundamental Analysis (CONCURRENT) ────
	var techOutput, fundOutput agents.AgentOutput

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		techOutput = technicalAgent.Run(gCtx, agents.AgentInput{
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
			slog.Debug("⚠️ TechnicalAgent failed", "pair", pair, "error", techOutput.Error)
		}
		return nil // non-fatal: pipeline tetap jalan meskipun satu agent gagal
	})

	g.Go(func() error {
		fundOutput = fundamentalAgent.Run(gCtx, agents.AgentInput{Pair: pair})
		if fundOutput.Success && fundOutput.Fundamental != nil {
			slog.Debug("✅ FundamentalAgent completed",
				"pair", pair,
				"sentiment", fundOutput.Fundamental.Sentiment,
				"confidence", fmt.Sprintf("%.2f", fundOutput.Fundamental.Confidence),
				"score", fmt.Sprintf("%.3f", fundOutput.Fundamental.Score),
				"from_cache", fundOutput.Fundamental.FromCache,
			)
		} else {
			slog.Debug("⚠️ FundamentalAgent failed", "pair", pair, "error", fundOutput.Error)
		}
		return nil
	})

	// Wait for both agents to complete
	_ = g.Wait()

	// ── Agent 4: Risk Management (needs technical signal) ─────────────
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
		slog.Debug("⚠️ RiskAgent failed", "pair", pair, "error", riskOutput.Error)
	}

	// ── Agent 5: Decision (aggregate all) ─────────────────────────────
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

	// Log final pipeline result
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

		// Update chatbot with latest signal
		chatHandler.SetLastSignal(fmt.Sprintf("%s %s %d%%", d.Signal, pair, d.ConfPct))

		// Persist signal to TimescaleDB
		if store != nil {
			if err := store.InsertSignal(ctx, d); err != nil {
				slog.Debug("⚠️ Failed to persist signal", "pair", pair, "error", err)
			}
		}

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

		// ── Agent 6: WhatsApp Notification ────────────────────────────
		waInput := agents.AgentInput{
			Pair:     pair,
			Decision: decisionOutput.Decision,
		}
		waOutput := whatsAppAgent.Run(ctx, waInput)
		if !waOutput.Success {
			slog.Debug("⚠️ WhatsAppAgent failed", "pair", pair, "error", waOutput.Error)
		}
	} else {
		slog.Warn("⚠️ DecisionAgent failed", "pair", pair, "error", decisionOutput.Error)
	}
}
