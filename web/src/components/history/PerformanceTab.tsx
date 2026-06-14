import { useQuery } from '@apollo/client'
import { GET_AGENT_SUMMARIES, GET_PERFORMANCE_LOGS } from '../../graphql/queries'
import { useHistoryStore } from '../../stores/historyStore'
import { Sparkline } from './shared/Sparkline'
import { formatPercent, formatTime, formatPips } from '../../utils/formatters'
import { getAgentColor } from '../../utils/colors'
import { TrendingUp, TrendingDown } from 'lucide-react'

export function PerformanceTab() {
  const { agentSummaries, performanceLogs, setAgentSummaries } = useHistoryStore()

  useQuery(GET_AGENT_SUMMARIES, {
    fetchPolicy: 'cache-and-network',
    pollInterval: 30_000,
    onCompleted: (d) => setAgentSummaries(d.agentSummaries ?? []),
  })

  useQuery(GET_PERFORMANCE_LOGS, {
    variables: { limit: 20 },
    fetchPolicy: 'cache-and-network',
  })

  return (
    <div className="overflow-auto h-full p-3 space-y-4">
      {/* Summary cards */}
      <div className="space-y-2">
        {agentSummaries.length === 0 && (
          <p className="text-text-muted text-xs text-center py-4">No performance data yet</p>
        )}
        {agentSummaries.map((agent) => {
          const color = getAgentColor(agent.agentName)
          const improved = agent.accuracy >= agent.accuracyPrev
          return (
            <div key={agent.agentName} className="flex items-center gap-3 px-3 py-2.5 rounded-lg bg-bg-elevated border border-border-subtle">
              <div className="w-2 h-6 rounded-full flex-shrink-0" style={{ background: color }} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 mb-1.5">
                  <span className="text-xs font-semibold" style={{ color }}>{agent.agentName}</span>
                  <span className="text-[10px] text-text-muted font-mono ml-auto">
                    W:{agent.winCount} L:{agent.lossCount}
                    {agent.lossStreak > 1 && <span className="text-sell-red ml-1">streak:{agent.lossStreak}</span>}
                  </span>
                </div>
                {/* Accuracy bar */}
                <div className="conf-bar-track">
                  <div className="conf-bar-fill" style={{ width: formatPercent(agent.accuracy), background: color }} />
                </div>
                <div className="flex items-center justify-between mt-1">
                  <div className="flex items-center gap-1 text-[10px]">
                    {improved
                      ? <TrendingUp size={9} className="text-buy-green" />
                      : <TrendingDown size={9} className="text-sell-red" />
                    }
                    <span className="font-mono font-bold" style={{ color }}>
                      {formatPercent(agent.accuracy)}
                    </span>
                  </div>
                  <Sparkline data={agent.history.slice(-30)} width={80} height={16} />
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {/* Recent evals table */}
      {performanceLogs.length > 0 && (
        <div>
          <p className="text-[10px] text-text-muted uppercase tracking-wider mb-2 font-medium">Recent Evaluations</p>
          <table className="data-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Agent</th>
                <th>Pair</th>
                <th>Result</th>
                <th>Pips</th>
              </tr>
            </thead>
            <tbody>
              {performanceLogs.slice(0, 10).map((log, i) => (
                <tr key={i}>
                  <td className="text-text-muted">{formatTime(log.evalTime)}</td>
                  <td style={{ color: getAgentColor(log.agentName) }}>{log.agentName.replace('Agent','')}</td>
                  <td className="text-text-secondary">{log.pair.replace('_', '/')}</td>
                  <td>{log.correct ? '✅' : '❌'}</td>
                  <td style={{ color: log.pipsMove > 0 ? '#2ea043' : '#f85149' }}>
                    {formatPips(log.pipsMove)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
