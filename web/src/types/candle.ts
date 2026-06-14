// ── Candle ────────────────────────────────────────────────────────────────
export interface CandleData {
  pair: string
  open: number
  high: number
  low: number
  close: number
  volume: number
  spread: number
  timeframe: string
  timestamp: string
}

// For lightweight-charts format
export interface ChartCandle {
  time: number  // Unix timestamp
  open: number
  high: number
  low: number
  close: number
}

export interface ChartVolume {
  time: number
  value: number
  color: string
}

export interface SignalMarker {
  time: number
  position: 'belowBar' | 'aboveBar'
  color: string
  shape: 'arrowUp' | 'arrowDown' | 'circle'
  text: string
}

export interface RegimeBand {
  startTime: number
  endTime: number
  regime: string
}
