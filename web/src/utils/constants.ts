export const TIMEFRAMES = ['1m', '5m', '15m', '30m', '1h', '4h', '1d'] as const
export type Timeframe = typeof TIMEFRAMES[number]

export const DEFAULT_PAIR      = 'EUR_USD'
export const DEFAULT_TIMEFRAME: Timeframe = '1h'
export const DEFAULT_CANDLE_LIMIT = 200
export const MAX_CANDLES          = 500
export const MAX_DEBATE_ENTRIES   = 100
export const MAX_LOGS             = 500

export const AVAILABLE_PAIRS = ['EUR_USD', 'GBP_USD', 'USD_JPY', 'AUD_USD']
