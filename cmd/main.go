package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/agents"
	"github.com/dhnnnn/forexAnalysis/internal/chatbot"
	"github.com/dhnnnn/forexAnalysis/internal/config"
	"github.com/dhnnnn/forexAnalysis/internal/feed"
	"github.com/dhnnnn/forexAnalysis/internal/graph"
	"github.com/dhnnnn/forexAnalysis/internal/graph/model"
	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
	"github.com/dhnnnn/forexAnalysis/internal/pipeline"
	"github.com/dhnnnn/forexAnalysis/internal/sentiment"
	"github.com/dhnnnn/forexAnalysis/internal/storage"
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

	// ── Initialize Storage (TimescaleDB) ──────────────────────────────
	store, err := storage.New(ctx, cfg.TimescaleDB.DSN)
	if err != nil {
		slog.Warn("⚠️ TimescaleDB not available — signals won't be persisted", "error", err)
		store = nil
	} else {
		defer store.Close()
	}

	// ── Initialize Redis ──────────────────────────────────────────────
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
	})

	// ── Initialize Feed Layer ─────────────────────────────────────────
	wsFeed := feed.NewWebSocketFeed(cfg.Oanda.WebSocketURL, cfg.Oanda.APIKey, cfg.Pairs)
	restPoller := feed.NewRESTPoller(cfg.TwelveData.BaseURL, cfg.TwelveData.APIKey, cfg.Pairs)

	// ── Initialize All Agents ─────────────────────────────────────────
	marketAgent := agents.NewMarketDataAgent(cfg.Pairs, cfg.Scheduler.Timeframes, wsFeed, restPoller)
	technicalAgent := agents.NewTechnicalAgent()
	regimeAgent := agents.NewRegimeDetectionAgentWithConfig(agents.RegimeConfig{
		ADXPeriod:    cfg.RegimeDetect.ADXPeriod,
		ATRPeriod:    cfg.RegimeDetect.ATRPeriod,
		ADXThreshold: cfg.RegimeDetect.ADXThreshold,
		VolThreshold: cfg.RegimeDetect.VolThreshold,
	})

	geminiTimeout := time.Duration(cfg.Gemini.TimeoutMs) * time.Millisecond
	geminiClient := sentiment.NewGeminiClient(cfg.Gemini.APIKey, cfg.Gemini.Model, geminiTimeout)
	if cfg.Groq.APIKey != "" {
		geminiClient.SetGroqFallback(cfg.Groq.APIKey, cfg.Groq.Model)
	}
	sentimentTTL := time.Duration(cfg.Redis.SentimentTTLMin) * time.Minute
	sentimentCache := sentiment.NewSentimentCache(redisClient, sentimentTTL)
	newsFetcher := sentiment.NewNewsFetcher(cfg.AlphaVantage.APIKey, cfg.TwelveData.APIKey, cfg.RSSFeeds.URLs)
	fundamentalAgent := agents.NewFundamentalAgent(geminiClient, newsFetcher, sentimentCache)

	riskAgent := agents.NewRiskAgent()

	signalCfg := agents.SignalConfig{
		BuyThreshold:  cfg.Signal.BuyThreshold,
		SellThreshold: cfg.Signal.SellThreshold,
		TechWeight:    cfg.Signal.Weights.Technical,
		FundWeight:    cfg.Signal.Weights.Fundamental,
		MLBoostWeight: cfg.Signal.MLBoostWeight,
	}
	decisionAgent := agents.NewDecisionAgent(signalCfg, nil)

	whatsAppAgent := agents.NewWhatsAppAgent(agents.WhatsAppConfig{
		ServiceURL:           cfg.WhatsApp.ServiceURL,
		TargetPhone:          cfg.WhatsApp.TargetPhone,
		MinConfidenceToAlert: cfg.WhatsApp.MinConfidenceToAlert,
		RateLimitSeconds:     cfg.WhatsApp.RateLimitSeconds,
	})

	metaObserver := agents.NewMetaObserverAgentWithConfig(agents.MetaObserverConfig{
		RollingWindow:     cfg.MetaObserver.RollingWindow,
		DropThreshold:     cfg.MetaObserver.DropThreshold,
		LossStreakTrigger: cfg.MetaObserver.LossStreakTrigger,
	})
	metaObserver.RegisterAgent("TechnicalAgent")
	metaObserver.RegisterAgent("FundamentalAgent")

	// ── Initialize Knowledge System ───────────────────────────────────
	kbStore := knowledge.NewStore(redisClient)
	decisionAgent.SetKBStore(kbStore)

	broadcaster := knowledge.NewBroadcaster(kbStore)
	broadcaster.Subscribe(technicalAgent)

	ktaTimeout := time.Duration(cfg.KnowledgeTransfer.TimeoutMs) * time.Millisecond
	if ktaTimeout == 0 {
		ktaTimeout = 10 * time.Second
	}
	ktaRuleTTL := time.Duration(cfg.KnowledgeTransfer.RuleTTLHours) * time.Hour
	if ktaRuleTTL == 0 {
		ktaRuleTTL = 24 * time.Hour
	}
	ktaAgent := agents.NewKnowledgeTransferAgent(agents.KTAConfig{
		GeminiAPIKey:  cfg.Gemini.APIKey,
		GeminiModel:   cfg.Gemini.Model,
		GroqAPIKey:    cfg.Groq.APIKey,
		GroqModel:     cfg.Groq.Model,
		Timeout:       ktaTimeout,
		RuleTTL:       ktaRuleTTL,
		MinConfidence: cfg.KnowledgeTransfer.MinConfidence,
	}, kbStore)

	// ── Initialize GraphQL PubSub ─────────────────────────────────────
	gqlPubSub := graph.NewPubSub()

	// Register callback on MarketDataAgent to publish candles to GraphQL subscribers
	marketAgent.SetOnCandleIngested(func(c agents.Candle) {
		gqlPubSub.PublishCandle(c.Pair, &model.Candle{
			Pair:      c.Pair,
			Open:      c.Open,
			High:      c.High,
			Low:       c.Low,
			Close:     c.Close,
			Volume:    c.Volume,
			Spread:    c.Spread,
			Timeframe: c.Timeframe,
			Timestamp: c.Timestamp.Format(time.RFC3339),
		})
	})

	// ── Build Pipeline ────────────────────────────────────────────────
	signalStore := agents.NewSignalStore()
	evalDelay := time.Duration(cfg.MetaObserver.EvalDelayMinutes) * time.Minute

	p := pipeline.NewPipeline(pipeline.PipelineDeps{
		MarketAgent:      marketAgent,
		RegimeAgent:      regimeAgent,
		TechnicalAgent:   technicalAgent,
		FundamentalAgent: fundamentalAgent,
		RiskAgent:        riskAgent,
		DecisionAgent:    decisionAgent,
		WhatsAppAgent:    whatsAppAgent,
		MetaObserver:     metaObserver,
		KTAAgent:         ktaAgent,
		SignalStore:      signalStore,
		Broadcaster:      broadcaster,
		PubSub:           gqlPubSub,
		ChatHandler:      nil, // set below after chatHandler init
		Store:            store,
		KBStore:          kbStore,
	}, pipeline.PipelineConfig{
		EvalDelay:      evalDelay,
		PipThreshold:   cfg.MetaObserver.PipThreshold,
		Pairs:          cfg.Pairs,
		Timeframes:     cfg.Scheduler.Timeframes,
		AccountBalance: cfg.Account.Balance,
		RiskPercent:    cfg.Account.RiskPercent,
	})

	slog.Info("Pipeline built",
		"pairs", cfg.Pairs,
		"eval_delay", evalDelay.String(),
		"pip_threshold", cfg.MetaObserver.PipThreshold,
	)

	// ── Start Market Data Collection ──────────────────────────────────
	marketAgent.StartCollecting(ctx)
	slog.Info("MarketDataAgent: collecting candles in background...")

	// ── Initialize ChatBot Handler ────────────────────────────────────
	chatHandler := chatbot.NewHandler()
	chatHandler.UpdateFromConfig(cfg.Account.Balance, cfg.Account.RiskPercent)
	chatHandler.SetPairs(cfg.Pairs)
	chatHandler.SetStatusFunc(func() string {
		for _, pair := range cfg.Pairs {
			if marketAgent.BufferSize(pair, cfg.Scheduler.Timeframes[0]) >= agents.MinCandlesRequired {
				return "🟢 Running — pipeline active"
			}
		}
		return "🟡 Warming up — collecting candle data"
	})

	geminiChat := chatbot.NewGeminiChat(cfg.Gemini.APIKey, cfg.Gemini.Model, geminiTimeout)
	if cfg.Groq.APIKey != "" {
		geminiChat.SetGroqFallback(cfg.Groq.APIKey, cfg.Groq.Model)
	}
	chatHandler.SetGeminiChat(geminiChat)

	// Set chatHandler on pipeline (was nil during construction)
	p.SetChatHandler(chatHandler)

	// ── HTTP Server + GraphQL ─────────────────────────────────────────
	mux := http.NewServeMux()
	mux.Handle("/chat", chatHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	gqlResolver := &graph.Resolver{
		Store:        store,
		KBStore:      kbStore,
		MarketAgent:  marketAgent,
		MetaObserver: metaObserver,
		PubSub:       gqlPubSub,
		Pairs:        cfg.Pairs,
		Timeframes:   cfg.Scheduler.Timeframes,
	}
	mux.Handle("/graphql", graph.NewHandler(gqlResolver))

	httpServer := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		slog.Info("HTTP server starting", "addr", ":8080", "endpoints", []string{"/graphql", "/chat", "/health"})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// ── Pipeline Loop ─────────────────────────────────────────────────
	go func() {
		// Jalankan evaluasi pipeline pertama kali secara langsung agar halaman awal langsung terisi data
		slog.Info("Running initial pipeline evaluation...")
		p.RunAll(ctx)
		p.ProcessMetaObserver(ctx)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.RunAll(ctx)
				p.ProcessMetaObserver(ctx)
			}
		}
	}()

	// ── Evaluator Goroutine ───────────────────────────────────────────
	go p.StartEvaluator(ctx)

	// ── Wait for shutdown ─────────────────────────────────────────────
	sig := <-sigChan
	slog.Info("Shutdown signal received", "signal", sig)
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	time.Sleep(1 * time.Second)
	slog.Info("🛑 Forex Multi-Agent Bot stopped. Bye!")
}
