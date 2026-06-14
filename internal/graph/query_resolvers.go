package graph

import (
	"context"
	"strings"
	"time"

	"github.com/dhnnnn/forexAnalysis/internal/graph/model"
)

// ════════════════════════════════════════════════════════════════════════
// Query Resolvers
// ════════════════════════════════════════════════════════════════════════

// Candles returns candle data for a pair and timeframe.
func (r *Resolver) Candles(ctx context.Context, pair string, timeframe string, limit *int) ([]*model.Candle, error) {
	n := 200
	if limit != nil {
		n = *limit
	}

	// Try from MarketAgent buffer first
	agentCandles := r.MarketAgent.GetCandles(pair, timeframe)
	if len(agentCandles) == 0 && r.Store != nil {
		// Fallback to DB
		dbCandles, err := r.Store.GetCandles(ctx, pair, timeframe, n)
		if err == nil {
			agentCandles = dbCandles
		}
	}

	// Limit
	if len(agentCandles) > n {
		agentCandles = agentCandles[len(agentCandles)-n:]
	}

	var result []*model.Candle
	for _, c := range agentCandles {
		result = append(result, &model.Candle{
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
	}
	return result, nil
}

// Signals returns signal history.
func (r *Resolver) Signals(ctx context.Context, pair *string, limit *int, status *string) ([]*model.SignalEntry, error) {
	n := 50
	if limit != nil {
		n = *limit
	}

	p := ""
	if pair != nil {
		p = *pair
	}

	if r.Store == nil {
		return []*model.SignalEntry{}, nil
	}

	// Get from DB
	if p == "" && len(r.Pairs) > 0 {
		p = r.Pairs[0]
	}

	dbSignals, err := r.Store.GetRecentSignals(ctx, p, n)
	if err != nil {
		return nil, err
	}

	var result []*model.SignalEntry
	for i, s := range dbSignals {
		entry := &model.SignalEntry{
			ID:            i + 1,
			Timestamp:     s.Timestamp.Format(time.RFC3339),
			Pair:          s.Pair,
			Signal:        s.Signal,
			Confidence:    s.Confidence,
			Regime:        strings.ToUpper(s.Regime),
			Entry:         s.Entry,
			StopLoss:      s.StopLoss,
			TakeProfit:    s.TakeProfit,
			LotSize:       s.LotSize,
			TechSignal:    s.TechSignal,
			TechConf:      s.TechConf,
			TechReason:    s.TechReason,
			FundSentiment: s.FundSentiment,
			FundConf:      s.FundConf,
			FundReason:    s.FundReason,
		}
		result = append(result, entry)
	}
	return result, nil
}

// AgentSummaries returns performance summaries for all agents.
func (r *Resolver) AgentSummaries(ctx context.Context) ([]*model.AgentSummary, error) {
	metrics := r.MetaObserver.GetMetrics()

	var result []*model.AgentSummary
	for _, m := range metrics {
		result = append(result, &model.AgentSummary{
			AgentName:      m.AgentName,
			Accuracy:       m.Accuracy,
			AccuracyPrev:   m.AccuracyPrev,
			WinCount:       m.WinCount,
			LossCount:      m.LossCount,
			LossStreak:     m.LossStreak,
			DominantRegime: strings.ToUpper(string(m.ActiveRegime)),
			History:        []bool{}, // TODO: populate from tracker
		})
	}
	return result, nil
}

// ActiveRules returns currently active knowledge rules.
func (r *Resolver) ActiveRules(ctx context.Context) ([]*model.KnowledgeRule, error) {
	rules, err := r.KBStore.GetActiveRules(ctx)
	if err != nil {
		return nil, err
	}

	var result []*model.KnowledgeRule
	for _, rule := range rules {
		result = append(result, &model.KnowledgeRule{
			ID:          rule.ID,
			SourceAgent: rule.SourceAgent,
			TargetAgent: rule.Action.TargetAgent,
			Regime:      strings.ToUpper(string(rule.Condition.Regime)),
			WeightDelta: rule.Action.WeightDelta,
			MinWeight:   rule.Action.MinWeight,
			Confidence:  rule.Confidence,
			Reasoning:   rule.Reasoning,
			ApplyCount:  rule.ApplyCount,
			CreatedAt:   rule.CreatedAt.Format(time.RFC3339),
			ExpiresAt:   rule.ExpiresAt.Format(time.RFC3339),
			Status:      "active",
		})
	}
	return result, nil
}

// Pairs returns configured trading pairs.
func (r *Resolver) QueryPairs(ctx context.Context) ([]string, error) {
	return r.Pairs, nil
}

// ConnectionStatus returns current connection status.
func (r *Resolver) ConnectionStatus(ctx context.Context) (string, error) {
	return "connected", nil
}
