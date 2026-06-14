import { useQuery, useSubscription } from '@apollo/client'
import { GET_REGIME_CHANGES, GET_CURRENT_REGIME } from '../../graphql/queries'
import { REGIME_CHANGED } from '../../graphql/subscriptions'
import { useHistoryStore } from '../../stores/historyStore'
import { useConnectionStore } from '../../stores/connectionStore'
import { useRegimeStore } from '../../stores/regimeStore'
import { RegimeBadge } from '../common/RegimeBadge'
import { formatTime } from '../../utils/formatters'
import type { RegimeChange, RegimeContext } from '../../types/regime'
import { getRegimeColor } from '../../utils/colors'

export function RegimeTab() {
  const activePair = useConnectionStore((s) => s.activePair)
  const { regimeChanges, setRegimeChanges } = useHistoryStore()
  const { currentRegime, setRegime } = useRegimeStore()

  useQuery(GET_REGIME_CHANGES, {
    variables: { pair: activePair, limit: 20 },
    fetchPolicy: 'cache-and-network',
    onCompleted: (d) => setRegimeChanges(d.regimeChanges ?? []),
  })

  useQuery(GET_CURRENT_REGIME, {
    variables: { pair: activePair },
    fetchPolicy: 'cache-and-network',
    pollInterval: 30_000,
    onCompleted: (d) => {
      if (d.currentRegime) setRegime(d.currentRegime)
    },
  })

  useSubscription(REGIME_CHANGED, {
    variables: { pair: activePair },
    onData: ({ data }) => {
      const regime = data.data?.regimeChanged as RegimeContext | undefined
      if (regime) setRegime(regime)
    },
  })

  const current = currentRegime[activePair]
  const changes = regimeChanges.filter((c) => c.pair === activePair)

  return (
    <div className="overflow-auto h-full p-3 space-y-3">
      {/* Current regime card */}
      {current && (
        <div className="p-3 rounded-lg border border-border-subtle bg-bg-elevated">
          <p className="text-[10px] text-text-muted uppercase tracking-wider mb-2 font-medium">Current</p>
          <div className="flex items-center gap-3">
            <RegimeBadge regime={current.regime} adx={current.adx} />
            <div className="grid grid-cols-2 gap-x-4 gap-y-0.5 text-[11px] font-mono ml-2">
              <span className="text-text-muted">ADX</span>
              <span className="text-text-primary">{current.adx.toFixed(1)}</span>
              <span className="text-text-muted">ATR</span>
              <span className="text-text-primary">{current.atr.toFixed(5)}</span>
              <span className="text-text-muted">Vol</span>
              <span className="text-text-primary">{(current.volatility * 100).toFixed(2)}%</span>
              <span className="text-text-muted">Trend</span>
              <span className="text-text-primary">{current.trendStrength.toFixed(2)}</span>
            </div>
          </div>
        </div>
      )}

      {/* Regime change log */}
      <div>
        <p className="text-[10px] text-text-muted uppercase tracking-wider mb-2 font-medium">
          Change Log
        </p>
        {changes.length === 0 && (
          <p className="text-text-muted text-xs text-center py-3">No regime changes recorded</p>
        )}
        <table className="data-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>From</th>
              <th>→ To</th>
              <th>ADX</th>
              <th>Vol</th>
            </tr>
          </thead>
          <tbody>
            {changes.map((c, i) => (
              <tr key={i}>
                <td className="text-text-muted">{formatTime(c.changedAt)}</td>
                <td>
                  <span style={{ color: getRegimeColor(c.fromRegime) }}>
                    {c.fromRegime}
                  </span>
                </td>
                <td>
                  <span style={{ color: getRegimeColor(c.toRegime) }}>
                    {c.toRegime}
                  </span>
                </td>
                <td>{c.adx.toFixed(1)}</td>
                <td>{(c.volatility * 100).toFixed(2)}%</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
