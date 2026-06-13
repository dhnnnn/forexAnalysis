# Requirements Document

## Introduction

Agent 2 (TechnicalAgent) is the technical analysis component of the Forex Multi-Agent Bot pipeline. It receives OHLCV candle data from MarketDataAgent, computes four technical indicators (RSI-14, MACD-12/26/9, EMA-50/200, Bollinger Bands-20/2), produces a weighted TechnicalScore, and outputs a BUY/SELL/HOLD signal with a confidence value between 0.0 and 1.0. The agent implements the existing `Agent` interface and populates the pre-defined `TechnicalOutput` struct.

## Glossary

- **TechnicalAgent**: Agent 2 in the pipeline, responsible for computing technical indicators and producing trading signals
- **Candle**: A single OHLCV (Open, High, Low, Close, Volume) price bar for a given timeframe
- **RSI**: Relative Strength Index with period 14, using Wilder's smoothing method
- **MACD**: Moving Average Convergence Divergence with fast=12, slow=26, signal=9 parameters
- **EMA**: Exponential Moving Average, specifically EMA-50 and EMA-200 for trend filtering
- **Bollinger_Bands**: Volatility bands using period=20 and standard deviation multiplier=2
- **BBPosition**: A normalized value (0.0–1.0) representing the current price position within the Bollinger Bands, where 0.0 is the lower band and 1.0 is the upper band
- **TechnicalScore**: A weighted aggregate score computed as (RSI_Score × 0.40) + (MACD_Score × 0.40) + (BB_Score × 0.20)
- **Signal**: One of "BUY", "SELL", or "HOLD" derived from the TechnicalScore against defined thresholds
- **Confidence**: A float64 value between 0.0 and 1.0 representing the strength of the signal
- **Crossover**: A MACD event where the MACD line crosses above (bullish) or below (bearish) the signal line
- **Agent_Interface**: The Go interface defined in `internal/agents/agent.go` requiring `Name() string` and `Run(ctx context.Context, input AgentInput) AgentOutput`

## Requirements

### Requirement 1: Agent Interface Compliance

**User Story:** As a pipeline orchestrator, I want TechnicalAgent to implement the Agent interface, so that it integrates seamlessly into the multi-agent pipeline.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL implement the Agent_Interface with `Name()` returning "TechnicalAgent" and `Run(ctx, AgentInput)` returning `AgentOutput`
2. WHEN the Run method is called, THE TechnicalAgent SHALL populate the `AgentOutput.Technical` field with a non-nil `TechnicalOutput` struct on success
3. WHEN the Run method succeeds, THE TechnicalAgent SHALL set `AgentOutput.Success` to true and `AgentOutput.Error` to nil
4. WHEN the Run method fails, THE TechnicalAgent SHALL set `AgentOutput.Success` to false and `AgentOutput.Error` to a descriptive error message

### Requirement 2: Input Validation

**User Story:** As a pipeline orchestrator, I want TechnicalAgent to validate its input, so that downstream agents receive reliable output or clear error messages.

#### Acceptance Criteria

1. WHEN `AgentInput.Candles` contains fewer than 26 candles, THE TechnicalAgent SHALL return an error output stating the minimum candle requirement
2. WHEN `AgentInput.Candles` contains 26 or more candles, THE TechnicalAgent SHALL proceed with indicator computation
3. THE TechnicalAgent SHALL use the Close price from each Candle for all indicator calculations

### Requirement 3: RSI Calculation

**User Story:** As a technical analyst, I want TechnicalAgent to compute RSI(14) using Wilder's smoothing, so that overbought and oversold conditions are detected.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL calculate RSI with a period of 14 using Wilder's smoothing method
2. THE TechnicalAgent SHALL produce RSI values in the range 0.0 to 100.0
3. THE TechnicalAgent SHALL store the computed RSI value in `TechnicalOutput.RSI`

### Requirement 4: RSI Scoring

**User Story:** As a signal generator, I want consistent RSI scoring rules, so that trading signals are deterministic and reproducible.

#### Acceptance Criteria

1. WHEN RSI is less than or equal to 30, THE TechnicalAgent SHALL assign an RSI score of 0.85 with direction BUY
2. WHEN RSI is greater than 30 and less than or equal to 40, THE TechnicalAgent SHALL assign an RSI score of 0.65 with direction BUY
3. WHEN RSI is greater than or equal to 70, THE TechnicalAgent SHALL assign an RSI score of 0.85 with direction SELL
4. WHEN RSI is less than 70 and greater than or equal to 60, THE TechnicalAgent SHALL assign an RSI score of 0.65 with direction SELL
5. WHEN RSI is greater than 40 and less than 60, THE TechnicalAgent SHALL assign an RSI score of 0.50 with direction HOLD

### Requirement 5: MACD Calculation

**User Story:** As a technical analyst, I want TechnicalAgent to compute MACD(12,26,9), so that momentum crossovers and divergences are detected.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL calculate MACD using EMA-12 (fast), EMA-26 (slow), and signal line EMA-9
2. THE TechnicalAgent SHALL compute the MACD histogram as the difference between the MACD line and the signal line
3. THE TechnicalAgent SHALL store the computed histogram value in `TechnicalOutput.MACDHist`

### Requirement 6: MACD Scoring

**User Story:** As a signal generator, I want consistent MACD scoring rules, so that crossover events produce deterministic signals.

#### Acceptance Criteria

1. WHEN a bullish crossover occurs (MACD line crosses above signal line), THE TechnicalAgent SHALL assign a MACD score of 0.80 with direction BUY
2. WHEN a bearish crossover occurs (MACD line crosses below signal line), THE TechnicalAgent SHALL assign a MACD score of 0.80 with direction SELL
3. WHEN no crossover occurs and the histogram is greater than zero, THE TechnicalAgent SHALL assign a MACD score of 0.60 with direction BUY
4. WHEN no crossover occurs and the histogram is less than zero, THE TechnicalAgent SHALL assign a MACD score of 0.60 with direction SELL
5. WHEN no crossover occurs and the histogram equals zero, THE TechnicalAgent SHALL assign a MACD score of 0.50 with direction HOLD

### Requirement 7: EMA Calculation

**User Story:** As a technical analyst, I want TechnicalAgent to compute EMA-50 and EMA-200, so that long-term trend context is available for downstream agents.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL calculate EMA-50 and EMA-200 from Candle Close prices
2. THE TechnicalAgent SHALL store EMA-50 in `TechnicalOutput.EMA50` and EMA-200 in `TechnicalOutput.EMA200`
3. WHEN fewer than 200 candles are available, THE TechnicalAgent SHALL compute EMA-200 using all available candle data with reduced accuracy noted

### Requirement 8: Bollinger Bands Calculation

**User Story:** As a technical analyst, I want TechnicalAgent to compute Bollinger Bands(20,2), so that price volatility and band position are measured.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL calculate Bollinger Bands using a 20-period simple moving average and 2 standard deviations
2. THE TechnicalAgent SHALL compute BBPosition as `(Close - LowerBand) / (UpperBand - LowerBand)`, normalized between 0.0 and 1.0
3. THE TechnicalAgent SHALL store the computed BBPosition in `TechnicalOutput.BBPosition`
4. IF the UpperBand equals the LowerBand (zero bandwidth), THEN THE TechnicalAgent SHALL set BBPosition to 0.50

### Requirement 9: Bollinger Bands Scoring

**User Story:** As a signal generator, I want consistent Bollinger Bands scoring rules, so that band extremes produce deterministic signals.

#### Acceptance Criteria

1. WHEN BBPosition is less than or equal to 0.10, THE TechnicalAgent SHALL assign a BB score of 0.80 with direction BUY
2. WHEN BBPosition is greater than or equal to 0.90, THE TechnicalAgent SHALL assign a BB score of 0.80 with direction SELL
3. WHEN BBPosition is greater than 0.10 and less than 0.90, THE TechnicalAgent SHALL assign a BB score of 0.50 with direction HOLD

### Requirement 10: TechnicalScore Aggregation

**User Story:** As a signal generator, I want a weighted composite score, so that all indicators contribute proportionally to the final signal.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL compute TechnicalScore as `(RSI_Score × 0.40) + (MACD_Score × 0.40) + (BB_Score × 0.20)`
2. THE TechnicalAgent SHALL store the computed TechnicalScore in `TechnicalOutput.TechScore`
3. THE TechnicalAgent SHALL produce TechnicalScore values in the range 0.0 to 1.0

### Requirement 11: Signal Determination

**User Story:** As a pipeline consumer, I want the final signal to be derived from the TechnicalScore using clear thresholds, so that signal interpretation is unambiguous.

#### Acceptance Criteria

1. WHEN TechnicalScore is greater than or equal to 0.65, THE TechnicalAgent SHALL output Signal "BUY"
2. WHEN TechnicalScore is less than or equal to 0.35, THE TechnicalAgent SHALL output Signal "SELL"
3. WHEN TechnicalScore is greater than 0.35 and less than 0.65, THE TechnicalAgent SHALL output Signal "HOLD"
4. THE TechnicalAgent SHALL set `TechnicalOutput.Confidence` equal to the TechnicalScore value
5. THE TechnicalAgent SHALL store the Signal in `TechnicalOutput.Signal`

### Requirement 12: Reason Generation

**User Story:** As a downstream agent, I want a human-readable reason string, so that the DecisionAgent and WhatsApp alert can explain why a signal was generated.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL generate a Reason string summarizing the active indicator signals
2. WHEN RSI is less than or equal to 30, THE TechnicalAgent SHALL include "RSI oversold" in the Reason
3. WHEN RSI is greater than or equal to 70, THE TechnicalAgent SHALL include "RSI overbought" in the Reason
4. WHEN a MACD crossover occurs, THE TechnicalAgent SHALL include the crossover direction in the Reason
5. WHEN BBPosition is less than or equal to 0.10 or greater than or equal to 0.90, THE TechnicalAgent SHALL include Bollinger Band position in the Reason
6. WHEN no strong indicator signal exists, THE TechnicalAgent SHALL set Reason to "No strong technical signal"

### Requirement 13: Indicator Package Structure

**User Story:** As a developer, I want indicator calculations separated into individual files in `internal/indicators/`, so that each indicator is independently testable and maintainable.

#### Acceptance Criteria

1. THE TechnicalAgent SHALL delegate RSI calculation to a function in `internal/indicators/rsi.go`
2. THE TechnicalAgent SHALL delegate MACD calculation to a function in `internal/indicators/macd.go`
3. THE TechnicalAgent SHALL delegate EMA calculation to a function in `internal/indicators/moving_average.go`
4. THE TechnicalAgent SHALL delegate Bollinger Bands calculation to a function in `internal/indicators/bollinger.go`
5. THE TechnicalAgent SHALL delegate score aggregation to a function in `internal/indicators/scorer.go`
6. THE indicators package SHALL expose pure functions that accept a slice of float64 (close prices) and return computed values

### Requirement 14: Context Cancellation

**User Story:** As a pipeline orchestrator, I want TechnicalAgent to respect context cancellation, so that the system shuts down gracefully.

#### Acceptance Criteria

1. WHEN the context is cancelled before computation completes, THE TechnicalAgent SHALL return an error output indicating cancellation
2. THE TechnicalAgent SHALL check for context cancellation before starting indicator computations
