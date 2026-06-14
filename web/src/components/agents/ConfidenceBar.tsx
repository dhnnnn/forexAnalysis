import { getSignalColors } from '../../utils/colors'
import { formatConfidence } from '../../utils/formatters'

interface ConfidenceBarProps {
  value: number  // 0-1
  signal?: string
  showLabel?: boolean
}

export function ConfidenceBar({ value, signal = 'HOLD', showLabel = true }: ConfidenceBarProps) {
  const colors = getSignalColors(signal)
  const pct = Math.round(value * 100)

  return (
    <div className="flex items-center gap-2">
      <div className="conf-bar-track flex-1" style={{ minWidth: 60 }}>
        <div
          className="conf-bar-fill"
          style={{
            width: `${pct}%`,
            background: colors.text,
            opacity: 0.8,
          }}
        />
      </div>
      {showLabel && (
        <span className="font-mono text-xs font-bold" style={{ color: colors.text, minWidth: 32 }}>
          {formatConfidence(value)}
        </span>
      )}
    </div>
  )
}
