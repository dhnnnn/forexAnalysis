package agents

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Package-level constants for risk calculations.
const (
	DefaultSLPips  = 20.0   // Stop loss distance in pips
	DefaultTPPips  = 40.0   // Take profit distance in pips (1:2 RR)
	PipValuePerLot = 10.0   // USD per pip for 1 standard lot (major pairs)
	PipSize        = 0.0001 // Price movement per pip (major pairs)
)

// Compile-time interface satisfaction check.
var _ Agent = (*RiskAgent)(nil)

// RiskAgent (Agent 4) calculates position sizing, stop loss, and take profit
// levels based on account parameters and upstream TechnicalOutput.
type RiskAgent struct{}

// NewRiskAgent creates a new RiskAgent instance.
func NewRiskAgent() *RiskAgent {
	return &RiskAgent{}
}

// Name returns the agent's identifier.
func (a *RiskAgent) Name() string {
	return "RiskAgent"
}

// Run executes risk calculations based on account parameters and technical signal.
// Control flow: context check → balance validation → nil Technical check →
// HOLD early return → RiskPercent default → lot/SL/TP calculation → return populated AgentOutput.
func (a *RiskAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	// 1. Context cancellation check
	if ctx.Err() != nil {
		return errorOutput(a.Name(), fmt.Errorf("context cancelled: %w", ctx.Err()))
	}

	// 2. Validate AccountBalance > 0
	if input.AccountBalance <= 0 {
		return errorOutput(a.Name(), fmt.Errorf("invalid balance: %.2f", input.AccountBalance))
	}

	// 3. Validate Technical output exists
	if input.Technical == nil {
		return errorOutput(a.Name(), fmt.Errorf("technical output required to determine direction"))
	}

	// 4. HOLD signal → early return with empty RiskOutput
	if input.Technical.Signal == "HOLD" {
		return AgentOutput{
			AgentName: a.Name(),
			Success:   true,
			Risk:      &RiskOutput{},
			Timestamp: time.Now(),
		}
	}

	// 5. Apply RiskPercent default fallback
	riskPct := input.RiskPercent
	if riskPct <= 0 {
		riskPct = 1.0
	}

	// 6. Calculate entry price from last candle
	entry := input.Candles[len(input.Candles)-1].Close

	// 7. Calculate risk amount and lot size
	riskAmount := input.AccountBalance * (riskPct / 100.0)
	lotSize := math.Round((riskAmount/(DefaultSLPips*PipValuePerLot))*100) / 100

	// 8. Calculate SL and TP based on direction
	var sl, tp float64
	switch input.Technical.Signal {
	case "BUY":
		sl = entry - (DefaultSLPips * PipSize)
		tp = entry + (DefaultTPPips * PipSize)
	case "SELL":
		sl = entry + (DefaultSLPips * PipSize)
		tp = entry - (DefaultTPPips * PipSize)
	}

	// 9. Round SL and TP to 5 decimal places
	round5 := func(v float64) float64 { return math.Round(v*100000) / 100000 }

	// 10. Return populated RiskOutput
	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Risk: &RiskOutput{
			LotSize:    lotSize,
			StopLoss:   round5(sl),
			TakeProfit: round5(tp),
			SLPips:     DefaultSLPips,
			TPPips:     DefaultTPPips,
			RiskAmount: riskAmount,
		},
		Timestamp: time.Now(),
	}
}
