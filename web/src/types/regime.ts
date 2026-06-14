// ── Regime Types ──────────────────────────────────────────────────────────
export type MarketRegime =
  | 'TRENDING'
  | 'RANGING'
  | 'BREAKOUT'
  | 'HIGH_VOL'
  | 'LOW_VOL'
  | 'UNKNOWN'

export interface RegimeContext {
  pair: string
  regime: MarketRegime
  adx: number
  atr: number
  volatility: number
  trendStrength: number
  detectedAt: string
}

export interface RegimeChange {
  pair: string
  fromRegime: MarketRegime
  toRegime: MarketRegime
  adx: number
  volatility: number
  changedAt: string
}

export interface AdaptiveWeights {
  techWeight: number
  fundWeight: number
  rulesApplied: number
  regime: MarketRegime
}
