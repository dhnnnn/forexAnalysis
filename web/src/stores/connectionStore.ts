import { create } from 'zustand'

export type ConnectionStatus = 'connected' | 'reconnecting' | 'disconnected'

interface ConnectionStore {
  status: ConnectionStatus
  lastMessage: string
  activePair: string
  timeframe: string
  setStatus: (status: ConnectionStatus) => void
  setLastMessage: (ts: string) => void
  setActivePair: (pair: string) => void
  setTimeframe: (tf: string) => void
  // Pipeline loading state per pair
  pipelineRunning: Record<string, boolean>
  setPipelineRunning: (pair: string, running: boolean) => void
}

export const useConnectionStore = create<ConnectionStore>((set) => ({
  status: 'disconnected',
  lastMessage: '',
  activePair: 'EUR_USD',
  timeframe: '1h',
  pipelineRunning: {},

  setStatus: (status) => set({ status }),
  setLastMessage: (lastMessage) => set({ lastMessage }),
  setActivePair: (activePair) => set({ activePair }),
  setTimeframe: (timeframe) => set({ timeframe }),
  setPipelineRunning: (pair, running) =>
    set((s) => ({ pipelineRunning: { ...s.pipelineRunning, [pair]: running } })),
}))
