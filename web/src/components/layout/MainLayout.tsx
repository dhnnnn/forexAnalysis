import { useEffect } from 'react'
import { useSubscription } from '@apollo/client'
import { PIPELINE_EVENT } from '../../graphql/subscriptions'
import { wsClient } from '../../graphql/apolloClient'
import { useConnectionStore } from '../../stores/connectionStore'
import { Header } from './Header'
import { StatusBar } from './StatusBar'
import { CandlestickChart } from '../chart/CandlestickChart'
import { AgentDebatePanel } from '../agents/AgentDebatePanel'
import { KnowledgePanel } from '../knowledge/KnowledgePanel'
import { HistoryPanel } from '../history/HistoryPanel'
import type { PipelineEvent } from '../../types/knowledge'

export function MainLayout() {
  const activePair          = useConnectionStore((s) => s.activePair)
  const setPipelineRunning  = useConnectionStore((s) => s.setPipelineRunning)
  const setStatus           = useConnectionStore((s) => s.setStatus)
  const setLastMessage      = useConnectionStore((s) => s.setLastMessage)

  // Subscribe to pipeline events to track loading state
  useSubscription(PIPELINE_EVENT, {
    variables: { pair: activePair },
    onData: ({ data }) => {
      const event = data.data?.pipelineEvent as PipelineEvent | undefined
      if (!event) return
      setPipelineRunning(event.pair, event.type === 'START')
      setStatus('connected')
      setLastMessage(event.timestamp)
    },
    onError: () => setStatus('reconnecting'),
  })

  // Track WebSocket connection status from graphql-ws client
  useEffect(() => {
    const unsub = wsClient.on('connected', () => setStatus('connected'))
    const unsub2 = wsClient.on('closed', () => setStatus('disconnected'))
    const unsub3 = wsClient.on('connecting', () => setStatus('reconnecting'))
    return () => { unsub(); unsub2(); unsub3() }
  }, [setStatus])

  return (
    <div className="flex flex-col h-full bg-bg-primary overflow-hidden">
      {/* Header */}
      <Header />

      {/* Main content — split horizontally */}
      <div className="flex flex-1 min-h-0">

        {/* Left: Chart area */}
        <main className="flex flex-col flex-1 min-w-0 overflow-hidden">
          {/* Chart + Indicator bar */}
          <div className="flex-1 min-h-0">
            <CandlestickChart />
          </div>

          {/* Status bar */}
          <StatusBar />
        </main>

        {/* Right: Agent debate + Knowledge */}
        <div className="flex flex-col border-l border-border-subtle bg-bg-secondary overflow-hidden flex-shrink-0"
          style={{ width: 380, minWidth: 300 }}>
          {/* Agent debate panel (scrollable) */}
          <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
            <AgentDebatePanel />
          </div>
          {/* Knowledge panel (fixed at bottom) */}
          <KnowledgePanel />
        </div>
      </div>

      {/* History panel (resizable) */}
      <HistoryPanel />
    </div>
  )
}
