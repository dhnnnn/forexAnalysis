import { useQuery, useSubscription } from '@apollo/client'
import { useState, useRef } from 'react'
import { GET_LOGS } from '../../graphql/queries'
import { LOG_ADDED } from '../../graphql/subscriptions'
import { useHistoryStore } from '../../stores/historyStore'
import { useAutoScroll } from '../../hooks/useAutoScroll'
import { getLogLevelColor, LOG_LEVEL_COLORS } from '../../utils/colors'
import { formatTime } from '../../utils/formatters'
import type { SystemLog, LogLevel } from '../../types/history'
import { Search } from 'lucide-react'

const LEVELS: LogLevel[] = ['DEBUG', 'INFO', 'WARN', 'ERROR']

export function SystemLogTab() {
  const { logs, addLog, setLogs, logFilter, setLogFilter } = useHistoryStore()
  const [autoScroll, setAutoScroll] = useState(true)
  const scrollRef = useAutoScroll(autoScroll ? [logs.length] : [])

  useQuery(GET_LOGS, {
    variables: { limit: 200 },
    fetchPolicy: 'cache-and-network',
    onCompleted: (d) => setLogs(d.logs ?? []),
  })

  useSubscription(LOG_ADDED, {
    onData: ({ data }) => {
      const log = data.data?.logAdded as SystemLog | undefined
      if (log) addLog(log)
    },
  })

  // Filter
  const filtered = logs.filter((l) => {
    if (logFilter.level && l.level !== logFilter.level) return false
    if (logFilter.search && !l.message.toLowerCase().includes(logFilter.search.toLowerCase())) return false
    return true
  })

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-2 px-3 py-2 border-b border-border-subtle flex-shrink-0">
        {/* Level filters */}
        <div className="flex gap-1">
          <button
            onClick={() => setLogFilter({ level: '' })}
            className={`text-[10px] px-2 py-0.5 rounded font-mono font-medium transition-all ${
              !logFilter.level ? 'bg-bg-elevated text-text-primary' : 'text-text-muted hover:text-text-primary'
            }`}
          >
            ALL
          </button>
          {LEVELS.map((level) => (
            <button
              key={level}
              onClick={() => setLogFilter({ level: logFilter.level === level ? '' : level })}
              className={`text-[10px] px-2 py-0.5 rounded font-mono font-medium transition-all ${
                logFilter.level === level ? 'bg-bg-elevated' : 'text-text-muted hover:text-text-secondary'
              }`}
              style={logFilter.level === level ? { color: LOG_LEVEL_COLORS[level] } : {}}
            >
              {level}
            </button>
          ))}
        </div>

        {/* Search */}
        <div className="flex items-center gap-1.5 ml-2 flex-1 max-w-40">
          <Search size={11} className="text-text-muted flex-shrink-0" />
          <input
            type="text"
            value={logFilter.search}
            onChange={(e) => setLogFilter({ search: e.target.value })}
            placeholder="Search…"
            className="bg-transparent text-[11px] font-mono text-text-primary placeholder-text-muted border-none outline-none w-full"
          />
        </div>

        {/* Auto-scroll toggle */}
        <label className="flex items-center gap-1.5 ml-auto text-[10px] text-text-muted cursor-pointer">
          <input
            type="checkbox"
            checked={autoScroll}
            onChange={(e) => setAutoScroll(e.target.checked)}
            className="accent-[#58a6ff]"
          />
          Auto-scroll
        </label>
      </div>

      {/* Log list */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto font-mono text-[11px] leading-relaxed">
        {filtered.length === 0 && (
          <p className="text-text-muted text-center py-6">No logs</p>
        )}
        {filtered.map((log, i) => (
          <div
            key={i}
            className="flex items-start gap-2 px-3 py-0.5 hover:bg-bg-tertiary transition-colors"
          >
            <span className="text-text-muted flex-shrink-0 text-[10px] mt-0.5">
              {formatTime(log.timestamp)}
            </span>
            <span
              className="font-bold flex-shrink-0 text-[10px] w-10 mt-0.5"
              style={{ color: getLogLevelColor(log.level) }}
            >
              {log.level}
            </span>
            {log.agent && (
              <span className="text-[#58a6ff] flex-shrink-0 mt-0.5 text-[10px]">
                [{log.agent}]
              </span>
            )}
            <span className="text-text-secondary break-all">{log.message}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
