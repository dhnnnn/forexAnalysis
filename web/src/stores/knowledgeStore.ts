import { create } from 'zustand'
import type { KnowledgeRule } from '../types/knowledge'
import type { MarketRegime, AdaptiveWeights } from '../types/regime'

interface KnowledgeStore {
  activeRules: KnowledgeRule[]
  expiredRules: KnowledgeRule[]
  adaptiveWeights: Record<string, AdaptiveWeights>
  addRule: (rule: KnowledgeRule) => void
  setActiveRules: (rules: KnowledgeRule[]) => void
  setExpiredRules: (rules: KnowledgeRule[]) => void
  setWeights: (pair: string, weights: AdaptiveWeights) => void
}

export const useKnowledgeStore = create<KnowledgeStore>((set) => ({
  activeRules: [],
  expiredRules: [],
  adaptiveWeights: {},

  addRule: (rule) =>
    set((state) => ({
      activeRules: [rule, ...state.activeRules].slice(0, 50),
    })),

  setActiveRules: (rules) => set({ activeRules: rules }),
  setExpiredRules: (rules) => set({ expiredRules: rules }),

  setWeights: (pair, weights) =>
    set((state) => ({
      adaptiveWeights: { ...state.adaptiveWeights, [pair]: weights },
    })),
}))

// Default weights for when no data is available
export const DEFAULT_WEIGHTS: AdaptiveWeights = {
  techWeight: 0.5,
  fundWeight: 0.5,
  rulesApplied: 0,
  regime: 'UNKNOWN' as MarketRegime,
}
