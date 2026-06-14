import { useEffect, useState, useRef } from 'react'
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
import { ChevronLeft, MessageSquare, Brain } from 'lucide-react'

export function MainLayout() {
  const activePair          = useConnectionStore((s) => s.activePair)
  const setPipelineRunning  = useConnectionStore((s) => s.setPipelineRunning)
  const setStatus           = useConnectionStore((s) => s.setStatus)
  const setLastMessage      = useConnectionStore((s) => s.setLastMessage)

  // Collapse and resize state for the right sidebar
  const [isRightPanelCollapsed, setIsRightPanelCollapsed] = useState(() => {
    return localStorage.getItem('right-panel-collapsed') === 'true'
  })
  const [rightPanelWidth, setRightPanelWidth] = useState(() => {
    const saved = localStorage.getItem('right-panel-width')
    return saved ? parseInt(saved, 10) : 380
  })

  useEffect(() => {
    localStorage.setItem('right-panel-collapsed', String(isRightPanelCollapsed))
  }, [isRightPanelCollapsed])

  useEffect(() => {
    localStorage.setItem('right-panel-width', String(rightPanelWidth))
  }, [rightPanelWidth])

  const startX = useRef(0)
  const startWidth = useRef(380)
  const isDragging = useRef(false)

  const onMouseDownWidth = (e: React.MouseEvent) => {
    e.preventDefault()
    isDragging.current = true
    startX.current = e.clientX
    startWidth.current = rightPanelWidth
    document.body.style.cursor = 'ew-resize'
    document.body.style.userSelect = 'none'
  }

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!isDragging.current) return
      const deltaX = startX.current - e.clientX
      const newW = Math.min(600, Math.max(260, startWidth.current + deltaX))
      setRightPanelWidth(newW)
    }
    const onMouseUp = () => {
      if (!isDragging.current) return
      isDragging.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    return () => {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
  }, [])

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
        {isRightPanelCollapsed ? (
          <button
            onClick={() => setIsRightPanelCollapsed(false)}
            className="flex flex-col items-center py-4 gap-6 w-12 border-l border-border-subtle bg-bg-secondary text-text-secondary hover:text-text-primary hover:bg-bg-tertiary transition-colors cursor-pointer flex-shrink-0"
            title="Expand Agent Debate & Knowledge"
          >
            <ChevronLeft size={16} className="text-text-secondary" />
            <div className="flex flex-col items-center gap-4">
              <MessageSquare size={16} className="text-[#58a6ff]" />
              <Brain size={16} className="text-[#56d364]" />
            </div>
            <span className="text-[10px] tracking-wider uppercase [writing-mode:vertical-lr] select-none font-medium mt-2">
              Agent Debate
            </span>
          </button>
        ) : (
          <div
            className="flex flex-col border-l border-border-subtle bg-bg-secondary overflow-hidden flex-shrink-0 relative"
            style={{ width: rightPanelWidth }}
          >
            {/* Drag Handle */}
            <div
              className="absolute left-0 top-0 bottom-0 w-1.5 cursor-ew-resize hover:bg-agent-technical/45 active:bg-agent-technical transition-colors z-30 group"
              onMouseDown={onMouseDownWidth}
              title="Drag to resize | Double-click to collapse"
              onDoubleClick={() => setIsRightPanelCollapsed(true)}
            >
              <div className="w-[1px] h-full bg-transparent group-hover:bg-agent-technical mx-auto transition-colors" />
            </div>

            {/* Agent debate panel (scrollable) */}
            <div className="flex-1 min-h-0 overflow-hidden flex flex-col">
              <AgentDebatePanel onCollapseToggle={() => setIsRightPanelCollapsed(true)} />
            </div>
            {/* Knowledge panel (fixed at bottom) */}
            <KnowledgePanel />
          </div>
        )}
      </div>

      {/* History panel (resizable) */}
      <HistoryPanel />
    </div>
  )
}
