import { create } from 'zustand'
import type { CandleData } from '../types/candle'

const MAX_CANDLES = 500

interface ChartStore {
  candles: Record<string, CandleData[]>
  addCandle: (candle: CandleData) => void
  setCandles: (pair: string, candles: CandleData[]) => void
}

export const useChartStore = create<ChartStore>((set) => ({
  candles: {},

  addCandle: (candle) =>
    set((state) => {
      const existing = state.candles[candle.pair] ?? []
      const updated = [...existing, candle].slice(-MAX_CANDLES)
      return { candles: { ...state.candles, [candle.pair]: updated } }
    }),

  setCandles: (pair, candles) =>
    set((state) => ({
      candles: { ...state.candles, [pair]: candles.slice(-MAX_CANDLES) },
    })),
}))
