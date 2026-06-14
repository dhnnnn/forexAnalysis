import type { KnowledgeRule } from '../../types/knowledge'
import { formatTTL, formatTime } from '../../utils/formatters'
import { getAgentColor, getRegimeColor } from '../../utils/colors'
import { Clock, TrendingDown } from 'lucide-react'

interface RuleCardProps {
  rule: KnowledgeRule
}

export function RuleCard({ rule }: RuleCardProps) {
  const isExpired = rule.status === 'expired'
  const agentColor = getAgentColor(rule.targetAgent)
  const regimeColor = getRegimeColor(rule.regime)
  const ttl = formatTTL(rule.expiresAt)

  const deltaSign = rule.weightDelta > 0 ? '+' : ''

  return (
    <div
      className={`p-3 rounded-lg border transition-all duration-150 ${
        isExpired
          ? 'border-border-subtle bg-bg-primary opacity-60'
          : 'border-[#30363d] bg-bg-elevated hover:border-[#484f58]'
      }`}
    >
      {/* Header */}
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-1.5">
          <span className={`w-1.5 h-1.5 rounded-full ${isExpired ? 'bg-text-muted' : 'bg-buy-green'}`} />
          <span className="text-[10px] font-mono text-text-muted">
            #{rule.id.slice(0, 8)}
          </span>
        </div>
        {!isExpired && (
          <div className="flex items-center gap-1 text-[10px] text-text-muted">
            <Clock size={9} />
            <span className="font-mono">{ttl}</span>
          </div>
        )}
        {isExpired && (
          <span className="text-[10px] text-text-muted font-mono">{formatTime(rule.expiresAt)}</span>
        )}
      </div>

      {/* Action */}
      <div className="flex items-center gap-2 mb-1.5">
        <span className="text-xs font-semibold" style={{ color: agentColor }}>
          {rule.targetAgent}
        </span>
        <span className="text-xs font-mono font-bold" style={{
          color: rule.weightDelta < 0 ? '#f85149' : '#2ea043'
        }}>
          {deltaSign}{rule.weightDelta.toFixed(2)}
        </span>
        <span
          className="text-[10px] px-1.5 py-0.5 rounded-full font-semibold"
          style={{ backgroundColor: `${regimeColor}22`, color: regimeColor }}
        >
          {rule.regime}
        </span>
      </div>

      {/* Reasoning */}
      <p className="text-[11px] text-text-secondary line-clamp-2 mb-2 leading-relaxed">
        {rule.reasoning}
      </p>

      {/* Footer stats */}
      <div className="flex items-center gap-3 text-[10px] font-mono text-text-muted">
        <span>Conf: <span className="text-text-secondary">{Math.round(rule.confidence * 100)}%</span></span>
        <span>Applied: <span className="text-[#56d364]">{rule.applyCount}x</span></span>
        {rule.impactAccuracyDelta !== undefined && rule.impactAccuracyDelta !== null && (
          <span>Impact: <span style={{ color: rule.impactAccuracyDelta > 0 ? '#2ea043' : '#f85149' }}>
            {rule.impactAccuracyDelta > 0 ? '+' : ''}{(rule.impactAccuracyDelta * 100).toFixed(1)}%
          </span></span>
        )}
      </div>
    </div>
  )
}
