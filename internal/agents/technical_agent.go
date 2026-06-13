package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dhnnnn/forex-agent/internal/indicators"
)

// TechnicalAgent (Agent 2) computes technical indicators and produces
// a BUY/SELL/HOLD signal with confidence.
type TechnicalAgent struct{}

// NewTechnicalAgent creates a new TechnicalAgent instance.
func NewTechnicalAgent() *TechnicalAgent {
	return &TechnicalAgent{}
}

// Name returns the agent's identifier.
func (a *TechnicalAgent) Name() string {
	return "TechnicalAgent"
}

// Run executes technical analysis on the provided candle data.
// It computes RSI, MACD, EMA-50/200, Bollinger Bands, and aggregates
// them into a weighted TechnicalScore to produce a BUY/SELL/HOLD signal.
func (a *TechnicalAgent) Run(ctx context.Context, input AgentInput) AgentOutput {
	// 1. Check context cancellation
	if ctx.Err() != nil {
		return errorOutput(a.Name(), fmt.Errorf("context cancelled: %w", ctx.Err()))
	}

	// 2. Validate minimum candle count
	if len(input.Candles) < 26 {
		return errorOutput(a.Name(), fmt.Errorf("need min 26 candles for MACD, got %d", len(input.Candles)))
	}

	// 3. Extract close prices
	closePrices := make([]float64, len(input.Candles))
	for i, c := range input.Candles {
		closePrices[i] = c.Close
	}

	// 4. Compute RSI(14)
	rsi := indicators.CalcRSI(closePrices, 14)

	// 5. Compute MACD(12, 26, 9)
	macdResult := indicators.CalcMACD(closePrices, 12, 26, 9)

	// 6. Compute EMA-50 and EMA-200
	ema50 := indicators.CalcEMA(closePrices, 50)
	ema200 := indicators.CalcEMA(closePrices, 200)

	// 7. Compute Bollinger Bands(20, 2.0)
	bbResult := indicators.CalcBollingerBands(closePrices, 20, 2.0)

	// 8. Compute aggregated score
	scoreResult := indicators.ComputeScore(rsi, macdResult, bbResult.BBPosition)

	// 9. Build reason string
	reason := buildReason(rsi, macdResult, bbResult.BBPosition)

	// 10. Build and return successful output
	output := TechnicalOutput{
		Signal:     scoreResult.Signal,
		Confidence: scoreResult.Confidence,
		RSI:        rsi,
		MACDHist:   macdResult.Histogram,
		EMA50:      ema50,
		EMA200:     ema200,
		BBPosition: bbResult.BBPosition,
		TechScore:  scoreResult.TechScore,
		Reason:     reason,
	}

	return AgentOutput{
		AgentName: a.Name(),
		Success:   true,
		Timestamp: time.Now(),
		Technical: &output,
	}
}

// buildReason constructs a human-readable explanation of the active indicator signals.
// Returns "No strong technical signal" when no strong signals are detected.
func buildReason(rsi float64, macd indicators.MACDResult, bbPos float64) string {
	var signals []string

	// RSI signals
	if rsi <= 30 {
		signals = append(signals, "RSI oversold")
	} else if rsi >= 70 {
		signals = append(signals, "RSI overbought")
	}

	// MACD crossover signals
	if macd.Crossover == "bullish" {
		signals = append(signals, "MACD bullish crossover")
	} else if macd.Crossover == "bearish" {
		signals = append(signals, "MACD bearish crossover")
	}

	// Bollinger Bands position signals
	if bbPos <= 0.10 {
		signals = append(signals, "Price near lower Bollinger Band")
	} else if bbPos >= 0.90 {
		signals = append(signals, "Price near upper Bollinger Band")
	}

	if len(signals) == 0 {
		return "No strong technical signal"
	}

	return strings.Join(signals, " + ")
}

// errorOutput is a helper that creates a failed AgentOutput with the given error.
func errorOutput(name string, err error) AgentOutput {
	return AgentOutput{
		AgentName: name,
		Success:   false,
		Error:     err,
		Timestamp: time.Now(),
	}
}
