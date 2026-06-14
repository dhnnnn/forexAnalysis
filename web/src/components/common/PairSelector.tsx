import { useConnectionStore } from '../../stores/connectionStore'
import { formatPairDisplay } from '../../utils/formatters'
import { ChevronDown } from 'lucide-react'

interface PairSelectorProps {
  pairs: string[]
}

export function PairSelector({ pairs }: PairSelectorProps) {
  const activePair   = useConnectionStore((s) => s.activePair)
  const setActivePair = useConnectionStore((s) => s.setActivePair)

  return (
    <div className="flex items-center gap-1">
      {pairs.map((pair) => (
        <button
          key={pair}
          id={`pair-btn-${pair}`}
          onClick={() => setActivePair(pair)}
          className={`px-3 py-1.5 rounded text-xs font-semibold transition-all duration-150 ${
            activePair === pair
              ? 'bg-[#1c2128] text-[#58a6ff] border border-[#58a6ff44]'
              : 'text-text-secondary hover:text-text-primary hover:bg-bg-tertiary border border-transparent'
          }`}
        >
          {formatPairDisplay(pair)}
        </button>
      ))}
    </div>
  )
}

interface TimeframeSelectorProps {
  timeframes: string[]
}

export function TimeframeSelector({ timeframes }: TimeframeSelectorProps) {
  const timeframe    = useConnectionStore((s) => s.timeframe)
  const setTimeframe = useConnectionStore((s) => s.setTimeframe)

  return (
    <div className="flex items-center gap-0.5 bg-bg-tertiary rounded-lg p-0.5">
      {timeframes.map((tf) => (
        <button
          key={tf}
          id={`tf-btn-${tf}`}
          onClick={() => setTimeframe(tf)}
          className={`px-2.5 py-1 rounded text-xs font-mono font-medium transition-all duration-150 ${
            timeframe === tf
              ? 'bg-bg-elevated text-text-primary shadow-sm'
              : 'text-text-secondary hover:text-text-primary'
          }`}
        >
          {tf}
        </button>
      ))}
    </div>
  )
}
