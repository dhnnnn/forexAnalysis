# Requirements Document

## Introduction

RiskAgent (Agent 4) calculates position sizing, stop loss, and take profit levels for the forex multi-agent pipeline. The agent receives technical analysis output and account parameters, then produces risk management values that downstream agents (DecisionAgent) use for final signal generation. The agent implements the standard `Agent` interface (`Name()` + `Run()`) and follows the existing patterns established in `technical_agent.go`.

## Glossary

- **RiskAgent**: The Go struct implementing the Agent interface responsible for position sizing and SL/TP calculation
- **Agent_Interface**: The contract (`Name() string` + `Run(ctx, AgentInput) AgentOutput`) that all agents in the pipeline must implement
- **LotSize**: The trading position size calculated based on account balance, risk percentage, stop loss distance, and pip value
- **StopLoss**: The price level at which a losing trade is closed to limit downside
- **TakeProfit**: The price level at which a winning trade is closed to lock in profit
- **SLPips**: Stop loss distance expressed in pips (default: 20)
- **TPPips**: Take profit distance expressed in pips (default: 40, maintaining 1:2 risk-reward ratio)
- **PipValue**: The monetary value of one pip for one standard lot (10 USD for major pairs)
- **PipSize**: The price movement representing one pip (0.0001 for major pairs)
- **EntryPrice**: The price at which a trade would be entered, taken from the last candle's Close price
- **RiskPercent**: The percentage of account balance risked per trade (default fallback: 1.0%)
- **AgentInput**: The shared input container carrying candles, account parameters, and upstream agent outputs
- **AgentOutput**: The shared output container carrying agent results including success status and error information
- **RiskOutput**: The struct holding calculated lot size, stop loss price, take profit price, SL pips, TP pips, and risk amount
- **TechnicalOutput**: The upstream agent output providing signal direction (BUY/SELL/HOLD)
- **HOLD_Signal**: A technical signal indicating no directional trade; RiskAgent returns an empty RiskOutput with success

## Requirements

### Requirement 1: Agent Interface Compliance

**User Story:** As a pipeline orchestrator, I want RiskAgent to implement the Agent interface, so that it integrates seamlessly into the multi-agent pipeline.

#### Acceptance Criteria

1. THE RiskAgent SHALL implement the `Name()` method returning the string "RiskAgent"
2. THE RiskAgent SHALL implement the `Run(ctx context.Context, input AgentInput) AgentOutput` method matching the Agent_Interface contract
3. THE RiskAgent SHALL reside in the `agents` package at `internal/agents/risk_agent.go`

### Requirement 2: Context Cancellation Handling

**User Story:** As a pipeline orchestrator, I want RiskAgent to respect context cancellation, so that the system can gracefully shut down or time out.

#### Acceptance Criteria

1. WHEN the context is cancelled before processing begins, THE RiskAgent SHALL return an AgentOutput with Success set to false and an Error wrapping the context error
2. WHEN the context is cancelled before processing begins, THE RiskAgent SHALL use the errorOutput helper function following the TechnicalAgent pattern

### Requirement 3: Input Validation — Account Balance

**User Story:** As a risk manager, I want RiskAgent to reject invalid account balances, so that nonsensical lot sizes are never calculated.

#### Acceptance Criteria

1. IF AccountBalance in AgentInput is zero or negative, THEN THE RiskAgent SHALL return an AgentOutput with Success set to false
2. IF AccountBalance in AgentInput is zero or negative, THEN THE RiskAgent SHALL return an Error containing the invalid balance value formatted to 2 decimal places

### Requirement 4: Input Validation — Technical Output Dependency

**User Story:** As a pipeline orchestrator, I want RiskAgent to require TechnicalOutput, so that the agent always has a direction for SL/TP calculation.

#### Acceptance Criteria

1. IF the Technical field in AgentInput is nil, THEN THE RiskAgent SHALL return an AgentOutput with Success set to false
2. IF the Technical field in AgentInput is nil, THEN THE RiskAgent SHALL return an Error stating that technical output is required to determine direction

### Requirement 5: HOLD Signal Handling

**User Story:** As a pipeline orchestrator, I want RiskAgent to handle HOLD signals gracefully, so that no unnecessary risk calculations occur when no trade is signaled.

#### Acceptance Criteria

1. WHEN the TechnicalOutput Signal is "HOLD", THE RiskAgent SHALL return an AgentOutput with Success set to true
2. WHEN the TechnicalOutput Signal is "HOLD", THE RiskAgent SHALL set the Risk field to an empty RiskOutput struct (all zero values)
3. WHEN the TechnicalOutput Signal is "HOLD", THE RiskAgent SHALL skip lot size, stop loss, and take profit calculations

### Requirement 6: RiskPercent Default Fallback

**User Story:** As a trader, I want a sensible default risk percentage when none is provided, so that the agent can always produce valid output.

#### Acceptance Criteria

1. IF RiskPercent in AgentInput is zero or negative, THEN THE RiskAgent SHALL use a default value of 1.0 percent for calculations
2. WHEN RiskPercent in AgentInput is greater than zero, THE RiskAgent SHALL use the provided RiskPercent value

### Requirement 7: Lot Size Calculation

**User Story:** As a trader, I want the lot size calculated from my balance and risk parameters, so that each trade risks the correct amount.

#### Acceptance Criteria

1. THE RiskAgent SHALL calculate RiskAmount as AccountBalance multiplied by RiskPercent divided by 100
2. THE RiskAgent SHALL calculate LotSize as RiskAmount divided by the product of SLPips and PipValuePerLot
3. THE RiskAgent SHALL round LotSize to exactly 2 decimal places
4. THE RiskAgent SHALL use the constant PipValuePerLot with value 10.0 in lot size calculation

### Requirement 8: Entry Price Determination

**User Story:** As a trader, I want the entry price derived from the latest candle, so that SL/TP calculations reflect current market price.

#### Acceptance Criteria

1. THE RiskAgent SHALL use the Close price of the last element in the Candles slice of AgentInput as the EntryPrice

### Requirement 9: Stop Loss Calculation by Direction

**User Story:** As a trader, I want stop loss placed correctly relative to entry based on trade direction, so that downside is properly bounded.

#### Acceptance Criteria

1. WHEN the TechnicalOutput Signal is "BUY", THE RiskAgent SHALL calculate StopLoss as EntryPrice minus the product of SLPips and PipSize
2. WHEN the TechnicalOutput Signal is "SELL", THE RiskAgent SHALL calculate StopLoss as EntryPrice plus the product of SLPips and PipSize
3. THE RiskAgent SHALL use the constant DefaultSLPips with value 20.0 as SLPips
4. THE RiskAgent SHALL use the constant PipSize with value 0.0001 in StopLoss calculation
5. THE RiskAgent SHALL round the StopLoss result to exactly 5 decimal places

### Requirement 10: Take Profit Calculation by Direction

**User Story:** As a trader, I want take profit placed correctly relative to entry based on trade direction, so that the 1:2 risk-reward ratio is maintained.

#### Acceptance Criteria

1. WHEN the TechnicalOutput Signal is "BUY", THE RiskAgent SHALL calculate TakeProfit as EntryPrice plus the product of TPPips and PipSize
2. WHEN the TechnicalOutput Signal is "SELL", THE RiskAgent SHALL calculate TakeProfit as EntryPrice minus the product of TPPips and PipSize
3. THE RiskAgent SHALL use the constant DefaultTPPips with value 40.0 as TPPips
4. THE RiskAgent SHALL round the TakeProfit result to exactly 5 decimal places

### Requirement 11: RiskOutput Population

**User Story:** As a downstream agent, I want a fully populated RiskOutput, so that DecisionAgent has all necessary risk parameters for the final signal.

#### Acceptance Criteria

1. WHEN calculation succeeds for a BUY or SELL signal, THE RiskAgent SHALL populate RiskOutput with LotSize, StopLoss, TakeProfit, SLPips, TPPips, and RiskAmount
2. WHEN calculation succeeds, THE RiskAgent SHALL return AgentOutput with Success set to true and the Risk field pointing to the populated RiskOutput
3. THE RiskAgent SHALL set AgentName to "RiskAgent" in every returned AgentOutput
4. THE RiskAgent SHALL set Timestamp to the current time in every returned AgentOutput

### Requirement 12: Hardcoded Constants

**User Story:** As a developer, I want risk constants defined as package-level Go constants, so that they are discoverable and easily adjustable in future iterations.

#### Acceptance Criteria

1. THE RiskAgent SHALL define DefaultSLPips as a package-level constant with value 20.0
2. THE RiskAgent SHALL define DefaultTPPips as a package-level constant with value 40.0
3. THE RiskAgent SHALL define PipValuePerLot as a package-level constant with value 10.0
4. THE RiskAgent SHALL define PipSize as a package-level constant with value 0.0001

### Requirement 13: Unit Test Coverage

**User Story:** As a developer, I want comprehensive table-driven unit tests, so that all calculation paths and edge cases are verified.

#### Acceptance Criteria

1. THE risk_agent_test.go file SHALL contain table-driven tests covering BUY signal lot size and SL/TP calculation
2. THE risk_agent_test.go file SHALL contain table-driven tests covering SELL signal lot size and SL/TP calculation
3. THE risk_agent_test.go file SHALL contain table-driven tests covering HOLD signal returning empty RiskOutput with Success true
4. THE risk_agent_test.go file SHALL contain table-driven tests covering invalid AccountBalance returning an error
5. THE risk_agent_test.go file SHALL contain table-driven tests covering nil TechnicalOutput returning an error
6. THE risk_agent_test.go file SHALL contain table-driven tests covering RiskPercent default fallback to 1.0 percent
7. THE risk_agent_test.go file SHALL contain table-driven tests covering context cancellation returning an error
