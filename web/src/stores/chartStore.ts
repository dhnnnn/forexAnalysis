import { create } from 'zustand'
import type { CandleData } from '../types/candle'

const MAX_CANDLES = 500

interface ChartStore {
  candles: Record<string, CandleData[]> // key: "pair:timeframe"
  addCandle: (candle: CandleData) => void
  setCandles: (pair: string, timeframe: string, candles: CandleData[]) => void
}

export const useChartStore = create<ChartStore>((set) => ({
  candles: {},

  addCandle: (candle) =>
    set((state) => {
      const key = `${candle.pair}:${candle.timeframe}`
      const existing = state.candles[key] ?? []
      if (existing.length > 0 && existing[existing.length - 1].timestamp === candle.timestamp) {
        const updated = [...existing]
        updated[updated.length - 1] = candle
        return { candles: { ...state.candles, [key]: updated } }
      }
      const updated = [...existing, candle].slice(-MAX_CANDLES)
      return { candles: { ...state.candles, [key]: updated } }
    }),

  setCandles: (pair, timeframe, candles) =>
    set((state) => {
      const key = `${pair}:${timeframe}`
      return {
        candles: { ...state.candles, [key]: candles.slice(-MAX_CANDLES) },
      }
    }),
}))
