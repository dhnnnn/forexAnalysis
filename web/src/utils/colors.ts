import type { MarketRegime } from '../types/regime'

// ── Agent Colors ──────────────────────────────────────────────────────────
export const AGENT_COLORS: Record<string, string> = {
  TechnicalAgent:   '#58a6ff',
  FundamentalAgent: '#d2a8ff',
  DecisionAgent:    '#ffa657',
  RegimeAgent:      '#79c0ff',
  MetaObserver:     '#f0883e',
  KTA:              '#56d364',
  RiskAgent:        '#ff7b72',
}

export function getAgentColor(agent: string): string {
  return AGENT_COLORS[agent] ?? '#8b949e'
}

export function getAgentInitial(agent: string): string {
  const map: Record<string, string> = {
    TechnicalAgent:   'T',
    FundamentalAgent: 'F',
    DecisionAgent:    'D',
    RegimeAgent:      'R',
    MetaObserver:     'M',
    KTA:              'K',
    RiskAgent:        'Rk',
  }
  return map[agent] ?? agent[0]?.toUpperCase() ?? '?'
}

// ── Regime Colors ─────────────────────────────────────────────────────────
export const REGIME_COLORS: Record<string, string> = {
  TRENDING: '#58a6ff',
  RANGING:  '#8b949e',
  BREAKOUT: '#bc8cff',
  HIGH_VOL: '#f85149',
  LOW_VOL:  '#3fb950',
  UNKNOWN:  '#484f58',
}

export function getRegimeColor(regime: MarketRegime | string): string {
  return REGIME_COLORS[regime?.toUpperCase()] ?? '#484f58'
}

export function getRegimeIcon(regime: MarketRegime | string): string {
  const icons: Record<string, string> = {
    TRENDING: '📈',
    RANGING:  '↔️',
    BREAKOUT: '💥',
    HIGH_VOL: '🌋',
    LOW_VOL:  '🧊',
    UNKNOWN:  '❓',
  }
  return icons[regime?.toUpperCase()] ?? '❓'
}

// ── Signal Colors ─────────────────────────────────────────────────────────
export const SIGNAL_COLORS = {
  BUY:  { text: '#2ea043', bg: 'rgba(46,160,67,0.15)',  border: '#2ea043' },
  SELL: { text: '#f85149', bg: 'rgba(248,81,73,0.15)', border: '#f85149' },
  HOLD: { text: '#d29922', bg: 'rgba(210,153,34,0.15)', border: '#d29922' },
}

export function getSignalColors(signal: string) {
  return SIGNAL_COLORS[signal as keyof typeof SIGNAL_COLORS] ?? SIGNAL_COLORS.HOLD
}

// ── Log Level Colors ──────────────────────────────────────────────────────
export const LOG_LEVEL_COLORS: Record<string, string> = {
  DEBUG: '#484f58',
  INFO:  '#58a6ff',
  WARN:  '#d29922',
  ERROR: '#f85149',
}

export function getLogLevelColor(level: string): string {
  return LOG_LEVEL_COLORS[level] ?? '#8b949e'
}

// ── Candle Colors ─────────────────────────────────────────────────────────
export const CANDLE_UP_COLOR   = '#2ea043'
export const CANDLE_DOWN_COLOR = '#f85149'
