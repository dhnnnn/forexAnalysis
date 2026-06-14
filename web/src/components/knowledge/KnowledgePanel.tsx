import { useQuery, useSubscription } from '@apollo/client'
import { GET_ACTIVE_RULES, GET_ADAPTIVE_WEIGHTS } from '../../graphql/queries'
import { RULE_CREATED } from '../../graphql/subscriptions'
import { useKnowledgeStore, DEFAULT_WEIGHTS } from '../../stores/knowledgeStore'
import { useConnectionStore } from '../../stores/connectionStore'
import { RuleCard } from './RuleCard'
import { WeightsDisplay } from './WeightsDisplay'
import type { KnowledgeRule } from '../../types/knowledge'
import { Brain } from 'lucide-react'

export function KnowledgePanel() {
  const activePair   = useConnectionStore((s) => s.activePair)
  const { activeRules, setActiveRules, addRule, adaptiveWeights, setWeights } = useKnowledgeStore()

  // Load active rules
  const { data: rulesData } = useQuery(GET_ACTIVE_RULES, {
    fetchPolicy: 'cache-and-network',
    pollInterval: 60_000,
    onCompleted: (data) => setActiveRules(data.activeRules ?? []),
  })

  // Load adaptive weights
  const { data: weightsData } = useQuery(GET_ADAPTIVE_WEIGHTS, {
    variables: { pair: activePair },
    fetchPolicy: 'cache-and-network',
    pollInterval: 30_000,
    onCompleted: (data) => {
      if (data.adaptiveWeights) setWeights(activePair, data.adaptiveWeights)
    },
  })

  // Subscribe to new rules
  useSubscription(RULE_CREATED, {
    onData: ({ data }) => {
      const rule = data.data?.ruleCreated as KnowledgeRule | undefined
      if (rule) addRule(rule)
    },
  })

  const weights = adaptiveWeights[activePair] ?? DEFAULT_WEIGHTS
  const rules   = activeRules

  return (
    <div className="flex flex-col border-t border-border-subtle">
      {/* Header */}
      <div className="flex items-center gap-2 px-4 py-2.5 border-b border-border-subtle bg-bg-secondary flex-shrink-0">
        <Brain size={13} className="text-[#56d364]" />
        <span className="text-xs font-semibold text-text-primary">Knowledge</span>
        {rules.length > 0 && (
          <span className="ml-auto bg-[#56d36422] text-[#56d364] border border-[#56d36444] text-[10px] font-mono px-1.5 py-0.5 rounded-full">
            {rules.length} active
          </span>
        )}
      </div>

      {/* Content */}
      <div className="px-3 py-3 space-y-3 overflow-y-auto" style={{ maxHeight: 280 }}>
        {/* Rules */}
        {rules.length > 0 ? (
          <div className="space-y-2">
            {rules.slice(0, 3).map((rule) => (
              <RuleCard key={rule.id} rule={rule} />
            ))}
          </div>
        ) : (
          <p className="text-[11px] text-text-muted text-center py-2">
            No active rules
          </p>
        )}

        {/* Weights */}
        <div className="border-t border-border-subtle pt-3">
          <p className="text-[10px] text-text-muted font-medium mb-2 uppercase tracking-wider">
            Adaptive Weights
          </p>
          <WeightsDisplay weights={weights} />
        </div>
      </div>
    </div>
  )
}
