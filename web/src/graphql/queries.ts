import { gql } from '@apollo/client'

// ── Candles ───────────────────────────────────────────────────────────────
export const GET_CANDLES = gql`
  query GetCandles($pair: String!, $timeframe: String!, $limit: Int) {
    candles(pair: $pair, timeframe: $timeframe, limit: $limit) {
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

// ── Signals ───────────────────────────────────────────────────────────────
export const GET_SIGNALS = gql`
  query GetSignals($pair: String, $limit: Int) {
    signals(pair: $pair, limit: $limit) {
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
      evalStatus
      evalPrice
      pipsMove
      evalTime
    }
  }
`

// ── Performance ───────────────────────────────────────────────────────────
export const GET_AGENT_SUMMARIES = gql`
  query GetAgentSummaries {
    agentSummaries {
      agentName
      accuracy
      accuracyPrev
      winCount
      lossCount
      lossStreak
      dominantRegime
      history
    }
  }
`

export const GET_PERFORMANCE_LOGS = gql`
  query GetPerformanceLogs($agent: String, $pair: String, $limit: Int) {
    performanceLogs(agent: $agent, pair: $pair, limit: $limit) {
      agentName
      pair
      regime
      signal
      entryPrice
      evalPrice
      correct
      pipsMove
      signalTime
      evalTime
    }
  }
`

// ── Knowledge Rules ───────────────────────────────────────────────────────
export const GET_ACTIVE_RULES = gql`
  query GetActiveRules {
    activeRules {
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

export const GET_EXPIRED_RULES = gql`
  query GetExpiredRules($limit: Int) {
    expiredRules(limit: $limit) {
      id
      sourceAgent
      targetAgent
      regime
      weightDelta
      confidence
      reasoning
      applyCount
      createdAt
      expiresAt
      status
    }
  }
`

export const GET_ADAPTIVE_WEIGHTS = gql`
  query GetAdaptiveWeights($pair: String!) {
    adaptiveWeights(pair: $pair) {
      techWeight
      fundWeight
      rulesApplied
      regime
    }
  }
`

// ── Regime ────────────────────────────────────────────────────────────────
export const GET_CURRENT_REGIME = gql`
  query GetCurrentRegime($pair: String!) {
    currentRegime(pair: $pair) {
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

export const GET_REGIME_CHANGES = gql`
  query GetRegimeChanges($pair: String!, $limit: Int) {
    regimeChanges(pair: $pair, limit: $limit) {
      pair
      fromRegime
      toRegime
      adx
      volatility
      changedAt
    }
  }
`

export const GET_REGIME_HISTORY = gql`
  query GetRegimeHistory($pair: String!, $limit: Int) {
    regimeHistory(pair: $pair, limit: $limit) {
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

// ── System ────────────────────────────────────────────────────────────────
export const GET_LOGS = gql`
  query GetLogs($level: LogLevel, $limit: Int) {
    logs(level: $level, limit: $limit) {
      timestamp
      level
      message
      agent
      pair
    }
  }
`

export const GET_PAIRS = gql`
  query GetPairs {
    pairs
  }
`

export const GET_CONNECTION_STATUS = gql`
  query GetConnectionStatus {
    connectionStatus
  }
`
