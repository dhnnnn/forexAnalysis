import type { MarketRegime } from '../../types/regime'
import { getRegimeColor, getRegimeIcon } from '../../utils/colors'

interface RegimeBadgeProps {
  regime: MarketRegime | string
  adx?: number
  compact?: boolean
}

export function RegimeBadge({ regime, adx, compact = false }: RegimeBadgeProps) {
  const color = getRegimeColor(regime)
  const icon  = getRegimeIcon(regime)
  const label = regime?.replace('_', ' ') ?? 'UNKNOWN'

  return (
    <span
      className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold"
      style={{
        backgroundColor: `${color}22`,
        color,
        border: `1px solid ${color}44`,
      }}
      title={adx !== undefined ? `ADX: ${adx.toFixed(1)}` : undefined}
    >
      <span>{icon}</span>
      {!compact && <span>{label}</span>}
    </span>
  )
}
