import type { AdaptiveWeights } from '../../types/regime'

interface WeightsDisplayProps {
  weights: AdaptiveWeights
}

export function WeightsDisplay({ weights }: WeightsDisplayProps) {
  const techPct = Math.round(weights.techWeight * 100)
  const fundPct = Math.round(weights.fundWeight * 100)

  return (
    <div className="space-y-2">
      <WeightRow label="Technical" value={weights.techWeight} color="#58a6ff" base={0.60} />
      <WeightRow label="Fundamental" value={weights.fundWeight} color="#d2a8ff" base={0.40} />
      {weights.rulesApplied > 0 && (
        <p className="text-[10px] text-text-muted font-mono">
          {weights.rulesApplied} rule{weights.rulesApplied > 1 ? 's' : ''} applied
        </p>
      )}
    </div>
  )
}

function WeightRow({ label, value, color, base }: {
  label: string
  value: number
  color: string
  base: number
}) {
  const pct  = Math.round(value * 100)
  const diff = value - base
  const diffSign = diff > 0 ? '+' : ''

  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-[11px]">
        <span className="font-medium" style={{ color }}>{label}</span>
        <div className="flex items-center gap-1.5 font-mono">
          <span className="text-text-primary font-bold">{value.toFixed(2)}</span>
          {diff !== 0 && (
            <span style={{ color: diff > 0 ? '#2ea043' : '#f85149', fontSize: 10 }}>
              ({diffSign}{diff.toFixed(2)})
            </span>
          )}
        </div>
      </div>
      <div className="conf-bar-track">
        <div
          className="conf-bar-fill"
          style={{ width: `${pct}%`, background: color }}
        />
      </div>
    </div>
  )
}
