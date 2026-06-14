import type { Signal } from '../../types/agent'
import { getSignalColors } from '../../utils/colors'

interface SignalBadgeProps {
  signal: Signal | string
  size?: 'sm' | 'md'
}

export function SignalBadge({ signal, size = 'md' }: SignalBadgeProps) {
  const colors = getSignalColors(signal)
  const cls = size === 'sm'
    ? 'text-[10px] px-1.5 py-0.5'
    : 'text-xs px-2 py-0.5'

  return (
    <span
      className={`inline-flex items-center rounded font-bold font-mono tracking-wider ${cls}`}
      style={{
        backgroundColor: colors.bg,
        color: colors.text,
        border: `1px solid ${colors.border}44`,
      }}
    >
      {signal === 'BUY'  && <span className="mr-1">🟢</span>}
      {signal === 'SELL' && <span className="mr-1">🔴</span>}
      {signal === 'HOLD' && <span className="mr-1">🟡</span>}
      {signal}
    </span>
  )
}
