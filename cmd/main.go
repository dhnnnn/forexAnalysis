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
	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
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

	// ── Initialize RegimeDetectionAgent ───────────────────────────────
	regimeAgent := agents.NewRegimeDetectionAgentWithConfig(agents.RegimeConfig{
		ADXPeriod:    cfg.RegimeDetect.ADXPeriod,
		ATRPeriod:    cfg.RegimeDetect.ATRPeriod,
		ADXThreshold: cfg.RegimeDetect.ADXThreshold,
		VolThreshold: cfg.RegimeDetect.VolThreshold,
	})
	slog.Info("Agent initialized", "agent", regimeAgent.Name())

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

	// ── Initialize MetaObserverAgent ──────────────────────────────────
	metaObserver := agents.NewMetaObserverAgentWithConfig(agents.MetaObserverConfig{
		RollingWindow:     cfg.MetaObserver.RollingWindow,
		DropThreshold:     cfg.MetaObserver.DropThreshold,
		LossStreakTrigger: cfg.MetaObserver.LossStreakTrigger,
	})
	metaObserver.RegisterAgent("TechnicalAgent")
	metaObserver.RegisterAgent("FundamentalAgent")
	slog.Info("Agent initialized", "agent", metaObserver.Name(),
		"rolling_window", cfg.MetaObserver.RollingWindow,
		"drop_threshold", cfg.MetaObserver.DropThreshold,
		"loss_streak_trigger", cfg.MetaObserver.LossStreakTrigger,
	)

	// ── Initialize Knowledge Store (Redis) ────────────────────────────
	kbStore := knowledge.NewStore(redisClient)
	slog.Info("Knowledge Store initialized (Redis)")

	// Hook KB Store ke DecisionAgent untuk adaptive weights
	decisionAgent.SetKBStore(kbStore)

	// ── Initialize Broadcaster ────────────────────────────────────────
	broadcaster := knowledge.NewBroadcaster(kbStore)
	broadcaster.Subscribe(technicalAgent)
	slog.Info("Broadcaster initialized", "subscribers", broadcaster.SubscriberCount())

	// ── Initialize Signal Store (untuk evaluator) ─────────────────────
	signalStore := agents.NewSignalStore()
	evalDelay := time.Duration(cfg.MetaObserver.EvalDelayMinutes) * time.Minute
	pipThreshold := cfg.MetaObserver.PipThreshold
	slog.Info("Signal evaluator configured",
		"eval_delay", evalDelay.String(),
		"pip_threshold", pipThreshold,
	)

	// ── Initialize KnowledgeTransferAgent ─────────────────────────────
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
	slog.Info("Agent initialized", "agent", ktaAgent.Name(),
		"rule_ttl", ktaRuleTTL.String(),
		"min_confidence", cfg.KnowledgeTransfer.MinConfidence,
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
						runPipeline(ctx, pair, cfg, marketAgent, regimeAgent, technicalAgent, fundamentalAgent,
							riskAgent, decisionAgent, whatsAppAgent, metaObserver, signalStore, evalDelay, broadcaster, chatHandler, store)
					}(pair)
				}
				wg.Wait()

				// Setelah pipeline selesai, cek MetaObserver untuk ExperienceReport
				reports := metaObserver.Observe()
				if len(reports) > 0 {
					slog.Info("🚨 MetaObserver detected degradation", "report_count", len(reports))

					// Persist ExperienceReports ke Postgres
					if store != nil {
						go func() {
							if err := store.InsertExperienceReports(ctx, reports); err != nil {
								slog.Debug("⚠️ Failed to persist experience reports", "error", err)
							}
						}()
					}

					// KnowledgeTransferAgent: proses reports → KnowledgeRules
					newRules := ktaAgent.Process(ctx, reports)
					if len(newRules) > 0 {
						slog.Info("✨ KTA generated new rules",
							"rule_count", len(newRules),
						)

						// Persist rules ke Postgres
						if store != nil {
							go func() {
								if err := store.InsertKnowledgeRules(ctx, newRules); err != nil {
									slog.Debug("⚠️ Failed to persist knowledge rules", "error", err)
								}
							}()
						}
					}
				}

				// Persist metrics ke Redis
				metrics := metaObserver.GetMetrics()
				if err := kbStore.SaveAllMetrics(ctx, metrics); err != nil {
					slog.Debug("⚠️ Failed to save metrics to Redis", "error", err)
				}
			}
		}
	}()

	// ── Evaluator Goroutine — evaluasi sinyal setelah delay ───────────
	go func() {
		evalTicker := time.NewTicker(1 * time.Minute) // cek setiap menit apakah ada sinyal ready
		defer evalTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-evalTicker.C:
				readySignals := signalStore.GetReadyForEvaluation()
				if len(readySignals) == 0 {
					continue
				}

				for _, sig := range readySignals {
					correct, pipsMove, evalPrice := evaluateSignalFull(sig, marketAgent, cfg.Scheduler.Timeframes[0], pipThreshold)

					// Record outcome ke KEDUA agen (Technical + Fundamental)
					for _, agentName := range []string{"TechnicalAgent", "FundamentalAgent"} {
						metaObserver.RecordOutcome(agents.SignalOutcome{
							AgentName: agentName,
							Pair:      sig.Pair,
							Correct:   correct,
							Regime:    sig.Regime,
							Timestamp: time.Now(),
						})
					}

					// Persist ke Postgres (non-blocking)
					if store != nil {
						go func(s agents.PendingSignal, c bool, pm float64, ep float64) {
							entry := storage.PerformanceLogEntry{
								AgentName:  "TechnicalAgent",
								Pair:       s.Pair,
								Regime:     string(s.Regime),
								Signal:     s.Signal,
								EntryPrice: s.Entry,
								EvalPrice:  ep,
								Correct:    c,
								PipsMove:   pm,
								SignalTime:  s.CreatedAt,
								EvalTime:   time.Now(),
							}
							if err := store.InsertPerformanceLog(ctx, entry); err != nil {
								slog.Debug("⚠️ Failed to persist performance log", "error", err)
							}
						}(sig, correct, pipsMove, evalPrice)
					}

					slog.Debug("📋 Signal evaluated",
						"pair", sig.Pair,
						"signal", sig.Signal,
						"entry", fmt.Sprintf("%.5f", sig.Entry),
						"eval_price", fmt.Sprintf("%.5f", evalPrice),
						"pips_move", fmt.Sprintf("%.1f", pipsMove),
						"correct", correct,
						"regime", string(sig.Regime),
					)
				}
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
	regimeAgent *agents.RegimeDetectionAgent,
	technicalAgent *agents.TechnicalAgent,
	fundamentalAgent *agents.FundamentalAgent,
	riskAgent *agents.RiskAgent,
	decisionAgent *agents.DecisionAgent,
	whatsAppAgent *agents.WhatsAppAgent,
	metaObserver *agents.MetaObserverAgent,
	signalStore *agents.SignalStore,
	evalDelay time.Duration,
	broadcaster *knowledge.Broadcaster,
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

	// ── Regime Detection: klasifikasi kondisi pasar ───────────────────
	regimeCtx := regimeAgent.Detect(ctx, pair, candles)
	slog.Debug("🔍 RegimeDetection completed",
		"pair", pair,
		"regime", string(regimeCtx.Regime),
		"adx", fmt.Sprintf("%.2f", regimeCtx.ADX),
		"atr", fmt.Sprintf("%.6f", regimeCtx.ATR),
		"volatility", fmt.Sprintf("%.4f", regimeCtx.Volatility),
		"trend_strength", fmt.Sprintf("%.2f", regimeCtx.TrendStrength),
	)

	// Persist regime log ke Postgres (non-blocking)
	if store != nil {
		go func() {
			if err := store.InsertRegimeLog(ctx, storage.RegimeLogEntry{
				Pair:          pair,
				Regime:        string(regimeCtx.Regime),
				ADX:           regimeCtx.ADX,
				ATR:           regimeCtx.ATR,
				Volatility:    regimeCtx.Volatility,
				TrendStrength: regimeCtx.TrendStrength,
				DetectedAt:    regimeCtx.DetectedAt,
			}); err != nil {
				slog.Debug("⚠️ Failed to persist regime log", "pair", pair, "error", err)
			}
		}()
	}

	// ── Broadcast KnowledgeRules ke semua subscriber agents ───────────
	broadcaster.Broadcast(ctx, regimeCtx)

	// ── Agent 2 + 3: Technical & Fundamental Analysis (CONCURRENT) ────
	var techOutput, fundOutput agents.AgentOutput

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		techOutput = technicalAgent.Run(gCtx, agents.AgentInput{
			Pair:    pair,
			Candles: candles,
			Regime:  &regimeCtx,
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
		fundOutput = fundamentalAgent.Run(gCtx, agents.AgentInput{Pair: pair, Regime: &regimeCtx})
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
		Regime:         &regimeCtx,
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
		Regime:         &regimeCtx,
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
			"regime", d.Regime,
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

			// Simpan ke signal store untuk evaluasi nanti oleh MetaObserver
			signalStore.Add(agents.PendingSignal{
				Pair:      pair,
				Signal:    d.Signal,
				Entry:     d.Entry,
				Regime:    knowledge.MarketRegime(d.Regime),
				CreatedAt: time.Now(),
				EvalAfter: time.Now().Add(evalDelay),
			})
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

// ════════════════════════════════════════════════════════════════════════════════
// evaluateSignal — cek apakah sinyal benar berdasarkan pergerakan harga aktual
// ════════════════════════════════════════════════════════════════════════════════

// evaluateSignalFull membandingkan harga entry dengan harga saat ini.
// Return: correct, pipsMove, evalPrice.
// Sinyal dianggap benar jika harga bergerak >= pipThreshold pip ke arah prediksi.
func evaluateSignalFull(sig agents.PendingSignal, marketAgent *agents.MarketDataAgent, timeframe string, pipThreshold float64) (bool, float64, float64) {
	// Ambil harga terbaru dari buffer
	latest := marketAgent.GetLatestCandle(sig.Pair, timeframe)
	if latest == nil {
		return false, 0, 0
	}

	currentPrice := latest.Close
	pipSize := 0.0001 // Major pairs
	if len(sig.Pair) > 4 && (sig.Pair[4:] == "JPY" || sig.Pair[:3] == "JPY") {
		pipSize = 0.01 // JPY pairs
	}

	pipsMove := (currentPrice - sig.Entry) / pipSize

	switch sig.Signal {
	case "BUY":
		return pipsMove >= pipThreshold, pipsMove, currentPrice
	case "SELL":
		return -pipsMove >= pipThreshold, -pipsMove, currentPrice
	default:
		return false, 0, currentPrice
	}
}
