import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { SignalEntry, SystemLog, HistoryTab } from '../types/history'
import type { AgentPerformanceSummary, PerformanceLog } from '../types/agent'
import type { KnowledgeRule } from '../types/knowledge'
import type { RegimeChange, RegimeContext } from '../types/regime'

interface HistoryStore {
  // Signals tab
  signals: SignalEntry[]
  addSignal: (signal: SignalEntry) => void
  setSignals: (signals: SignalEntry[]) => void

  // Performance tab
  agentSummaries: AgentPerformanceSummary[]
  performanceLogs: PerformanceLog[]
  setAgentSummaries: (summaries: AgentPerformanceSummary[]) => void
  addPerformanceLog: (log: PerformanceLog) => void

  // Rules tab
  activeRules: KnowledgeRule[]
  expiredRules: KnowledgeRule[]
  setHistoryActiveRules: (rules: KnowledgeRule[]) => void
  setHistoryExpiredRules: (rules: KnowledgeRule[]) => void

  // Regime tab
  regimeChanges: RegimeChange[]
  regimeHistory: Record<string, RegimeContext[]>
  setRegimeChanges: (changes: RegimeChange[]) => void
  setRegimeHistory: (pair: string, history: RegimeContext[]) => void

  // System log tab
  logs: SystemLog[]
  logFilter: { level: string; search: string }
  addLog: (log: SystemLog) => void
  setLogs: (logs: SystemLog[]) => void
  setLogFilter: (filter: Partial<{ level: string; search: string }>) => void

  // Active tab state
  activeTab: HistoryTab
  setActiveTab: (tab: HistoryTab) => void

  // Panel state
  panelHeight: number
  isCollapsed: boolean
  setPanelHeight: (h: number) => void
  toggleCollapse: () => void
}

export const useHistoryStore = create<HistoryStore>()(
  persist(
    (set) => ({
      signals: [],
      addSignal: (signal) =>
        set((s) => ({ signals: [signal, ...s.signals].slice(0, 200) })),
      setSignals: (signals) => set({ signals }),

      agentSummaries: [],
      performanceLogs: [],
      setAgentSummaries: (agentSummaries) => set({ agentSummaries }),
      addPerformanceLog: (log) =>
        set((s) => ({ performanceLogs: [log, ...s.performanceLogs].slice(0, 200) })),

      activeRules: [],
      expiredRules: [],
      setHistoryActiveRules: (activeRules) => set({ activeRules }),
      setHistoryExpiredRules: (expiredRules) => set({ expiredRules }),

      regimeChanges: [],
      regimeHistory: {},
      setRegimeChanges: (regimeChanges) => set({ regimeChanges }),
      setRegimeHistory: (pair, history) =>
        set((s) => ({ regimeHistory: { ...s.regimeHistory, [pair]: history } })),

      logs: [],
      logFilter: { level: '', search: '' },
      addLog: (log) =>
        set((s) => ({ logs: [log, ...s.logs].slice(0, 500) })),
      setLogs: (logs) => set({ logs }),
      setLogFilter: (filter) =>
        set((s) => ({ logFilter: { ...s.logFilter, ...filter } })),

      activeTab: 'signals',
      setActiveTab: (activeTab) => set({ activeTab }),

      panelHeight: 280,
      isCollapsed: false,
      setPanelHeight: (panelHeight) => set({ panelHeight }),
      toggleCollapse: () => set((s) => ({ isCollapsed: !s.isCollapsed })),
    }),
    {
      name: 'forex-history-store',
      partialize: (state) => ({
        activeTab: state.activeTab,
        panelHeight: state.panelHeight,
        isCollapsed: state.isCollapsed,
        logFilter: state.logFilter,
      }),
    }
  )
)
