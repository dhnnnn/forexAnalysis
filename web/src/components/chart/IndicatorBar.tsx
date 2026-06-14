import type { CandleData } from '../../types/candle'
import { useSubscription } from '@apollo/client'
import { SIGNAL_GENERATED } from '../../graphql/subscriptions'
import { useConnectionStore } from '../../stores/connectionStore'
import type { SignalEntry } from '../../types/history'

interface IndicatorBarProps {
  pair: string
  lastCandle?: CandleData
}

// Simple indicator display — in production these come from the backend
export function IndicatorBar({ pair, lastCandle }: IndicatorBarProps) {
  const activePair = useConnectionStore((s) => s.activePair)

  // Track last signal for weight info
  const lastSignalRef = { current: null as SignalEntry | null }
  useSubscription(SIGNAL_GENERATED, {
    variables: { pair: activePair },
    onData: ({ data }) => {
      lastSignalRef.current = data.data?.signalGenerated ?? null
    },
  })

  if (!lastCandle) {
    return (
      <div className="h-8 bg-bg-secondary border-t border-border-subtle flex items-center px-4 gap-4 text-xs text-text-muted font-mono flex-shrink-0">
        <span className="animate-pulse">Loading indicators…</span>
      </div>
    )
  }

  const spread = lastCandle.spread ?? 0

  return (
    <div className="h-9 bg-bg-secondary border-t border-border-subtle flex items-center px-4 gap-5 text-[11px] text-text-muted font-mono flex-shrink-0 overflow-x-auto">
      <IndicatorItem
        label="SPREAD"
        value={(spread * 10000).toFixed(1)}
        unit="pips"
      />
      <div className="h-3 w-px bg-border-subtle" />
      <IndicatorItem
        label="CLOSE"
        value={lastCandle.close.toFixed(5)}
      />
      <div className="h-3 w-px bg-border-subtle" />
      <IndicatorItem
        label="VOL"
        value={lastCandle.volume.toFixed(0)}
      />
      <div className="h-3 w-px bg-border-subtle" />
      <div className="text-text-muted text-[10px]">
        O:<span className="text-text-primary ml-1">{lastCandle.open.toFixed(5)}</span>
        <span className="mx-1.5">H:<span className="text-buy-green ml-1">{lastCandle.high.toFixed(5)}</span></span>
        L:<span className="text-sell-red ml-1">{lastCandle.low.toFixed(5)}</span>
      </div>
    </div>
  )
}

function IndicatorItem({ label, value, unit, colorClass }: {
  label: string
  value: string
  unit?: string
  colorClass?: string
}) {
  return (
    <div className="flex items-center gap-1">
      <span className="text-text-muted">{label}:</span>
      <span className={`font-semibold ${colorClass ?? 'text-text-primary'}`}>{value}</span>
      {unit && <span className="text-text-muted">{unit}</span>}
    </div>
  )
}
