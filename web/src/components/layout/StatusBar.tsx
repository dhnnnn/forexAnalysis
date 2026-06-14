import { useConnectionStore } from '../../stores/connectionStore'
import { useRegimeStore } from '../../stores/regimeStore'
import { useKnowledgeStore } from '../../stores/knowledgeStore'
import { RegimeBadge } from '../common/RegimeBadge'
import { formatWeight, formatTime, formatPairDisplay } from '../../utils/formatters'
import { Clock, Activity, BookOpen, Scale } from 'lucide-react'

export function StatusBar() {
  const activePair        = useConnectionStore((s) => s.activePair)
  const pipelineRunning   = useConnectionStore((s) => s.pipelineRunning[activePair])
  const lastMessage       = useConnectionStore((s) => s.lastMessage)
  const currentRegime     = useRegimeStore((s) => s.currentRegime[activePair])
  const activeRules       = useKnowledgeStore((s) => s.activeRules)
  const adaptiveWeights   = useKnowledgeStore((s) => s.adaptiveWeights[activePair])

  const tech = adaptiveWeights?.techWeight ?? 0.5
  const fund = adaptiveWeights?.fundWeight ?? 0.5

  return (
    <div className="h-9 flex items-center gap-4 px-4 border-t border-border-subtle bg-bg-secondary flex-shrink-0 text-xs text-text-secondary overflow-x-auto">

      {/* Pair */}
      <span className="font-semibold text-text-primary flex-shrink-0">
        {formatPairDisplay(activePair)}
      </span>

      <div className="h-3.5 w-px bg-border-subtle flex-shrink-0" />

      {/* Regime */}
      <div className="flex items-center gap-1.5 flex-shrink-0">
        <Activity size={11} className="text-text-muted" />
        <span className="text-text-muted">Regime:</span>
        {currentRegime ? (
          <RegimeBadge regime={currentRegime.regime} adx={currentRegime.adx} compact />
        ) : (
          <span className="text-text-muted">—</span>
        )}
        {currentRegime && (
          <span className="font-mono text-text-muted">
            ADX:{currentRegime.adx.toFixed(1)}
          </span>
        )}
      </div>

      <div className="h-3.5 w-px bg-border-subtle flex-shrink-0" />

      {/* Weights */}
      <div className="flex items-center gap-1.5 flex-shrink-0">
        <Scale size={11} className="text-text-muted" />
        <span className="text-text-muted">Tech:</span>
        <span className="font-mono text-[#58a6ff]">{formatWeight(tech)}</span>
        <span className="text-text-muted ml-1">Fund:</span>
        <span className="font-mono text-[#d2a8ff]">{formatWeight(fund)}</span>
      </div>

      <div className="h-3.5 w-px bg-border-subtle flex-shrink-0" />

      {/* Active Rules */}
      <div className="flex items-center gap-1.5 flex-shrink-0">
        <BookOpen size={11} className="text-text-muted" />
        <span className="text-text-muted">Rules:</span>
        <span className="font-mono text-[#56d364]">{activeRules.length} active</span>
      </div>

      <div className="h-3.5 w-px bg-border-subtle flex-shrink-0" />

      {/* Pipeline status */}
      {pipelineRunning && (
        <div className="flex items-center gap-1 text-[#ffa657] flex-shrink-0">
          <span className="animate-pulse">⟳</span>
          <span>Pipeline running…</span>
        </div>
      )}

      {/* Last evaluation */}
      {lastMessage && (
        <div className="flex items-center gap-1.5 flex-shrink-0 ml-auto">
          <Clock size={11} className="text-text-muted" />
          <span className="text-text-muted">Last: {formatTime(lastMessage)}</span>
        </div>
      )}
    </div>
  )
}
