import { useQuery } from '@apollo/client'
import { GET_ACTIVE_RULES, GET_EXPIRED_RULES } from '../../graphql/queries'
import { useHistoryStore } from '../../stores/historyStore'
import { RuleCard } from '../knowledge/RuleCard'
import type { KnowledgeRule } from '../../types/knowledge'

export function RulesTab() {
  const { activeRules, expiredRules, setHistoryActiveRules, setHistoryExpiredRules } = useHistoryStore()

  useQuery(GET_ACTIVE_RULES, {
    fetchPolicy: 'cache-and-network',
    pollInterval: 30_000,
    onCompleted: (d) => setHistoryActiveRules(d.activeRules ?? []),
  })

  useQuery(GET_EXPIRED_RULES, {
    variables: { limit: 10 },
    fetchPolicy: 'cache-and-network',
    pollInterval: 60_000,
    onCompleted: (d) => setHistoryExpiredRules(d.expiredRules ?? []),
  })

  return (
    <div className="overflow-auto h-full p-3 space-y-4">
      {/* Active rules */}
      <div>
        <p className="text-[10px] text-[#56d364] uppercase tracking-wider mb-2 font-medium">
          Active Rules ({activeRules.length})
        </p>
        {activeRules.length === 0 ? (
          <p className="text-text-muted text-xs text-center py-3">No active rules</p>
        ) : (
          <div className="space-y-2">
            {activeRules.map((r) => <RuleCard key={r.id} rule={r} />)}
          </div>
        )}
      </div>

      {/* Expired rules */}
      {expiredRules.length > 0 && (
        <div>
          <p className="text-[10px] text-text-muted uppercase tracking-wider mb-2 font-medium">
            Expired (last 24h)
          </p>
          <div className="space-y-2 opacity-70">
            {expiredRules.map((r) => <RuleCard key={r.id} rule={{ ...r, status: 'expired' }} />)}
          </div>
        </div>
      )}
    </div>
  )
}
