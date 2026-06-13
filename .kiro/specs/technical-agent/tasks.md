# Implementation Plan: TechnicalAgent (Agent 2)

## Overview

Implement TechnicalAgent as the second agent in the Forex Multi-Agent Bot pipeline. The implementation builds from foundational moving average utilities up through individual indicators, scoring logic, and finally the agent orchestration layer. All indicator functions are pure (accept `[]float64`, return deterministic results) and live in `internal/indicators/`. Property-based tests using `pgregory.net/rapid` validate correctness properties throughout.

## Tasks

- [x] 1. Set up indicators package and moving average utilities
  - [x] 1.1 Create `internal/indicators/moving_average.go` with `CalcSMA`, `CalcEMA`, and `CalcEMASeries`
    - Implement `CalcSMA(closes []float64, period int) float64` — simple moving average over last `period` elements
    - Implement `CalcEMA(closes []float64, period int) float64` — EMA with multiplier `2/(period+1)`, seeded with SMA of first `period` elements, returns final value
    - Implement `CalcEMASeries(closes []float64, period int) []float64` — full EMA series (first `period-1` entries are 0), used internally by MACD
    - _Requirements: 7.1, 13.3_

  - [ ]* 1.2 Write property tests for moving average functions in `internal/indicators/moving_average_test.go`
    - **Property: EMA series length equals input length**
    - **Property: SMA of constant series equals the constant value**
    - **Property: EMA converges toward recent prices (last EMA is between min and max of closes)**
    - Use `pgregory.net/rapid` with `rapid.Float64Range(0.5, 2.0)` slices of length 26–200
    - **Validates: Requirements 7.1**

- [x] 2. Implement RSI indicator
  - [x] 2.1 Create `internal/indicators/rsi.go` with `CalcRSI(closes []float64, period int) float64`
    - Implement Wilder's smoothing method: compute gains/losses, first average, smoothed average, RS, RSI
    - Handle edge case: avgLoss == 0 → return RSI = 100.0
    - Requires `len(closes) >= period + 1`
    - _Requirements: 3.1, 3.2, 3.3, 13.1_

  - [ ]* 2.2 Write property test for RSI range invariant in `internal/indicators/rsi_test.go`
    - **Property 3: RSI range invariant**
    - For any slice of close prices with length ≥ 15, `CalcRSI(closes, 14)` returns a value in [0.0, 100.0]
    - **Validates: Requirements 3.2**

- [x] 3. Implement MACD indicator
  - [x] 3.1 Create `internal/indicators/macd.go` with `MACDResult` struct and `CalcMACD(closes []float64, fast, slow, signal int) MACDResult`
    - Define `MACDResult` struct with MACDLine, SignalLine, Histogram, Crossover fields
    - Implement using `CalcEMASeries` from moving_average.go for fast and slow EMAs
    - Compute MACDLine series, then signal line as EMA of MACDLine series
    - Detect crossover by comparing current and previous histogram signs
    - _Requirements: 5.1, 5.2, 5.3, 13.2_

  - [ ]* 3.2 Write property test for MACD histogram invariant in `internal/indicators/macd_test.go`
    - **Property 5: MACD histogram invariant**
    - For any slice of close prices with length ≥ 35, `MACDResult.Histogram == MACDResult.MACDLine - MACDResult.SignalLine` (within epsilon)
    - **Validates: Requirements 5.2**

- [x] 4. Implement Bollinger Bands indicator
  - [x] 4.1 Create `internal/indicators/bollinger.go` with `BollingerResult` struct and `CalcBollingerBands(closes []float64, period int, multiplier float64) BollingerResult`
    - Define `BollingerResult` struct with Upper, Middle, Lower, BBPosition fields
    - Compute SMA(period) for middle band, standard deviation for band width
    - Compute BBPosition as `(close - lower) / (upper - lower)`, clamped to [0.0, 1.0]
    - Handle zero bandwidth: if upper == lower, set BBPosition = 0.50
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 13.4_

  - [ ]* 4.2 Write property test for BBPosition range invariant in `internal/indicators/bollinger_test.go`
    - **Property 7: BBPosition range invariant**
    - For any slice of close prices with length ≥ 20, `CalcBollingerBands(closes, 20, 2.0).BBPosition` is in [0.0, 1.0]
    - **Validates: Requirements 8.2**

- [x] 5. Checkpoint - Ensure all indicator tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 6. Implement scoring logic
  - [x] 6.1 Create `internal/indicators/scorer.go` with `ScoreResult` struct, `ScoreRSI`, `ScoreMACD`, `ScoreBB`, and `ComputeScore` functions
    - Define `ScoreResult` struct with RSIScore, RSIDir, MACDScore, MACDDir, BBScore, BBDir, TechScore, Signal, Confidence
    - Define weight constants: RSIWeight=0.40, MACDWeight=0.40, BBWeight=0.20
    - Implement `ScoreRSI(rsi float64) (float64, string)` with threshold table: ≤30→(0.85,BUY), (30,40]→(0.65,BUY), [60,70)→(0.65,SELL), ≥70→(0.85,SELL), (40,60)→(0.50,HOLD)
    - Implement `ScoreMACD(macd MACDResult) (float64, string)` with crossover and histogram rules
    - Implement `ScoreBB(bbPosition float64) (float64, string)` with threshold rules: ≤0.10→(0.80,BUY), ≥0.90→(0.80,SELL), else→(0.50,HOLD)
    - Implement `ComputeScore(rsi float64, macd MACDResult, bbPosition float64) ScoreResult` with weighted aggregation and signal determination (≥0.65→BUY, ≤0.35→SELL, else→HOLD)
    - _Requirements: 4.1–4.5, 6.1–6.5, 9.1–9.3, 10.1–10.3, 11.1–11.5, 13.5_

  - [ ]* 6.2 Write property tests for scoring functions in `internal/indicators/scorer_test.go`
    - **Property 4: RSI scoring correctness** — For any RSI in [0,100], ScoreRSI returns correct threshold-based (score, direction)
    - **Property 6: MACD scoring correctness** — For any MACDResult, ScoreMACD returns correct (score, direction) per crossover/histogram rules
    - **Property 8: Bollinger Bands scoring correctness** — For any BBPosition in [0,1], ScoreBB returns correct (score, direction) per threshold rules
    - **Property 9: TechnicalScore weighted aggregation formula** — ComputeScore produces TechScore == RSIScore×0.40 + MACDScore×0.40 + BBScore×0.20 (within epsilon)
    - **Property 10: TechnicalScore range invariant** — TechScore is always in [0.0, 1.0]
    - **Property 11: Signal determination from TechnicalScore** — Signal is BUY when ≥0.65, SELL when ≤0.35, HOLD otherwise
    - **Validates: Requirements 4.1–4.5, 6.1–6.5, 9.1–9.3, 10.1–10.3, 11.1–11.5**

- [x] 7. Implement TechnicalAgent wiring
  - [x] 7.1 Create `internal/agents/technical_agent.go` with `TechnicalAgent` struct implementing the `Agent` interface
    - Implement `NewTechnicalAgent() *TechnicalAgent`
    - Implement `Name() string` returning "TechnicalAgent"
    - Implement `Run(ctx context.Context, input AgentInput) AgentOutput`:
      1. Check `ctx.Err()` — return error output if cancelled
      2. Validate `len(input.Candles) >= 26` — return error if insufficient
      3. Extract `closePrices []float64` from `Candle.Close`
      4. Call `indicators.CalcRSI(closePrices, 14)`
      5. Call `indicators.CalcMACD(closePrices, 12, 26, 9)`
      6. Call `indicators.CalcEMA(closePrices, 50)` and `CalcEMA(closePrices, 200)`
      7. Call `indicators.CalcBollingerBands(closePrices, 20, 2.0)`
      8. Call `indicators.ComputeScore(rsi, macdResult, bbResult.BBPosition)`
      9. Build reason string using indicator signals
      10. Return `AgentOutput{Success: true, Technical: &TechnicalOutput{...}}`
    - Implement `buildReason(rsi float64, macd MACDResult, bbPos float64) string` helper
    - Include "RSI oversold"/"RSI overbought" when RSI ≤30/≥70
    - Include MACD crossover direction when crossover detected
    - Include BB band reference when BBPosition ≤0.10 or ≥0.90
    - Return "No strong technical signal" when no strong signals
    - _Requirements: 1.1–1.4, 2.1–2.3, 12.1–12.6, 14.1–14.2_

  - [ ]* 7.2 Write property tests for TechnicalAgent in `internal/agents/technical_agent_test.go`
    - **Property 1: Valid input produces successful output** — For any AgentInput with ≥26 candles with positive closes, Run returns Success=true, Error=nil, Technical!=nil
    - **Property 2: Insufficient data produces error output** — For any AgentInput with <26 candles, Run returns Success=false, Error!=nil
    - **Property 12: Confidence equals TechnicalScore** — TechnicalOutput.Confidence == TechnicalOutput.TechScore
    - **Property 13: Context cancellation returns error** — Cancelled context always produces Success=false, Error!=nil
    - **Property 14: Reason includes active indicator descriptions** — RSI ≤30 → Reason contains "RSI oversold", RSI ≥70 → "RSI overbought", MACD crossover → crossover text, BB extremes → BB text
    - **Validates: Requirements 1.2–1.4, 2.1–2.2, 11.4, 12.2–12.5, 14.1–14.2**

- [x] 8. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Add `pgregory.net/rapid` dependency and verify full test suite
  - [x] 9.1 Run `go get pgregory.net/rapid` to add the property-based testing library to `go.mod`
    - Verify the dependency is added correctly
    - Run `go mod tidy` to clean up
    - _Requirements: 13.6_

  - [x] 9.2 Run full test suite (`go test ./internal/indicators/... ./internal/agents/...`) and verify all tests pass
    - Fix any compilation errors or test failures
    - Ensure all property tests run minimum 100 iterations
    - _Requirements: 1.1–1.4, 3.1–3.3, 5.1–5.3, 7.1–7.3, 8.1–8.4, 10.1–10.3, 11.1–11.5_

- [x] 10. Final checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document (14 properties)
- Unit tests validate specific examples and edge cases
- All indicator functions are pure — accept `[]float64`, return deterministic results
- The `pgregory.net/rapid` dependency should be added early (task 9.1) but test files can reference it from the start since `go test` will trigger the download
- EMA/SMA utilities are built first because MACD and Bollinger Bands depend on them

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1"] },
    { "id": 2, "tasks": ["2.2", "3.1", "4.1"] },
    { "id": 3, "tasks": ["3.2", "4.2", "6.1"] },
    { "id": 4, "tasks": ["6.2", "7.1"] },
    { "id": 5, "tasks": ["7.2", "9.1"] },
    { "id": 6, "tasks": ["9.2"] }
  ]
}
```
