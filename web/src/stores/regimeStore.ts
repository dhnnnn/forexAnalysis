import { create } from 'zustand'
import type { RegimeContext, RegimeChange } from '../types/regime'

interface RegimeStore {
  currentRegime: Record<string, RegimeContext>
  regimeChanges: RegimeChange[]
  setRegime: (regime: RegimeContext) => void
  addRegimeChange: (change: RegimeChange) => void
}

export const useRegimeStore = create<RegimeStore>((set) => ({
  currentRegime: {},
  regimeChanges: [],

  setRegime: (regime) =>
    set((state) => ({
      currentRegime: { ...state.currentRegime, [regime.pair]: regime },
    })),

  addRegimeChange: (change) =>
    set((state) => ({
      regimeChanges: [change, ...state.regimeChanges].slice(0, 100),
    })),
}))
