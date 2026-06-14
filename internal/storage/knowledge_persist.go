package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/knowledge"
)

// ════════════════════════════════════════════════════════════════════════
// Knowledge Persistence — persist data ke Postgres untuk paper/eksperimen
// ════════════════════════════════════════════════════════════════════════

// InsertExperienceReport menyimpan ExperienceReport ke tabel experience_reports.
func (s *Store) InsertExperienceReport(ctx context.Context, report knowledge.ExperienceReport) error {
	if s == nil {
		return nil
	}

	const query = `
		INSERT INTO experience_reports (
			agent_name, pair, accuracy_before, accuracy_now, accuracy_delta,
			loss_streak, active_regime, cause, reasoning, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.pool.Exec(ctx, query,
		report.AgentName,
		report.Pair,
		report.AccuracyBefore,
		report.AccuracyNow,
		report.AccuracyDelta,
		report.LossStreak,
		string(report.ActiveRegime),
		report.Cause,
		report.Reasoning,
		report.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert experience report: %w", err)
	}

	slog.Debug("💾 ExperienceReport persisted",
		"agent", report.AgentName,
		"regime", string(report.ActiveRegime),
	)
	return nil
}

// InsertExperienceReports menyimpan batch ExperienceReport.
func (s *Store) InsertExperienceReports(ctx context.Context, reports []knowledge.ExperienceReport) error {
	if s == nil || len(reports) == 0 {
		return nil
	}

	const query = `
		INSERT INTO experience_reports (
			agent_name, pair, accuracy_before, accuracy_now, accuracy_delta,
			loss_streak, active_regime, cause, reasoning, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	batch := &pgxBatch{}
	for _, r := range reports {
		batch.Queue(query,
			r.AgentName, r.Pair, r.AccuracyBefore, r.AccuracyNow, r.AccuracyDelta,
			r.LossStreak, string(r.ActiveRegime), r.Cause, r.Reasoning, r.Timestamp,
		)
	}

	br := s.pool.SendBatch(ctx, batch.batch())
	defer br.Close()

	for i := 0; i < len(reports); i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert experience report %d: %w", i, err)
		}
	}

	slog.Debug("💾 ExperienceReports persisted", "count", len(reports))
	return nil
}

// InsertKnowledgeRule menyimpan KnowledgeRule ke tabel knowledge_rules.
func (s *Store) InsertKnowledgeRule(ctx context.Context, rule knowledge.KnowledgeRule) error {
	if s == nil {
		return nil
	}

	conditionJSON, _ := json.Marshal(rule.Condition)
	actionJSON, _ := json.Marshal(rule.Action)

	const query = `
		INSERT INTO knowledge_rules (
			id, source_agent, target_agent, regime,
			condition_json, action_json, confidence,
			reasoning, apply_count, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			apply_count = EXCLUDED.apply_count
	`

	_, err := s.pool.Exec(ctx, query,
		rule.ID,
		rule.SourceAgent,
		rule.Action.TargetAgent,
		string(rule.Condition.Regime),
		conditionJSON,
		actionJSON,
		rule.Confidence,
		rule.Reasoning,
		rule.ApplyCount,
		rule.CreatedAt,
		rule.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert knowledge rule: %w", err)
	}

	slog.Debug("💾 KnowledgeRule persisted",
		"id", rule.ID[:8],
		"source", rule.SourceAgent,
		"target", rule.Action.TargetAgent,
	)
	return nil
}

// InsertKnowledgeRules menyimpan batch KnowledgeRule.
func (s *Store) InsertKnowledgeRules(ctx context.Context, rules []knowledge.KnowledgeRule) error {
	if s == nil || len(rules) == 0 {
		return nil
	}

	for _, rule := range rules {
		if err := s.InsertKnowledgeRule(ctx, rule); err != nil {
			slog.Debug("⚠️ Failed to persist rule", "id", rule.ID[:8], "error", err)
		}
	}
	return nil
}

// InsertPerformanceLog menyimpan satu evaluasi sinyal ke agent_performance_log.
func (s *Store) InsertPerformanceLog(ctx context.Context, log PerformanceLogEntry) error {
	if s == nil {
		return nil
	}

	const query = `
		INSERT INTO agent_performance_log (
			agent_name, pair, regime, signal,
			entry_price, eval_price, correct, pips_move,
			signal_time, eval_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.pool.Exec(ctx, query,
		log.AgentName,
		log.Pair,
		log.Regime,
		log.Signal,
		log.EntryPrice,
		log.EvalPrice,
		log.Correct,
		log.PipsMove,
		log.SignalTime,
		log.EvalTime,
	)
	if err != nil {
		return fmt.Errorf("insert performance log: %w", err)
	}
	return nil
}

// InsertRegimeLog menyimpan regime detection result ke regime_log.
func (s *Store) InsertRegimeLog(ctx context.Context, log RegimeLogEntry) error {
	if s == nil {
		return nil
	}

	const query = `
		INSERT INTO regime_log (
			pair, regime, adx, atr, volatility, trend_strength, detected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.pool.Exec(ctx, query,
		log.Pair,
		log.Regime,
		log.ADX,
		log.ATR,
		log.Volatility,
		log.TrendStrength,
		log.DetectedAt,
	)
	if err != nil {
		return fmt.Errorf("insert regime log: %w", err)
	}
	return nil
}

// ════════════════════════════════════════════════════════════════════════
// Log Entry Structs
// ════════════════════════════════════════════════════════════════════════

// PerformanceLogEntry adalah data untuk tabel agent_performance_log.
type PerformanceLogEntry struct {
	AgentName  string
	Pair       string
	Regime     string
	Signal     string
	EntryPrice float64
	EvalPrice  float64
	Correct    bool
	PipsMove   float64
	SignalTime  time.Time
	EvalTime   time.Time
}

// RegimeLogEntry adalah data untuk tabel regime_log.
type RegimeLogEntry struct {
	Pair          string
	Regime        string
	ADX           float64
	ATR           float64
	Volatility    float64
	TrendStrength float64
	DetectedAt    time.Time
}
