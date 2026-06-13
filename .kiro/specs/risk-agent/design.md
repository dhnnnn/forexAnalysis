# Design Document — RiskAgent

## Overview

RiskAgent (Agent 4) is a pure-math computation agent that takes account parameters and upstream TechnicalOutput, then produces position sizing (LotSize), stop loss, and take profit levels. It follows the established patterns from `technical_agent.go` — context check first, input validation, then calculation. The agent has no external dependencies (no network, no database), making it fully deterministic and unit-testable.

## Architecture

RiskAgent sits in the multi-agent pipeline between TechnicalAgent (Agent 2) and DecisionAgent (Agent 5):

```
TechnicalAgent → RiskAgent → DecisionAgent
     (signal)     (SL/TP/lot)    (final signal)
```

The agent receives `AgentInput` containing:
- `AccountBalance` and `RiskPercent` (account parameters)
- `Technical` pointer (upstream signal direction)
- `Candles` slice (last element provides entry price)

It produces `AgentOutput` with a populated `Risk` field containing `RiskOutput`.

## Components and Interfaces

### Package-Level Constants

```go
const (
    DefaultSLPips  = 20.0   // Stop loss distance in pips
    DefaultTPPips  = 40.0   // Take profit distance in pips (1:2 RR)
    PipValuePerLot = 10.0   // USD per pip for 1 standard lot (major pairs)
    PipSize        = 0.0001 // Price movement per pip (major pairs)
)
```

### RiskAgent Struct

```go
type RiskAgent struct{}

func NewRiskAgent() *RiskAgent {
    return &RiskAgent{}
}
```

The struct is stateless — all parameters come from `AgentInput` or package constants. This matches the pattern where `TechnicalAgent` is also an empty struct.

### Run Method — Control Flow

```go
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

    // 6. Calculate lot size, SL, TP
    entry := input.Candles[len(input.Candles)-1].Close
    riskAmount := input.AccountBalance * (riskPct / 100.0)
    lotSize := math.Round((riskAmount/(DefaultSLPips*PipValuePerLot))*100) / 100

    var sl, tp float64
    switch input.Technical.Signal {
    case "BUY":
        sl = entry - (DefaultSLPips * PipSize)
        tp = entry + (DefaultTPPips * PipSize)
    case "SELL":
        sl = entry + (DefaultSLPips * PipSize)
        tp = entry - (DefaultTPPips * PipSize)
    }

    round5 := func(v float64) float64 { return math.Round(v*100000) / 100000 }

    // 7. Return populated RiskOutput
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
```

### Agent Interface Implementation

RiskAgent implements the `Agent` interface defined in `agent.go`:

```go
type Agent interface {
    Name() string
    Run(ctx context.Context, input AgentInput) AgentOutput
}
```

Compile-time satisfaction check:

```go
var _ Agent = (*RiskAgent)(nil)
```

## Data Models

### Input Dependencies

| Field | Source | Required |
|-------|--------|----------|
| `AgentInput.AccountBalance` | Pipeline config | Yes, must be > 0 |
| `AgentInput.RiskPercent` | Pipeline config | No, defaults to 1.0 |
| `AgentInput.Technical` | TechnicalAgent output | Yes, must be non-nil |
| `AgentInput.Technical.Signal` | TechnicalAgent | Yes ("BUY"/"SELL"/"HOLD") |
| `AgentInput.Candles` | MarketDataAgent | Yes, at least 1 candle |

### Output — RiskOutput

| Field | Type | Description |
|-------|------|-------------|
| `LotSize` | float64 | Position size, rounded to 2 decimal places |
| `StopLoss` | float64 | SL price level, rounded to 5 decimal places |
| `TakeProfit` | float64 | TP price level, rounded to 5 decimal places |
| `SLPips` | float64 | Stop loss distance (always 20.0) |
| `TPPips` | float64 | Take profit distance (always 40.0) |
| `RiskAmount` | float64 | Dollar amount at risk |

## Formulas

### Lot Size
```
RiskAmount = AccountBalance × (RiskPercent / 100)
LotSize    = round2(RiskAmount / (SLPips × PipValuePerLot))
           = round2(RiskAmount / 200)
```

### Stop Loss
```
BUY:  StopLoss = round5(EntryPrice - SLPips × PipSize) = round5(Entry - 0.0020)
SELL: StopLoss = round5(EntryPrice + SLPips × PipSize) = round5(Entry + 0.0020)
```

### Take Profit
```
BUY:  TakeProfit = round5(EntryPrice + TPPips × PipSize) = round5(Entry + 0.0040)
SELL: TakeProfit = round5(EntryPrice - TPPips × PipSize) = round5(Entry - 0.0040)
```

## Error Handling

| Condition | Response |
|-----------|----------|
| Context cancelled | `errorOutput("RiskAgent", fmt.Errorf("context cancelled: %w", ctx.Err()))` |
| AccountBalance ≤ 0 | `errorOutput("RiskAgent", fmt.Errorf("invalid balance: %.2f", balance))` |
| Technical == nil | `errorOutput("RiskAgent", fmt.Errorf("technical output required to determine direction"))` |
| Signal == "HOLD" | Success with empty `RiskOutput{}` (not an error) |

All error paths use the shared `errorOutput` helper from `technical_agent.go`, which sets `Success=false`, populates `Error`, sets `AgentName` and `Timestamp`.

## Testing Strategy

Standard table-driven Go unit tests in `risk_agent_test.go`. No property-based testing — the user explicitly requested standard table-driven tests only.

Test cases:
1. **BUY signal** — verify lot size, SL below entry, TP above entry
2. **SELL signal** — verify lot size, SL above entry, TP below entry
3. **HOLD signal** — verify Success=true, empty RiskOutput
4. **Invalid balance (zero)** — verify error
5. **Invalid balance (negative)** — verify error
6. **Nil TechnicalOutput** — verify error
7. **RiskPercent default** — verify RiskPercent=0 uses 1.0%
8. **Context cancellation** — verify error wraps ctx.Err()

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Invalid balance rejection

*For any* AccountBalance that is zero or negative, the RiskAgent SHALL return an AgentOutput with Success=false and an Error message containing that balance value formatted to 2 decimal places.

**Validates: Requirements 3.1, 3.2**

### Property 2: Lot size formula correctness

*For any* valid AgentInput with positive AccountBalance, positive-or-defaulted RiskPercent, and a BUY or SELL TechnicalOutput signal, the resulting LotSize SHALL equal `round2((balance × effectiveRiskPct / 100) / (20 × 10))` where effectiveRiskPct is 1.0 when the provided RiskPercent is ≤ 0, otherwise the provided value.

**Validates: Requirements 6.1, 6.2, 7.1, 7.2, 7.3, 7.4**

### Property 3: Stop loss directional correctness

*For any* valid AgentInput with a BUY signal, the StopLoss SHALL equal `round5(lastCandle.Close - 20 × 0.0001)`, and for any SELL signal, the StopLoss SHALL equal `round5(lastCandle.Close + 20 × 0.0001)`, where round5 rounds to exactly 5 decimal places.

**Validates: Requirements 8.1, 9.1, 9.2, 9.3, 9.4, 9.5**

### Property 4: Take profit directional correctness

*For any* valid AgentInput with a BUY signal, the TakeProfit SHALL equal `round5(lastCandle.Close + 40 × 0.0001)`, and for any SELL signal, the TakeProfit SHALL equal `round5(lastCandle.Close - 40 × 0.0001)`, where round5 rounds to exactly 5 decimal places.

**Validates: Requirements 8.1, 10.1, 10.2, 10.3, 10.4**

### Property 5: Successful output completeness

*For any* valid AgentInput with a BUY or SELL signal, the AgentOutput SHALL have Success=true, AgentName="RiskAgent", a non-zero Timestamp, and a non-nil Risk field where LotSize > 0, StopLoss > 0, TakeProfit > 0, SLPips=20, TPPips=40, and RiskAmount > 0.

**Validates: Requirements 11.1, 11.2, 11.3, 11.4**
