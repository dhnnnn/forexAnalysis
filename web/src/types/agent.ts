// ── Agent Types ───────────────────────────────────────────────────────────
export type AgentName =
  | 'TechnicalAgent'
  | 'FundamentalAgent'
  | 'DecisionAgent'
  | 'RegimeAgent'
  | 'MetaObserver'
  | 'KTA'
  | 'RiskAgent'

export type Signal = 'BUY' | 'SELL' | 'HOLD'

export interface AgentDetails {
  rsi?: number | null
  macdHist?: number | null
  bbPosition?: number | null
  ema50?: number | null
  ema200?: number | null
  sentiment?: string | null
  score?: number | null
  regime?: string | null
  techWeight?: number | null
  fundWeight?: number | null
}

export interface AgentDebateEntry {
  id: string
  timestamp: string
  pair: string
  agent: AgentName | string
  signal: Signal
  confidence: number
  reasoning: string
  details?: AgentDetails | null
}

export interface AgentPerformanceSummary {
  agentName: string
  accuracy: number
  accuracyPrev: number
  winCount: number
  lossCount: number
  lossStreak: number
  dominantRegime: string
  history: boolean[]
}

export interface PerformanceLog {
  agentName: string
  pair: string
  regime: string
  signal: Signal
  entryPrice: number
  evalPrice: number
  correct: boolean
  pipsMove: number
  signalTime: string
  evalTime: string
}
