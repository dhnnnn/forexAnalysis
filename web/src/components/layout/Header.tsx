import { Settings, TrendingUp } from 'lucide-react'
import { ConnectionStatus } from '../common/ConnectionStatus'
import { PairSelector, TimeframeSelector } from '../common/PairSelector'
import { useQuery } from '@apollo/client'
import { GET_PAIRS } from '../../graphql/queries'
import { AVAILABLE_PAIRS, TIMEFRAMES } from '../../utils/constants'
import { formatPairDisplay } from '../../utils/formatters'
import { useConnectionStore } from '../../stores/connectionStore'

export function Header() {
  const { data } = useQuery(GET_PAIRS, {
    fetchPolicy: 'cache-and-network',
    pollInterval: 30_000,
  })
  const pairs = (data?.pairs as string[]) ?? AVAILABLE_PAIRS
  const activePair = useConnectionStore((s) => s.activePair)

  return (
    <header className="h-12 flex items-center gap-4 px-4 border-b border-border-subtle bg-bg-secondary flex-shrink-0 z-10">
      {/* Logo */}
      <div className="flex items-center gap-2 mr-2">
        <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-[#58a6ff] to-[#bc8cff] flex items-center justify-center shadow-glow-blue flex-shrink-0">
          <TrendingUp size={14} className="text-white" />
        </div>
        <span className="font-bold text-sm text-gradient hidden sm:block tracking-tight">
          ForexAI
        </span>
      </div>

      {/* Divider */}
      <div className="h-5 w-px bg-border-subtle" />

      {/* Pair selector */}
      <PairSelector pairs={pairs} />

      {/* Divider */}
      <div className="h-5 w-px bg-border-subtle" />

      {/* Timeframe selector */}
      <TimeframeSelector timeframes={['5m', '15m', '1h', '4h']} />

      {/* Spacer */}
      <div className="flex-1" />

      {/* Active pair display */}
      <div className="hidden md:flex items-center gap-1 text-xs text-text-muted">
        <span className="font-semibold text-text-secondary">
          {formatPairDisplay(activePair)}
        </span>
      </div>

      {/* Connection status */}
      <ConnectionStatus />

      {/* Settings */}
      <button
        id="settings-btn"
        className="p-1.5 rounded text-text-muted hover:text-text-primary hover:bg-bg-tertiary transition-all duration-150"
        aria-label="Settings"
      >
        <Settings size={14} />
      </button>
    </header>
  )
}
