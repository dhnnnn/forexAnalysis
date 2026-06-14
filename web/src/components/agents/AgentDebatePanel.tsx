import { useSubscription } from '@apollo/client'
import { useEffect, useRef } from 'react'
import { AGENT_OUTPUT } from '../../graphql/subscriptions'
import { useAgentStore } from '../../stores/agentStore'
import { useConnectionStore } from '../../stores/connectionStore'
import { useAutoScroll } from '../../hooks/useAutoScroll'
import { AgentCard } from './AgentCard'
import type { AgentDebateEntry } from '../../types/agent'
import { formatTime } from '../../utils/formatters'
import { MessageSquare, ChevronRight } from 'lucide-react'

interface AgentDebatePanelProps {
  onCollapseToggle?: () => void
}

export function AgentDebatePanel({ onCollapseToggle }: AgentDebatePanelProps) {
  const activePair = useConnectionStore((s) => s.activePair)
  const debates    = useAgentStore((s) => s.debates)
  const addEntry   = useAgentStore((s) => s.addDebateEntry)

  // Filter debates for active pair
  const pairDebates = debates.filter((d) => d.pair === activePair || !d.pair)

  // Subscribe to agent outputs
  const { error } = useSubscription(AGENT_OUTPUT, {
    variables: { pair: activePair },
    onData: ({ data }) => {
      const entry = data.data?.agentOutput as AgentDebateEntry | undefined
      if (entry) addEntry(entry)
    },
  })

  const scrollRef = useAutoScroll([pairDebates.length])

  // Group by cycle timestamp (minutes)
  const groups = groupByCycle(pairDebates)

  return (
    <aside
      className="flex flex-col flex-1 min-h-0 w-full"
      aria-label="Agent Debate Panel"
    >
      {/* Panel header */}
      <div 
        className="flex items-center gap-2 px-4 py-3 border-b border-border-subtle flex-shrink-0 cursor-pointer hover:bg-bg-tertiary select-none group/header transition-colors"
        onClick={onCollapseToggle}
        title="Click to collapse panel"
      >
        <MessageSquare size={14} className="text-[#58a6ff]" />
        <span className="text-sm font-semibold text-text-primary group-hover/header:text-agent-technical transition-colors">
          Agent Debate
        </span>
        <span className="ml-auto text-xs text-text-muted group-hover/header:text-text-secondary transition-colors mr-1">
          {pairDebates.length} entries
        </span>
        <ChevronRight size={14} className="text-text-muted group-hover/header:text-text-primary transition-colors" />
      </div>

      {/* Scrollable content */}
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto px-3 py-3 space-y-4"
      >
        {groups.length === 0 && (
          <div className="flex flex-col items-center justify-center h-40 text-text-muted text-xs gap-2">
            <MessageSquare size={24} className="opacity-30" />
            <p>Waiting for pipeline…</p>
            {error && <p className="text-sell-red text-[10px]">{error.message}</p>}
          </div>
        )}

        {groups.map(({ cycleKey, entries }) => (
          <div key={cycleKey} className="space-y-2">
            {/* Cycle timestamp separator */}
            <div className="flex items-center gap-2">
              <div className="flex-1 h-px bg-border-subtle" />
              <span className="text-[10px] font-mono text-text-muted px-2">
                {cycleKey}
              </span>
              <div className="flex-1 h-px bg-border-subtle" />
            </div>

            {entries.map((entry) => (
              <AgentCard key={entry.id} entry={entry} />
            ))}
          </div>
        ))}
      </div>
    </aside>
  )
}

// Group entries by pipeline cycle (minute-level timestamp)
function groupByCycle(entries: AgentDebateEntry[]) {
  const groups: Record<string, AgentDebateEntry[]> = {}
  for (const e of entries) {
    const d = new Date(e.timestamp)
    const key = `${d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} ${d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit' })}`
    if (!groups[key]) groups[key] = []
    groups[key].push(e)
  }
  return Object.entries(groups).map(([cycleKey, entries]) => ({ cycleKey, entries }))
}
