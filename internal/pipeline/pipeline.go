package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/agents"
	"github.com/dhnnnn/forexAnalysis/internal/chatbot"
	"github.com/dhnnnn/forexAnalysis/internal/graph"
	"github.com/dhnnnn/forexAnalysis/internal/graph/model"
	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
	"github.com/dhnnnn/forexAnalysis/internal/storage"
	"golang.org/x/sync/errgroup"
)

// ════════════════════════════════════════════════════════════════════════
// Pipeline — orchestrates the multi-agent forex analysis pipeline
// ════════════════════════════════════════════════════════════════════════

// Pipeline holds all dependencies required to run the analysis pipeline.
type Pipeline struct {
	// Agents
	marketAgent      *agents.MarketDataAgent
	regimeAgent      *agents.RegimeDetectionAgent
	technicalAgent   *agents.TechnicalAgent
	fundamentalAgent *agents.FundamentalAgent
	riskAgent        *agents.RiskAgent
	decisionAgent    *agents.DecisionAgent
	whatsAppAgent    *agents.WhatsAppAgent
	metaObserver     *agents.MetaObserverAgent
	ktaAgent         *agents.KnowledgeTransferAgent

	// Stores & infrastructure
	signalStore *agents.SignalStore
	broadcaster *knowledge.Broadcaster
	pubSub      *graph.PubSub
	chatHandler *chatbot.Handler
	store       *storage.Store
	kbStore     *knowledge.Store

	// Config values
	evalDelay    time.Duration
	pipThreshold float64
	pairs        []string
	timeframes   []string

	// Account config
	accountBalance float64
	riskPercent    float64
}

// PipelineConfig holds configuration values extracted from the main config.
type PipelineConfig struct {
	EvalDelay      time.Duration
	PipThreshold   float64
	Pairs          []string
	Timeframes     []string
	AccountBalance float64
	RiskPercent    float64
}

// PipelineDeps holds all external dependencies for the pipeline.
type PipelineDeps struct {
	MarketAgent      *agents.MarketDataAgent
	RegimeAgent      *agents.RegimeDetectionAgent
	TechnicalAgent   *agents.TechnicalAgent
	FundamentalAgent *agents.FundamentalAgent
	RiskAgent        *agents.RiskAgent
	DecisionAgent    *agents.DecisionAgent
	WhatsAppAgent    *agents.WhatsAppAgent
	MetaObserver     *agents.MetaObserverAgent
	KTAAgent         *agents.KnowledgeTransferAgent
	SignalStore       *agents.SignalStore
	Broadcaster      *knowledge.Broadcaster
	PubSub           *graph.PubSub
	ChatHandler      *chatbot.Handler
	Store            *storage.Store
	KBStore          *knowledge.Store
}

// NewPipeline creates a new Pipeline with all dependencies.
func NewPipeline(deps PipelineDeps, cfg PipelineConfig) *Pipeline {
	return &Pipeline{
		marketAgent:      deps.MarketAgent,
		regimeAgent:      deps.RegimeAgent,
		technicalAgent:   deps.TechnicalAgent,
		fundamentalAgent: deps.FundamentalAgent,
		riskAgent:        deps.RiskAgent,
		decisionAgent:    deps.DecisionAgent,
		whatsAppAgent:    deps.WhatsAppAgent,
		metaObserver:     deps.MetaObserver,
		ktaAgent:         deps.KTAAgent,
		signalStore:      deps.SignalStore,
		broadcaster:      deps.Broadcaster,
		pubSub:           deps.PubSub,
		chatHandler:      deps.ChatHandler,
		store:            deps.Store,
		kbStore:          deps.KBStore,
		evalDelay:        cfg.EvalDelay,
		pipThreshold:     cfg.PipThreshold,
		pairs:            cfg.Pairs,
		timeframes:       cfg.Timeframes,
		accountBalance:   cfg.AccountBalance,
		riskPercent:      cfg.RiskPercent,
	}
}

// SetChatHandler sets the chat handler after pipeline construction.
// Useful when chatHandler depends on pipeline state.
func (p *Pipeline) SetChatHandler(handler *chatbot.Handler) {
	p.chatHandler = handler
}

// ════════════════════════════════════════════════════════════════════════
// RunAll — runs the pipeline for all pairs concurrently
// ════════════════════════════════════════════════════════════════════════

// RunAll executes the pipeline for all configured pairs concurrently.
func (p *Pipeline) RunAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, pair := range p.pairs {
		wg.Add(1)
		go func(pair string) {
			defer wg.Done()
			if err := p.RunForPair(ctx, pair); err != nil {
				p.logError("RunForPair", err, "pair", pair)
			}
		}(pair)
	}
	wg.Wait()
}

// ════════════════════════════════════════════════════════════════════════
// RunForPair — processes a single pair through the full pipeline
// ════════════════════════════════════════════════════════════════════════

// RunForPair runs the full analysis pipeline for a single currency pair.
func (p *Pipeline) RunForPair(ctx context.Context, pair string) error {
	// Agent 1: Check if MarketDataAgent has enough data
	output := p.marketAgent.Run(ctx, agents.AgentInput{Pair: pair})
	if !output.Success {
		bufSize := p.marketAgent.BufferSize(pair, p.timeframes[0])
		slog.Debug("⏳ MarketDataAgent collecting...",
			"pair", pair,
			"buffer", fmt.Sprintf("%d/%d", bufSize, agents.MinCandlesRequired),
		)
		return ErrInsufficientData
	}

	candles := p.marketAgent.GetCandles(pair, p.timeframes[0])

	// Persist candles to TimescaleDB (non-blocking, best-effort)
	if p.store != nil {
		go func() {
			persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := p.store.InsertCandles(persistCtx, candles); err != nil {
				p.logError("persist_candles", err, "pair", pair)
			}
		}()
	}

	// ── Regime Detection: klasifikasi kondisi pasar ───────────────────
	regimeCtx := p.regimeAgent.Detect(ctx, pair, candles)
	slog.Debug("🔍 RegimeDetection completed",
		"pair", pair,
		"regime", string(regimeCtx.Regime),
		"adx", fmt.Sprintf("%.2f", regimeCtx.ADX),
		"atr", fmt.Sprintf("%.6f", regimeCtx.ATR),
		"volatility", fmt.Sprintf("%.4f", regimeCtx.Volatility),
		"trend_strength", fmt.Sprintf("%.2f", regimeCtx.TrendStrength),
	)

	// Persist regime log ke Postgres (non-blocking)
	if p.store != nil {
		go func() {
			persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := p.store.InsertRegimeLog(persistCtx, storage.RegimeLogEntry{
				Pair:          pair,
				Regime:        string(regimeCtx.Regime),
				ADX:           regimeCtx.ADX,
				ATR:           regimeCtx.ATR,
				Volatility:    regimeCtx.Volatility,
				TrendStrength: regimeCtx.TrendStrength,
				DetectedAt:    regimeCtx.DetectedAt,
			}); err != nil {
				p.logError("persist_regime", err, "pair", pair)
			}
		}()
	}

	// ── Broadcast KnowledgeRules ke semua subscriber agents ───────────
	p.broadcaster.Broadcast(ctx, regimeCtx)

	// Publish regime ke GraphQL subscribers
	p.pubSub.PublishRegime(&model.RegimeContext{
		Pair:          pair,
		Regime:        strings.ToUpper(string(regimeCtx.Regime)),
		ADX:           regimeCtx.ADX,
		ATR:           regimeCtx.ATR,
		Volatility:    regimeCtx.Volatility,
		TrendStrength: regimeCtx.TrendStrength,
		DetectedAt:    regimeCtx.DetectedAt.Format(time.RFC3339),
	})

	// ── Agent 2 + 3: Technical & Fundamental Analysis (CONCURRENT) ────
	var techOutput, fundOutput agents.AgentOutput

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		techOutput = p.technicalAgent.Run(gCtx, agents.AgentInput{
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
		fundOutput = p.fundamentalAgent.Run(gCtx, agents.AgentInput{Pair: pair, Regime: &regimeCtx})
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

	// Publish agent debate entries ke GraphQL subscribers
	if techOutput.Success && techOutput.Technical != nil {
		rsi := techOutput.Technical.RSI
		macdH := techOutput.Technical.MACDHist
		bbPos := techOutput.Technical.BBPosition
		p.pubSub.PublishAgentOutput(&model.AgentDebateEntry{
			ID:         fmt.Sprintf("tech-%s-%d", pair, time.Now().UnixMilli()),
			Timestamp:  time.Now().Format(time.RFC3339),
			Pair:       pair,
			Agent:      "TechnicalAgent",
			Signal:     techOutput.Technical.Signal,
			Confidence: techOutput.Technical.Confidence,
			Reasoning:  techOutput.Technical.Reason,
			Details: &model.AgentDetails{
				RSI:        &rsi,
				MACDHist:   &macdH,
				BBPosition: &bbPos,
			},
		})
	}
	if fundOutput.Success && fundOutput.Fundamental != nil {
		score := fundOutput.Fundamental.Score
		sent := fundOutput.Fundamental.Sentiment
		p.pubSub.PublishAgentOutput(&model.AgentDebateEntry{
			ID:         fmt.Sprintf("fund-%s-%d", pair, time.Now().UnixMilli()),
			Timestamp:  time.Now().Format(time.RFC3339),
			Pair:       pair,
			Agent:      "FundamentalAgent",
			Signal:     fundSentimentToSignal(fundOutput.Fundamental.Sentiment),
			Confidence: fundOutput.Fundamental.Confidence,
			Reasoning:  fundOutput.Fundamental.Reason,
			Details: &model.AgentDetails{
				Sentiment: &sent,
				Score:     &score,
			},
		})
	}

	// ── Agent 4: Risk Management (needs technical signal) ─────────────
	riskInput := agents.AgentInput{
		Pair:           pair,
		Candles:        candles,
		Technical:      techOutput.Technical,
		Regime:         &regimeCtx,
		AccountBalance: p.accountBalance,
		RiskPercent:    p.riskPercent,
	}
	riskOutput := p.riskAgent.Run(ctx, riskInput)
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
		AccountBalance: p.accountBalance,
		RiskPercent:    p.riskPercent,
	}
	decisionOutput := p.decisionAgent.Run(ctx, decisionInput)

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
		p.chatHandler.SetLastSignal(fmt.Sprintf("%s %s %d%%", d.Signal, pair, d.ConfPct))

		// Persist signal to TimescaleDB
		if p.store != nil {
			if err := p.store.InsertSignal(ctx, d); err != nil {
				p.logError("persist_signal", err, "pair", pair)
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
			p.signalStore.Add(agents.PendingSignal{
				Pair:      pair,
				Signal:    d.Signal,
				Entry:     d.Entry,
				Regime:    knowledge.MarketRegime(d.Regime),
				CreatedAt: time.Now(),
				EvalAfter: time.Now().Add(p.evalDelay),
			})
		}

		// ── Agent 6: WhatsApp Notification ────────────────────────────
		waInput := agents.AgentInput{
			Pair:     pair,
			Decision: decisionOutput.Decision,
		}
		waOutput := p.whatsAppAgent.Run(ctx, waInput)
		if !waOutput.Success {
			p.logError("whatsapp_notify", waOutput.Error, "pair", pair)
		}
	} else {
		slog.Warn("⚠️ DecisionAgent failed", "pair", pair, "error", decisionOutput.Error)
		return NewPipelineError("DecisionAgent", pair, "run", decisionOutput.Error)
	}

	return nil
}

// ════════════════════════════════════════════════════════════════════════
// ProcessMetaObserver — post-pipeline MetaObserver + KTA logic
// ════════════════════════════════════════════════════════════════════════

// ProcessMetaObserver runs the MetaObserver after pipeline completion,
// processes experience reports through KTA, and persists results.
func (p *Pipeline) ProcessMetaObserver(ctx context.Context) {
	// Check context before proceeding
	if ctx.Err() != nil {
		return
	}

	reports := p.metaObserver.Observe()
	if len(reports) > 0 {
		slog.Info("🚨 MetaObserver detected degradation", "report_count", len(reports))

		// Persist ExperienceReports ke Postgres
		if p.store != nil {
			go func() {
				persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := p.store.InsertExperienceReports(persistCtx, reports); err != nil {
					p.logError("persist_experience_reports", err)
				}
			}()
		}

		// KnowledgeTransferAgent: proses reports → KnowledgeRules
		newRules := p.ktaAgent.Process(ctx, reports)
		if len(newRules) > 0 {
			slog.Info("✨ KTA generated new rules", "rule_count", len(newRules))

			// Persist rules ke Postgres
			if p.store != nil {
				go func() {
					persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if err := p.store.InsertKnowledgeRules(persistCtx, newRules); err != nil {
						p.logError("persist_knowledge_rules", err)
					}
				}()
			}
		}
	}

	// Persist metrics ke Redis
	if ctx.Err() != nil {
		return
	}
	metrics := p.metaObserver.GetMetrics()
	if err := p.kbStore.SaveAllMetrics(ctx, metrics); err != nil {
		p.logError("save_metrics_redis", err)
	}
}

// ════════════════════════════════════════════════════════════════════════
// StartEvaluator — evaluator goroutine for signal evaluation
// ════════════════════════════════════════════════════════════════════════

// StartEvaluator runs the signal evaluator loop that checks pending signals
// after their evaluation delay has passed. Blocks until ctx is cancelled.
func (p *Pipeline) StartEvaluator(ctx context.Context) {
	evalTicker := time.NewTicker(1 * time.Minute)
	defer evalTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-evalTicker.C:
			p.evaluatePendingSignals(ctx)
		}
	}
}

// evaluatePendingSignals checks and evaluates all signals that are ready.
func (p *Pipeline) evaluatePendingSignals(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}

	readySignals := p.signalStore.GetReadyForEvaluation()
	if len(readySignals) == 0 {
		return
	}

	for _, sig := range readySignals {
		correct, pipsMove, evalPrice := p.evaluateSignal(sig)

		// Record outcome ke KEDUA agen (Technical + Fundamental)
		for _, agentName := range []string{"TechnicalAgent", "FundamentalAgent"} {
			p.metaObserver.RecordOutcome(agents.SignalOutcome{
				AgentName: agentName,
				Pair:      sig.Pair,
				Correct:   correct,
				Regime:    sig.Regime,
				Timestamp: time.Now(),
			})
		}

		// Persist ke Postgres (non-blocking)
		if p.store != nil {
			go func(s agents.PendingSignal, c bool, pm float64, ep float64) {
				persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
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
				if err := p.store.InsertPerformanceLog(persistCtx, entry); err != nil {
					p.logError("persist_performance_log", err, "pair", s.Pair)
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

// ════════════════════════════════════════════════════════════════════════
// evaluateSignal — checks if a signal was correct based on price movement
// ════════════════════════════════════════════════════════════════════════

// evaluateSignal compares entry price with current price.
// Returns: correct, pipsMove, evalPrice.
func (p *Pipeline) evaluateSignal(sig agents.PendingSignal) (bool, float64, float64) {
	// Ambil harga terbaru dari buffer
	latest := p.marketAgent.GetLatestCandle(sig.Pair, p.timeframes[0])
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
		return pipsMove >= p.pipThreshold, pipsMove, currentPrice
	case "SELL":
		return -pipsMove >= p.pipThreshold, -pipsMove, currentPrice
	default:
		return false, 0, currentPrice
	}
}

// ════════════════════════════════════════════════════════════════════════
// logError — structured error logging helper
// ════════════════════════════════════════════════════════════════════════

// logError logs an error with structured fields for consistent error reporting.
func (p *Pipeline) logError(operation string, err error, fields ...any) {
	if err == nil {
		return
	}
	args := []any{"operation", operation, "error", err}
	args = append(args, fields...)
	slog.Debug("⚠️ Pipeline error", args...)
}

// ════════════════════════════════════════════════════════════════════════
// fundSentimentToSignal — package-level helper
// ════════════════════════════════════════════════════════════════════════

// fundSentimentToSignal maps fundamental sentiment to a signal string.
func fundSentimentToSignal(sentiment string) string {
	switch strings.ToLower(sentiment) {
	case "bullish":
		return "BUY"
	case "bearish":
		return "SELL"
	default:
		return "HOLD"
	}
}
