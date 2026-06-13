package agents

import (
	"context"
	"math"
	"strings"
	"testing"
)

const epsilon = 1e-5

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestRiskAgentRun(t *testing.T) {
	entry := 1.08450
	defaultCandles := []Candle{{Close: entry}}

	tests := []struct {
		name string
		ctx  context.Context
		input AgentInput
		// Expected
		wantSuccess    bool
		wantErrContain string
		// For successful BUY/SELL cases
		wantLotSize    float64
		wantSL         float64
		wantTP         float64
		wantRiskAmount float64
		checkRisk      bool // whether to check Risk field values
		checkEmpty     bool // whether to check Risk is empty (HOLD)
	}{
		{
			name: "BUY signal: correct LotSize, SL below entry, TP above entry",
			ctx:  context.Background(),
			input: AgentInput{
				AccountBalance: 1000,
				RiskPercent:    1.0,
				Candles:        defaultCandles,
				Technical:      &TechnicalOutput{Signal: "BUY"},
			},
			wantSuccess:    true,
			wantLotSize:    0.05,
			wantSL:         1.08250,
			wantTP:         1.08850,
			wantRiskAmount: 10.0,
			checkRisk:      true,
		},
		{
			name: "SELL signal: correct LotSize, SL above entry, TP below entry",
			ctx:  context.Background(),
			input: AgentInput{
				AccountBalance: 1000,
				RiskPercent:    1.0,
				Candles:        defaultCandles,
				Technical:      &TechnicalOutput{Signal: "SELL"},
			},
			wantSuccess:    true,
			wantLotSize:    0.05,
			wantSL:         1.08650,
			wantTP:         1.08050,
			wantRiskAmount: 10.0,
			checkRisk:      true,
		},
		{
			name: "HOLD signal: Success=true, empty RiskOutput",
			ctx:  context.Background(),
			input: AgentInput{
				AccountBalance: 1000,
				RiskPercent:    1.0,
				Candles:        defaultCandles,
				Technical:      &TechnicalOutput{Signal: "HOLD"},
			},
			wantSuccess: true,
			checkEmpty:  true,
		},
		{
			name: "Invalid balance (zero): Success=false, error contains 0.00",
			ctx:  context.Background(),
			input: AgentInput{
				AccountBalance: 0,
				RiskPercent:    1.0,
				Candles:        defaultCandles,
				Technical:      &TechnicalOutput{Signal: "BUY"},
			},
			wantSuccess:    false,
			wantErrContain: "0.00",
		},
		{
			name: "Invalid balance (negative): Success=false, error contains value",
			ctx:  context.Background(),
			input: AgentInput{
				AccountBalance: -500.50,
				RiskPercent:    1.0,
				Candles:        defaultCandles,
				Technical:      &TechnicalOutput{Signal: "BUY"},
			},
			wantSuccess:    false,
			wantErrContain: "-500.50",
		},
		{
			name: "Nil TechnicalOutput: Success=false, error mentions technical output required",
			ctx:  context.Background(),
			input: AgentInput{
				AccountBalance: 1000,
				RiskPercent:    1.0,
				Candles:        defaultCandles,
				Technical:      nil,
			},
			wantSuccess:    false,
			wantErrContain: "technical output required",
		},
		{
			name: "RiskPercent default: RiskPercent=0 uses 1.0%",
			ctx:  context.Background(),
			input: AgentInput{
				AccountBalance: 1000,
				RiskPercent:    0,
				Candles:        defaultCandles,
				Technical:      &TechnicalOutput{Signal: "BUY"},
			},
			wantSuccess:    true,
			wantLotSize:    0.05, // 1000 * 0.01 / (20*10) = 0.05
			wantSL:         1.08250,
			wantTP:         1.08850,
			wantRiskAmount: 10.0,
			checkRisk:      true,
		},
		{
			name: "Context cancellation: Success=false, error wraps context error",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			input: AgentInput{
				AccountBalance: 1000,
				RiskPercent:    1.0,
				Candles:        defaultCandles,
				Technical:      &TechnicalOutput{Signal: "BUY"},
			},
			wantSuccess:    false,
			wantErrContain: "context canceled",
		},
	}

	agent := NewRiskAgent()

	// Verify Name() returns correct identifier
	if agent.Name() != "RiskAgent" {
		t.Fatalf("expected Name()=%q, got %q", "RiskAgent", agent.Name())
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := agent.Run(tc.ctx, tc.input)

			// Check AgentName is always set
			if got.AgentName != "RiskAgent" {
				t.Errorf("AgentName = %q, want %q", got.AgentName, "RiskAgent")
			}

			// Check Timestamp is non-zero
			if got.Timestamp.IsZero() {
				t.Error("Timestamp should not be zero")
			}

			// Check Success
			if got.Success != tc.wantSuccess {
				t.Errorf("Success = %v, want %v", got.Success, tc.wantSuccess)
			}

			// Check error cases
			if !tc.wantSuccess {
				if got.Error == nil {
					t.Fatal("expected non-nil Error for failure case")
				}
				if !strings.Contains(got.Error.Error(), tc.wantErrContain) {
					t.Errorf("Error = %q, want it to contain %q", got.Error.Error(), tc.wantErrContain)
				}
				return
			}

			// For successful cases, Risk must be non-nil
			if got.Risk == nil {
				t.Fatal("expected non-nil Risk for success case")
			}

			// Check HOLD (empty RiskOutput)
			if tc.checkEmpty {
				if got.Risk.LotSize != 0 || got.Risk.StopLoss != 0 || got.Risk.TakeProfit != 0 {
					t.Errorf("HOLD should have empty RiskOutput, got LotSize=%.5f, SL=%.5f, TP=%.5f",
						got.Risk.LotSize, got.Risk.StopLoss, got.Risk.TakeProfit)
				}
				if got.Risk.SLPips != 0 || got.Risk.TPPips != 0 || got.Risk.RiskAmount != 0 {
					t.Errorf("HOLD should have zero SLPips/TPPips/RiskAmount, got SLPips=%.1f, TPPips=%.1f, RiskAmount=%.2f",
						got.Risk.SLPips, got.Risk.TPPips, got.Risk.RiskAmount)
				}
				return
			}

			// Check BUY/SELL risk calculations
			if tc.checkRisk {
				if !floatEqual(got.Risk.LotSize, tc.wantLotSize) {
					t.Errorf("LotSize = %.5f, want %.5f", got.Risk.LotSize, tc.wantLotSize)
				}
				if !floatEqual(got.Risk.StopLoss, tc.wantSL) {
					t.Errorf("StopLoss = %.5f, want %.5f", got.Risk.StopLoss, tc.wantSL)
				}
				if !floatEqual(got.Risk.TakeProfit, tc.wantTP) {
					t.Errorf("TakeProfit = %.5f, want %.5f", got.Risk.TakeProfit, tc.wantTP)
				}
				if !floatEqual(got.Risk.RiskAmount, tc.wantRiskAmount) {
					t.Errorf("RiskAmount = %.5f, want %.5f", got.Risk.RiskAmount, tc.wantRiskAmount)
				}
				if got.Risk.SLPips != DefaultSLPips {
					t.Errorf("SLPips = %.1f, want %.1f", got.Risk.SLPips, DefaultSLPips)
				}
				if got.Risk.TPPips != DefaultTPPips {
					t.Errorf("TPPips = %.1f, want %.1f", got.Risk.TPPips, DefaultTPPips)
				}
				if got.Risk.LotSize <= 0 {
					t.Error("LotSize should be > 0 for BUY/SELL")
				}
				if got.Risk.StopLoss <= 0 {
					t.Error("StopLoss should be > 0")
				}
				if got.Risk.TakeProfit <= 0 {
					t.Error("TakeProfit should be > 0")
				}

				// Directional checks
				if tc.input.Technical.Signal == "BUY" {
					if got.Risk.StopLoss >= entry {
						t.Errorf("BUY StopLoss=%.5f should be below entry=%.5f", got.Risk.StopLoss, entry)
					}
					if got.Risk.TakeProfit <= entry {
						t.Errorf("BUY TakeProfit=%.5f should be above entry=%.5f", got.Risk.TakeProfit, entry)
					}
				}
				if tc.input.Technical.Signal == "SELL" {
					if got.Risk.StopLoss <= entry {
						t.Errorf("SELL StopLoss=%.5f should be above entry=%.5f", got.Risk.StopLoss, entry)
					}
					if got.Risk.TakeProfit >= entry {
						t.Errorf("SELL TakeProfit=%.5f should be below entry=%.5f", got.Risk.TakeProfit, entry)
					}
				}
			}
		})
	}
}
