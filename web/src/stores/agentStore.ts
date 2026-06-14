import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AgentDebateEntry } from '../types/agent'

const MAX_ENTRIES = 100

interface AgentStore {
  debates: AgentDebateEntry[]
  addDebateEntry: (entry: AgentDebateEntry) => void
  clearOlderThan: (hours: number) => void
}

export const useAgentStore = create<AgentStore>()(
  persist(
    (set) => ({
      debates: [],

      addDebateEntry: (entry) =>
        set((state) => {
          // Avoid duplicate entries by checking the ID
          const exists = state.debates.some((d) => d.id === entry.id)
          if (exists) return state
          return {
            debates: [...state.debates, entry].slice(-MAX_ENTRIES),
          }
        }),

      clearOlderThan: (hours) =>
        set((state) => {
          const cutoff = Date.now() - hours * 3_600_000
          return {
            debates: state.debates.filter(
              (d) => new Date(d.timestamp).getTime() > cutoff
            ),
          }
        }),
    }),
    {
      name: 'forex-agent-store',
    }
  )
)
