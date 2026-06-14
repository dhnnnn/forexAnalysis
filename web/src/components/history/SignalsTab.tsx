import { useQuery, useSubscription } from '@apollo/client'
import { useState } from 'react'
import { GET_SIGNALS } from '../../graphql/queries'
import { SIGNAL_GENERATED } from '../../graphql/subscriptions'
import { useHistoryStore } from '../../stores/historyStore'
import { useConnectionStore } from '../../stores/connectionStore'
import { StatusBadge } from './shared/StatusBadge'
import { SignalBadge } from '../agents/SignalBadge'
import { formatTime, formatPrice, formatConfidence, formatPips } from '../../utils/formatters'
import { RegimeBadge } from '../common/RegimeBadge'
import type { SignalEntry } from '../../types/history'

export function SignalsTab() {
  const activePair = useConnectionStore((s) => s.activePair)
  const { signals, addSignal, setSignals } = useHistoryStore()
  const [expandedId, setExpandedId] = useState<number | null>(null)

  useQuery(GET_SIGNALS, {
    variables: { pair: activePair, limit: 50 },
    fetchPolicy: 'cache-and-network',
    onCompleted: (data) => setSignals(data.signals ?? []),
  })

  useSubscription(SIGNAL_GENERATED, {
    variables: { pair: activePair },
    onData: ({ data }) => {
      const sig = data.data?.signalGenerated as SignalEntry | undefined
      if (sig) addSignal(sig)
    },
  })

  const pairSignals = signals.filter((s) => s.pair === activePair)

  return (
    <div className="overflow-auto h-full">
      <table className="data-table">
        <thead className="sticky top-0 bg-bg-secondary z-10">
          <tr>
            <th>#</th>
            <th>Time</th>
            <th>Pair</th>
            <th>Signal</th>
            <th>Conf</th>
            <th>Regime</th>
            <th>Entry</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {pairSignals.length === 0 && (
            <tr>
              <td colSpan={8} className="text-center text-text-muted py-6">
                No signals yet
              </td>
            </tr>
          )}
          {pairSignals.map((sig, i) => (
            <>
              <tr
                key={sig.id}
                onClick={() => setExpandedId(expandedId === sig.id ? null : sig.id)}
                className={`cursor-pointer transition-colors ${
                  sig.evalStatus === 'CORRECT'   ? 'bg-[#2ea04308]' :
                  sig.evalStatus === 'INCORRECT' ? 'bg-[#f8514908]' : ''
                }`}
              >
                <td className="text-text-muted">{i + 1}</td>
                <td>{formatTime(sig.timestamp)}</td>
                <td className="font-semibold text-text-primary">{sig.pair.replace('_', '/')}</td>
                <td><SignalBadge signal={sig.signal} size="sm" /></td>
                <td className="font-bold">{formatConfidence(sig.confidence)}</td>
                <td><RegimeBadge regime={sig.regime} compact /></td>
                <td>{sig.entry > 0 ? formatPrice(sig.entry) : '—'}</td>
                <td><StatusBadge status={sig.evalStatus} /></td>
              </tr>
              {expandedId === sig.id && (
                <tr key={`${sig.id}-detail`} className="bg-bg-tertiary">
                  <td colSpan={8} className="px-4 py-2 text-[11px] font-mono">
                    <div className="grid grid-cols-3 gap-x-6 gap-y-1 text-text-secondary">
                      <div>SL: <span className="text-sell-red">{formatPrice(sig.stopLoss)}</span></div>
                      <div>TP: <span className="text-buy-green">{formatPrice(sig.takeProfit)}</span></div>
                      <div>Lot: <span className="text-text-primary">{sig.lotSize}</span></div>
                      <div className="col-span-3 mt-1 text-text-muted">Tech: {sig.techSignal} ({formatConfidence(sig.techConf)}) — {sig.techReason}</div>
                      <div className="col-span-3 text-text-muted">Fund: {sig.fundSentiment} ({formatConfidence(sig.fundConf)}) — {sig.fundReason}</div>
                      {sig.pipsMove !== null && sig.pipsMove !== undefined && (
                        <div className="col-span-3 mt-1">
                          Result: <span style={{ color: sig.pipsMove > 0 ? '#2ea043' : '#f85149' }}>
                            {formatPips(sig.pipsMove)}
                          </span>
                        </div>
                      )}
                    </div>
                  </td>
                </tr>
              )}
            </>
          ))}
        </tbody>
      </table>
    </div>
  )
}
