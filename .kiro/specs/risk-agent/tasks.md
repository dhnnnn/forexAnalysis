# Implementation Plan: RiskAgent

## Overview

Implement the RiskAgent (Agent 4) as a pure-math computation agent following the established TechnicalAgent pattern. The agent calculates position sizing, stop loss, and take profit levels from account parameters and upstream TechnicalOutput. Implementation is straightforward: one file for the agent, one file for tests, then verify.

## Tasks

- [x] 1. Implement RiskAgent
  - [x] 1.1 Create `internal/agents/risk_agent.go` with RiskAgent struct and Run method
    - Define package-level constants: `DefaultSLPips=20.0`, `DefaultTPPips=40.0`, `PipValuePerLot=10.0`, `PipSize=0.0001`
    - Create `RiskAgent` struct (empty, stateless) and `NewRiskAgent()` constructor
    - Add compile-time interface check: `var _ Agent = (*RiskAgent)(nil)`
    - Implement `Name()` returning `"RiskAgent"`
    - Implement `Run(ctx, input)` with control flow: context check → balance validation → nil Technical check → HOLD early return → RiskPercent default → lot/SL/TP calculation → return populated AgentOutput
    - Use `errorOutput` helper from `technical_agent.go` for all error paths
    - Round LotSize to 2 decimal places, StopLoss and TakeProfit to 5 decimal places
    - _Requirements: 1.1, 1.2, 1.3, 2.1, 2.2, 3.1, 3.2, 4.1, 4.2, 5.1, 5.2, 5.3, 6.1, 6.2, 7.1, 7.2, 7.3, 7.4, 8.1, 9.1, 9.2, 9.3, 9.4, 9.5, 10.1, 10.2, 10.3, 10.4, 11.1, 11.2, 11.3, 11.4, 12.1, 12.2, 12.3, 12.4_

- [x] 2. Write unit tests for RiskAgent
  - [x] 2.1 Create `internal/agents/risk_agent_test.go` with table-driven tests
    - Test BUY signal: verify LotSize, StopLoss below entry, TakeProfit above entry, all fields populated
    - Test SELL signal: verify LotSize, StopLoss above entry, TakeProfit below entry, all fields populated
    - Test HOLD signal: verify Success=true, empty RiskOutput (zero values)
    - Test invalid balance (zero): verify Success=false, error contains "0.00"
    - Test invalid balance (negative): verify Success=false, error contains the negative value
    - Test nil TechnicalOutput: verify Success=false, error mentions "technical output required"
    - Test RiskPercent default: verify RiskPercent=0 uses 1.0% (check resulting LotSize matches formula)
    - Test context cancellation: verify Success=false, error wraps context error
    - Use `math.Abs(got-want) < epsilon` for float comparisons with epsilon=1e-5
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 13.7_

- [x] 3. Checkpoint — Verify all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Pure math agent with no external dependencies — fully deterministic and easy to test
- Reuses `errorOutput` helper already defined in `technical_agent.go` (same package)
- Standard table-driven unit tests only, no property-based testing
- Constants are hardcoded per design; future iterations may make them configurable
- All float comparisons in tests should use epsilon-based equality (1e-5)

## Task Dependency Graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["2.1"] }
  ]
}
```
