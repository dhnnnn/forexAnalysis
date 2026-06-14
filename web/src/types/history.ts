import type { Signal } from './agent'

// ── History / Signal Entry ────────────────────────────────────────────────
export type EvalStatus = 'PENDING' | 'CORRECT' | 'INCORRECT' | 'SKIPPED'

export interface SignalEntry {
  id: number
  timestamp: string
  pair: string
  signal: Signal
  confidence: number
  regime: string
  entry: number
  stopLoss: number
  takeProfit: number
  lotSize: number
  techSignal: string
  techConf: number
  techReason: string
  fundSentiment: string
  fundConf: number
  fundReason: string
  evalStatus?: EvalStatus | null
  evalPrice?: number | null
  pipsMove?: number | null
  evalTime?: string | null
}

export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR'

export interface SystemLog {
  timestamp: string
  level: LogLevel
  message: string
  agent?: string | null
  pair?: string | null
}

export type HistoryTab = 'signals' | 'performance' | 'rules' | 'regime' | 'system'
