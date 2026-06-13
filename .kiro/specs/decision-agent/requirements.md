# Requirements Document

## Introduction

DecisionAgent (Agent 5) is the final decision-making component in the forex multi-agent pipeline. It aggregates outputs from TechnicalAgent (Agent 2), FundamentalAgent (Agent 3), and RiskAgent (Agent 4) to produce a unified trading signal (BUY, SELL, or HOLD) with confidence scoring, risk level assessment, and full upstream data transparency. The agent uses configurable weighted scoring and threshold-based signal determination.

## Glossary

- **Decision_Agent**: The Go struct implementing the Agent interface that produces the final trading decision by aggregating upstream agent outputs
- **Agent_Input**: The AgentInput struct containing upstream outputs (Technical, Fundamental, Risk fields) and the currency pair identifier
- **Agent_Output**: The AgentOutput struct returned by Run() containing Success status, error, and a populated DecisionOutput pointer
- **Decision_Output**: The DecisionOutput struct holding the final signal, confidence, risk parameters, and all upstream data for transparency
- **Technical_Output**: Output from TechnicalAgent containing Signal (BUY/SELL/HOLD), Confidence (0.0–1.0), TechScore, and Reason
- **Fundamental_Output**: Output from FundamentalAgent containing Sentiment (bullish/bearish/neutral), Confidence (0.0–1.0), Score, and Reason
- **Risk_Output**: Output from RiskAgent containing LotSize, StopLoss, TakeProfit values
- **Weighted_Score**: A composite score calculated as (TechScore × technical_weight) + (FundamentalScore × fundamental_weight)
- **Signal_Config**: Configuration parameters including buy_threshold (0.65), sell_threshold (0.35), and weights (technical: 0.60, fundamental: 0.40)
- **ML_Service**: Optional gRPC-based machine learning prediction service that can boost confidence scoring when enabled
- **Risk_Level**: A categorical assessment (LOW/MEDIUM/HIGH) derived from the overall confidence value

## Requirements

### Requirement 1: Agent Interface Compliance

**User Story:** As a pipeline orchestrator, I want DecisionAgent to implement the standard Agent interface, so that it integrates seamlessly into the multi-agent pipeline.

#### Acceptance Criteria

1. THE Decision_Agent SHALL implement the Agent interface with Name() returning "DecisionAgent" and Run(ctx, AgentInput) returning AgentOutput, verified by a compile-time interface satisfaction check
2. THE Decision_Agent SHALL be instantiated via a NewDecisionAgent(config Signal_Config, mlClient ML_Service) constructor that returns a *DecisionAgent satisfying the Agent interface, where mlClient may be nil to indicate ML_Service is not available

### Requirement 2: Weighted Score Calculation

**User Story:** As a trader, I want the decision to be based on a weighted combination of technical and fundamental analysis, so that both chart patterns and market sentiment influence the final signal.

#### Acceptance Criteria

1. WHEN Run is invoked with a non-nil Technical_Output and a non-nil Fundamental_Output, THE Decision_Agent SHALL calculate Weighted_Score as (Technical_Output.TechScore × 0.60) + (Fundamental_Output.Score × 0.40), producing a result in the range 0.0 to 1.0
2. WHERE Signal_Config specifies custom weights, THE Decision_Agent SHALL use the configured technical and fundamental weights instead of defaults, provided both weights are in the range 0.0 to 1.0 and their sum equals 1.0
3. WHEN Fundamental_Output is nil, THE Decision_Agent SHALL use a default fundamental score of 0.50 and default fundamental confidence of 0.50 for Weighted_Score calculation
4. IF Technical_Output is nil, THEN THE Decision_Agent SHALL use a default technical score of 0.50 and default technical confidence of 0.50 for Weighted_Score calculation
5. IF Signal_Config specifies custom weights that do not sum to 1.0 or contain values outside the range 0.0 to 1.0, THEN THE Decision_Agent SHALL fall back to the default weights of 0.60 technical and 0.40 fundamental

### Requirement 3: Signal Determination via Thresholds

**User Story:** As a trader, I want clear threshold-based signal generation, so that trading decisions are consistent and predictable.

#### Acceptance Criteria

1. WHEN Weighted_Score is greater than or equal to the buy_threshold (default 0.65), THE Decision_Agent SHALL produce a BUY signal
2. WHEN Weighted_Score is less than or equal to the sell_threshold (default 0.35), THE Decision_Agent SHALL produce a SELL signal
3. WHEN Weighted_Score is greater than sell_threshold and less than buy_threshold, THE Decision_Agent SHALL produce a HOLD signal
4. WHERE Signal_Config specifies custom thresholds, THE Decision_Agent SHALL use the configured buy_threshold and sell_threshold values only if sell_threshold is less than buy_threshold and both values are within the range 0.0 to 1.0 inclusive
5. IF Signal_Config specifies custom thresholds where sell_threshold is greater than or equal to buy_threshold or either value is outside the range 0.0 to 1.0, THEN THE Decision_Agent SHALL reject the configuration and use the default thresholds (buy_threshold 0.65, sell_threshold 0.35)

### Requirement 4: Confidence Calculation

**User Story:** As a trader, I want an overall confidence score that reflects how strongly the system backs its decision, so that I can gauge signal reliability.

#### Acceptance Criteria

1. WHEN Run is invoked and Technical_Output is available, THE Decision_Agent SHALL calculate overall confidence as (Technical_Output.Confidence × 0.60) + (Fundamental_Output.Confidence × 0.40), yielding a value in the range 0.0 to 1.0
2. IF Fundamental_Output is nil, THEN THE Decision_Agent SHALL use 0.50 as the fundamental confidence in the calculation
3. WHEN the overall confidence value is calculated, THE Decision_Agent SHALL populate ConfPct as the integer percentage of the confidence value (confidence × 100, truncated to integer), producing a value in the range 0 to 100
4. IF Technical_Output is nil, THEN THE Decision_Agent SHALL use 0.50 as the technical confidence in the calculation

### Requirement 5: ML Score Boost (Optional)

**User Story:** As a system operator, I want the ML service to optionally boost decision confidence, so that machine learning predictions can enhance signal quality when available.

#### Acceptance Criteria

1. WHILE ML_Service is enabled and reachable via gRPC within 500 milliseconds, WHEN Run is invoked, THE Decision_Agent SHALL request a prediction from ML_Service
2. WHEN ML_Service returns a score in the range 0.0 to 1.0 and greater than zero, THE Decision_Agent SHALL adjust confidence as (confidence × 0.80) + (ml_score × 0.20)
3. IF ML_Service is disabled, unreachable, returns an error, or does not respond within 500 milliseconds, THEN THE Decision_Agent SHALL proceed with the base confidence calculation without ML adjustment
4. WHEN ML_Service returns a score in the range 0.0 to 1.0, THE Decision_Agent SHALL populate the MLScore field in Decision_Output with the returned prediction value
5. IF ML_Service is disabled or does not return a valid score, THEN THE Decision_Agent SHALL populate the MLScore field in Decision_Output with 0.0

### Requirement 6: Risk Level Assessment

**User Story:** As a trader, I want a risk level classification alongside the signal, so that I can quickly understand how risky the recommended trade is.

#### Acceptance Criteria

1. WHEN the final overall confidence (after any ML_Service adjustment per Requirement 5) is greater than or equal to 0.75, THE Decision_Agent SHALL assign Risk_Level as "LOW"
2. WHEN the final overall confidence is greater than or equal to 0.50 and less than 0.75, THE Decision_Agent SHALL assign Risk_Level as "MEDIUM"
3. WHEN the final overall confidence is less than 0.50, THE Decision_Agent SHALL assign Risk_Level as "HIGH"
4. THE Decision_Agent SHALL populate Risk_Level in Decision_Output for every signal type including HOLD

### Requirement 7: Upstream Data Transparency

**User Story:** As a trader, I want the decision output to include all upstream data, so that I can audit how the final signal was derived.

#### Acceptance Criteria

1. THE Decision_Agent SHALL populate TechSignal from Technical_Output.Signal, TechConf from Technical_Output.Confidence, and TechReason from Technical_Output.Reason in Decision_Output
2. THE Decision_Agent SHALL populate FundSentiment from Fundamental_Output.Sentiment, FundConf from Fundamental_Output.Confidence, and FundReason from Fundamental_Output.Reason in Decision_Output
3. WHEN Fundamental_Output is nil, THE Decision_Agent SHALL populate FundSentiment as "neutral", FundConf as 0.50, and FundReason as empty string
4. WHEN the signal is BUY or SELL, THE Decision_Agent SHALL populate Entry from the last candle close price in Agent_Input.Candles, and StopLoss, TakeProfit, and LotSize from Risk_Output
5. WHEN the signal is HOLD, THE Decision_Agent SHALL populate Entry as 0.0, StopLoss as 0.0, TakeProfit as 0.0, and LotSize as 0.0
6. THE Decision_Agent SHALL populate Pair from Agent_Input.Pair and Timestamp from the system clock at the time Run executes
7. WHEN Technical_Output is nil, THE Decision_Agent SHALL populate TechSignal as "HOLD", TechConf as 0.0, and TechReason as empty string

### Requirement 8: Guaranteed Valid Output

**User Story:** As a pipeline orchestrator, I want DecisionAgent to always return a valid DecisionOutput, so that downstream agents never receive nil decision data.

#### Acceptance Criteria

1. THE Decision_Agent SHALL return a non-nil Decision_Output pointer in Agent_Output for all executions where Success is true
2. WHEN Technical_Output is nil in Agent_Input, THE Decision_Agent SHALL return a HOLD signal with default confidence of 0.50, default Weighted_Score of 0.50, and populate TechSignal as "HOLD", TechConf as 0.50, and TechReason as empty string
3. WHEN both Technical_Output and Fundamental_Output are nil in Agent_Input, THE Decision_Agent SHALL return a HOLD signal with default confidence of 0.50 and Risk_Level "HIGH"
4. IF context is cancelled, THEN THE Decision_Agent SHALL set Success to false and MAY return a nil Decision_Output pointer
5. THE Decision_Agent SHALL set Success to true for all cases where context is not cancelled, including cases where upstream inputs are nil

### Requirement 9: Context Cancellation Handling

**User Story:** As a pipeline orchestrator, I want DecisionAgent to respect context cancellation, so that the system can shut down gracefully without hanging.

#### Acceptance Criteria

1. WHEN the context is cancelled before processing begins, THE Decision_Agent SHALL return Agent_Output with Success set to false, Error containing the context cancellation reason, Decision_Output set to nil, and AgentName set to "DecisionAgent"
2. IF the context is cancelled during processing (after weighted score calculation or after ML_Service call), THEN THE Decision_Agent SHALL abort processing and return Agent_Output with Success set to false, Error containing the context cancellation reason, and Decision_Output set to nil
3. THE Decision_Agent SHALL check context cancellation as the first operation in Run()
4. IF the context is cancelled during an ML_Service call, THEN THE Decision_Agent SHALL treat the ML_Service as unavailable and proceed with base confidence calculation without aborting

### Requirement 10: Risk Parameters Passthrough

**User Story:** As a trader, I want the decision output to include the calculated risk parameters, so that I have a complete trade setup in one message.

#### Acceptance Criteria

1. WHEN Risk_Output is non-nil and the signal is BUY or SELL, THE Decision_Agent SHALL copy LotSize, StopLoss, and TakeProfit from Risk_Output into Decision_Output
2. WHEN Risk_Output is nil and the signal is BUY or SELL, THE Decision_Agent SHALL populate LotSize, StopLoss, and TakeProfit as 0.0
3. WHEN the signal is HOLD, THE Decision_Agent SHALL populate LotSize, StopLoss, and TakeProfit as 0.0 regardless of Risk_Output availability
4. THE Decision_Agent SHALL populate RiskPct in Decision_Output from Agent_Input.RiskPercent, using the value as-is (including 0.0 if RiskPercent is zero or unset)
