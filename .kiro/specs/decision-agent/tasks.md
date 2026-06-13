# Implementation Plan: DecisionAgent (Agent 5)

## Overview

Implement DecisionAgent, the final decision-making component in the forex multi-agent pipeline. It aggregates outputs from TechnicalAgent, FundamentalAgent, and RiskAgent to produce a unified trading signal (BUY, SELL, or HOLD) with weighted scoring, confidence calculation, optional ML boost, risk level assessment, and full upstream data transparency. The implementation follows established patterns (compile-time interface check, context cancellation first, `errorOutput` helper, interface-based DI).

## Tasks

- [x] 1. Define MLPredictor interface and SignalConfig
  - [x] 1.1 Add `MLPredictor` interface and `SignalConfig` struct to `internal/agents/agent.go`
    - Define `MLPredictor` interface with `Predict(ctx context.Context, tech *TechnicalOutput, candles []Candle) (float64, error)`
    - Define `SignalConfig` struct with `BuyThreshold`, `SellThreshold`, `TechWeight`, `FundWeight`, `MLBoostWeight` (all float64)
    - Define `DefaultSignalConfig()` function returning defaults: BuyThreshold=0.65, SellThreshold=0.35, TechWeight=0.60, FundWeight=0.40, MLBoostWeight=0.20
    - _Requirements: 1.2, 2.1, 2.2, 3.1, 3.4, 5.1_

  - [x] 1.2 Implement `validateConfig` function in `internal/agents/decision_agent.go`
    - Validate TechWeight and FundWeight are each in [0.0, 1.0] and sum to 1.0; else use defaults
    - Validate SellThreshold < BuyThreshold and both in [0.0, 1.0]; else use defaults
    - Validate MLBoostWeight is in [0.0, 1.0]; else use default 0.20
    - Return normalized SignalConfig with valid values
    - _Requirements: 2.2, 2.5, 3.4, 3.5_

  - [ ]* 1.3 Write property test for config validation fallback (Property 6)
    - **Property 6: Config Validation Fallback**
    - Generate random invalid SignalConfig values (weights not summing to 1.0, out of range, thresholds unordered)
    - Assert validateConfig produces a config that yields identical results to DefaultSignalConfig given the same inputs
    - **Validates: Requirements 2.2, 2.5, 3.4, 3.5**

- [x] 2. Implement core computation functions
  - [x] 2.1 Implement `calcWeightedScore` in `internal/agents/decision_agent.go`
    - Accept `*TechnicalOutput`, `*FundamentalOutput`, and `SignalConfig`
    - Use default 0.5 for nil Technical TechScore or nil Fundamental Score
    - Clamp input scores to [0.0, 1.0]
    - Return `(techScore × cfg.TechWeight) + (fundScore × cfg.FundWeight)`
    - _Requirements: 2.1, 2.3, 2.4_

  - [x] 2.2 Implement `determineSignal` in `internal/agents/decision_agent.go`
    - Accept weightedScore float64 and SignalConfig
    - Return "BUY" if score >= BuyThreshold, "SELL" if score <= SellThreshold, "HOLD" otherwise
    - _Requirements: 3.1, 3.2, 3.3_

  - [x] 2.3 Implement `calcConfidence` in `internal/agents/decision_agent.go`
    - Accept `*TechnicalOutput`, `*FundamentalOutput`, and `SignalConfig`
    - Use default 0.5 for nil Technical Confidence or nil Fundamental Confidence
    - Clamp input confidences to [0.0, 1.0]
    - Return `(techConf × cfg.TechWeight) + (fundConf × cfg.FundWeight)`
    - _Requirements: 4.1, 4.2, 4.4_

  - [x] 2.4 Implement `applyMLBoost` in `internal/agents/decision_agent.go`
    - Accept context, baseConfidence, MLPredictor, *TechnicalOutput, []Candle, and SignalConfig
    - If mlClient is nil, return baseConfidence and mlScore=0.0
    - Create child context with 500ms timeout
    - Call Predict; on error or score <= 0, return baseConfidence and mlScore=0.0
    - On success: return `(baseConf × (1.0 - cfg.MLBoostWeight)) + (mlScore × cfg.MLBoostWeight)` and the mlScore
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [x] 2.5 Implement `assessRiskLevel` in `internal/agents/decision_agent.go`
    - Accept confidence float64
    - Return "LOW" if >= 0.75, "MEDIUM" if >= 0.50, "HIGH" if < 0.50
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [ ]* 2.6 Write property test for weighted score calculation (Property 1)
    - **Property 1: Weighted Score Calculation**
    - Generate random TechnicalOutput (TechScore in [0,1]) and FundamentalOutput (Score in [0,1]), either of which may be nil
    - Assert result equals `(techScore × techWeight) + (fundScore × fundWeight)` and is in [0,1]
    - **Validates: Requirements 2.1, 2.3, 2.4**

  - [ ]* 2.7 Write property test for signal determination (Property 2)
    - **Property 2: Signal Determination by Threshold Partition**
    - Generate random weighted scores in [0,1] and valid SignalConfig thresholds
    - Assert signal is exactly one of BUY/SELL/HOLD matching the threshold partition
    - Assert the three regions are non-overlapping and complete
    - **Validates: Requirements 3.1, 3.2, 3.3**

  - [ ]* 2.8 Write property test for confidence calculation (Property 3)
    - **Property 3: Confidence Calculation**
    - Generate random TechnicalOutput (Confidence in [0,1]) and FundamentalOutput (Confidence in [0,1]), either nil
    - Assert result equals `(techConf × techWeight) + (fundConf × fundWeight)` and ConfPct == int(confidence × 100)
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.4**

  - [ ]* 2.9 Write property test for ML boost formula (Property 4)
    - **Property 4: ML Boost Formula**
    - Generate random base confidence in [0,1] and ML score in (0,1]
    - Assert adjusted confidence equals `(baseConf × (1.0 - mlBoostWeight)) + (mlScore × mlBoostWeight)` and remains in [0,1]
    - **Validates: Requirements 5.2**

  - [ ]* 2.10 Write property test for risk level classification (Property 5)
    - **Property 5: Risk Level Classification**
    - Generate random confidence values in [0,1]
    - Assert "LOW" if >= 0.75, "MEDIUM" if in [0.50, 0.75), "HIGH" if < 0.50
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4**

- [x] 3. Checkpoint - Ensure core computation tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. Implement DecisionAgent struct and Run method
  - [x] 4.1 Create `internal/agents/decision_agent.go` with DecisionAgent struct, constructor, and Name()
    - Define `DecisionAgent` struct with `config SignalConfig` and `mlClient MLPredictor` fields
    - Add compile-time interface check: `var _ Agent = (*DecisionAgent)(nil)`
    - Implement `NewDecisionAgent(config SignalConfig, mlClient MLPredictor) *DecisionAgent` that validates config
    - Implement `Name()` returning "DecisionAgent"
    - _Requirements: 1.1, 1.2_

  - [x] 4.2 Implement `Run` method on DecisionAgent
    - Check context cancellation first → return errorOutput on cancelled
    - Call calcWeightedScore with input.Technical, input.Fundamental, config
    - Call determineSignal with weighted score and config
    - Call calcConfidence with input.Technical, input.Fundamental, config
    - Call applyMLBoost with mlClient, base confidence, tech, candles, config
    - Call assessRiskLevel with final confidence
    - Build entry/SL/TP/lot: zero for HOLD; from Risk+last candle for BUY/SELL
    - Build upstream transparency fields (nil-safe defaults for missing inputs)
    - Populate ConfPct as int(confidence × 100)
    - Return AgentOutput with Success=true, non-nil DecisionOutput
    - _Requirements: 1.1, 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 8.1, 8.2, 8.3, 8.4, 8.5, 9.1, 9.3, 10.1, 10.2, 10.3, 10.4_

  - [ ]* 4.3 Write property test for upstream data transparency (Property 7)
    - **Property 7: Upstream Data Transparency**
    - Generate random non-nil TechnicalOutput and FundamentalOutput
    - Assert DecisionOutput fields exactly match upstream: TechSignal, TechConf, TechReason, FundSentiment, FundConf, FundReason, Pair, RiskPct
    - **Validates: Requirements 7.1, 7.2, 7.6, 10.4**

  - [ ]* 4.4 Write property test for HOLD signal zeroes risk parameters (Property 8)
    - **Property 8: HOLD Signal Zeroes Risk Parameters**
    - Generate inputs that produce HOLD signal (weighted score between thresholds)
    - Assert Entry=0.0, StopLoss=0.0, TakeProfit=0.0, LotSize=0.0
    - **Validates: Requirements 7.5, 10.3**

  - [ ]* 4.5 Write property test for BUY/SELL signal passes risk parameters (Property 9)
    - **Property 9: BUY/SELL Signal Passes Risk Parameters**
    - Generate inputs that produce BUY or SELL signal with non-nil RiskOutput
    - Assert StopLoss==Risk.StopLoss, TakeProfit==Risk.TakeProfit, LotSize==Risk.LotSize, Entry==last candle close
    - Also test with nil RiskOutput → all four are 0.0
    - **Validates: Requirements 7.4, 10.1, 10.2**

  - [ ]* 4.6 Write property test for guaranteed valid output (Property 10)
    - **Property 10: Guaranteed Valid Output Invariant**
    - Generate random AgentInput with any combination of nil/non-nil upstream outputs
    - Assert AgentOutput has Success=true and non-nil Decision pointer for all non-cancelled contexts
    - **Validates: Requirements 8.1, 8.5**

- [ ] 5. Write unit tests for DecisionAgent
  - [ ]* 5.1 Write unit tests in `internal/agents/decision_agent_test.go`
    - Test `NewDecisionAgent` with nil mlClient accepts nil gracefully (Req 1.2)
    - Test `Run` with cancelled context returns Success=false, nil Decision (Req 9.1, 9.3)
    - Test `Run` with nil Technical → TechSignal="HOLD", TechConf=0.0 (Req 7.7, 8.2)
    - Test `Run` with nil Fundamental → FundSentiment="neutral", FundConf=0.5 (Req 7.3)
    - Test `Run` with both nil → HOLD, confidence=0.5, RiskLevel="HIGH" (Req 8.2, 8.3)
    - Test `Run` with ML timeout → proceeds without boost, MLScore=0.0 (Req 5.3, 5.5)
    - Test `Run` with ML returning valid score → MLScore populated (Req 5.4)
    - Test `Run` with ML disabled (nil client) → MLScore=0.0 (Req 5.5)
    - Test `Run` ML context cancel doesn't abort agent (Req 9.4)
    - _Requirements: 1.2, 5.3, 5.4, 5.5, 7.3, 7.7, 8.2, 8.3, 9.1, 9.3, 9.4_

- [x] 6. Checkpoint - Ensure all DecisionAgent tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Wire DecisionAgent into pipeline
  - [x] 7.1 Update `config/config.yaml` with signal configuration section
    - Add `signal` section with buy_threshold, sell_threshold, tech_weight, fund_weight, ml_boost_weight
    - Add `ml` section with enabled flag and timeout setting
    - _Requirements: 2.2, 3.4, 5.1_

  - [x] 7.2 Update `cmd/main.go` to construct and register DecisionAgent
    - Read signal config from config.yaml
    - Create MLPredictor client (or nil if ML disabled)
    - Call `NewDecisionAgent(config, mlClient)` to instantiate
    - Register DecisionAgent in the pipeline after RiskAgent
    - Ensure DecisionAgent output is passed to downstream agents (WhatsAppAgent)
    - _Requirements: 1.1, 1.2, 5.1_

- [x] 8. Final checkpoint - Ensure all tests pass and pipeline integration works
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties using `pgregory.net/rapid`
- Unit tests validate specific examples and edge cases
- The MLPredictor interface enables testing without a real gRPC service
- Implementation follows established patterns: compile-time interface check, context cancellation first, `errorOutput` helper
- Task 4.1 creates the file; tasks 1.2 and 2.x add functions to the same file — wave ordering ensures no conflicts

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "2.1", "2.2", "2.3", "2.4", "2.5"] },
    { "id": 2, "tasks": ["1.3", "2.6", "2.7", "2.8", "2.9", "2.10"] },
    { "id": 3, "tasks": ["4.1"] },
    { "id": 4, "tasks": ["4.2"] },
    { "id": 5, "tasks": ["4.3", "4.4", "4.5", "4.6", "5.1"] },
    { "id": 6, "tasks": ["7.1", "7.2"] }
  ]
}
```
