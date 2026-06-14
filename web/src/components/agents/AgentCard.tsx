import { useState } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import type { AgentDebateEntry } from '../../types/agent'
import { getAgentColor, getAgentInitial } from '../../utils/colors'
import { formatTime } from '../../utils/formatters'
import { SignalBadge } from './SignalBadge'
import { ConfidenceBar } from './ConfidenceBar'

interface AgentCardProps {
  entry: AgentDebateEntry
}

export function AgentCard({ entry }: AgentCardProps) {
  const [expanded, setExpanded] = useState(false)
  const color = getAgentColor(entry.agent)
  const initial = getAgentInitial(entry.agent)

  const hasDetails = entry.details && (
    entry.details.rsi !== null ||
    entry.details.macdHist !== null ||
    entry.details.bbPosition !== null
  )

  return (
    <article
      className="agent-card group"
      onClick={() => hasDetails && setExpanded(!expanded)}
      role={hasDetails ? 'button' : undefined}
      aria-expanded={hasDetails ? expanded : undefined}
    >
      {/* Header row */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          {/* Avatar */}
          <div
            className="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold flex-shrink-0 shadow-md"
            style={{ backgroundColor: `${color}28`, color, border: `1.5px solid ${color}55` }}
            aria-label={entry.agent}
          >
            {initial}
          </div>
          {/* Agent name */}
          <div>
            <span className="text-xs font-semibold" style={{ color }}>
              {entry.agent}
            </span>
            {entry.pair && (
              <span className="text-[10px] text-text-muted ml-1.5">
                {entry.pair.replace('_', '/')}
              </span>
            )}
          </div>
        </div>
        {/* Time + expand icon */}
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] font-mono text-text-muted">
            {formatTime(entry.timestamp)}
          </span>
          {hasDetails && (
            <span className="text-text-muted opacity-50 group-hover:opacity-100 transition-opacity">
              {expanded ? <ChevronUp size={12} /> : <ChevronDown size={12} />}
            </span>
          )}
        </div>
      </div>

      {/* Signal + confidence */}
      <div className="flex items-center gap-2 mb-2">
        <SignalBadge signal={entry.signal} />
        <div className="flex-1">
          <ConfidenceBar value={entry.confidence} signal={entry.signal} />
        </div>
      </div>

      {/* Reasoning */}
      <p className="text-xs text-text-secondary leading-relaxed line-clamp-2">
        {entry.reasoning}
      </p>

      {/* Expanded details */}
      {expanded && entry.details && (
        <div className="mt-3 pt-3 border-t border-border-subtle grid grid-cols-2 gap-x-3 gap-y-1 text-[11px] font-mono animate-fade-in-up">
          {entry.details.rsi !== null && entry.details.rsi !== undefined && (
            <>
              <span className="text-text-muted">RSI</span>
              <span className={entry.details.rsi < 30 ? 'text-buy-green' : entry.details.rsi > 70 ? 'text-sell-red' : 'text-text-primary'}>
                {entry.details.rsi.toFixed(1)}
              </span>
            </>
          )}
          {entry.details.macdHist !== null && entry.details.macdHist !== undefined && (
            <>
              <span className="text-text-muted">MACD Hist</span>
              <span className={entry.details.macdHist > 0 ? 'text-buy-green' : 'text-sell-red'}>
                {entry.details.macdHist.toFixed(5)}
              </span>
            </>
          )}
          {entry.details.bbPosition !== null && entry.details.bbPosition !== undefined && (
            <>
              <span className="text-text-muted">BB Position</span>
              <span className="text-text-primary">{(entry.details.bbPosition * 100).toFixed(1)}%</span>
            </>
          )}
          {entry.details.ema50 !== null && entry.details.ema50 !== undefined && (
            <>
              <span className="text-text-muted">EMA 50</span>
              <span className="text-text-primary">{entry.details.ema50.toFixed(5)}</span>
            </>
          )}
          {entry.details.ema200 !== null && entry.details.ema200 !== undefined && (
            <>
              <span className="text-text-muted">EMA 200</span>
              <span className="text-text-primary">{entry.details.ema200.toFixed(5)}</span>
            </>
          )}
          {entry.details.techWeight !== null && entry.details.techWeight !== undefined && (
            <>
              <span className="text-text-muted">Weight</span>
              <span className="text-text-primary">
                T:{entry.details.techWeight.toFixed(2)} F:{(entry.details.fundWeight ?? 0).toFixed(2)}
              </span>
            </>
          )}
        </div>
      )}
    </article>
  )
}
