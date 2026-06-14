import { gql } from '@apollo/client'

// ── Candle real-time updates ──────────────────────────────────────────────
export const CANDLE_UPDATED = gql`
  subscription CandleUpdated($pair: String!) {
    candleUpdated(pair: $pair) {
      pair
      open
      high
      low
      close
      volume
      spread
      timeframe
      timestamp
    }
  }
`

// ── Agent debate entries ──────────────────────────────────────────────────
export const AGENT_OUTPUT = gql`
  subscription AgentOutput($pair: String) {
    agentOutput(pair: $pair) {
      id
      timestamp
      pair
      agent
      signal
      confidence
      reasoning
      details {
        rsi
        macdHist
        bbPosition
        ema50
        ema200
        sentiment
        score
        regime
        techWeight
        fundWeight
      }
    }
  }
`

// ── Final trading signal ──────────────────────────────────────────────────
export const SIGNAL_GENERATED = gql`
  subscription SignalGenerated($pair: String) {
    signalGenerated(pair: $pair) {
      id
      timestamp
      pair
      signal
      confidence
      regime
      entry
      stopLoss
      takeProfit
      lotSize
      techSignal
      techConf
      techReason
      fundSentiment
      fundConf
      fundReason
    }
  }
`

// ── Regime changes ────────────────────────────────────────────────────────
export const REGIME_CHANGED = gql`
  subscription RegimeChanged($pair: String) {
    regimeChanged(pair: $pair) {
      pair
      regime
      adx
      atr
      volatility
      trendStrength
      detectedAt
    }
  }
`

// ── Knowledge rules ───────────────────────────────────────────────────────
export const RULE_CREATED = gql`
  subscription RuleCreated {
    ruleCreated {
      id
      sourceAgent
      targetAgent
      regime
      weightDelta
      minWeight
      confidence
      reasoning
      applyCount
      createdAt
      expiresAt
      status
    }
  }
`

// ── System logs ───────────────────────────────────────────────────────────
export const LOG_ADDED = gql`
  subscription LogAdded {
    logAdded {
      timestamp
      level
      message
      agent
      pair
    }
  }
`

// ── Pipeline lifecycle ────────────────────────────────────────────────────
export const PIPELINE_EVENT = gql`
  subscription PipelineEvent($pair: String) {
    pipelineEvent(pair: $pair) {
      type
      pair
      timestamp
      durationMs
    }
  }
`
