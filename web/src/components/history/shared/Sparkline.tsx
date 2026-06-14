interface SparklineProps {
  data: boolean[]  // true=win, false=loss
  width?: number
  height?: number
}

export function Sparkline({ data, width = 120, height = 24 }: SparklineProps) {
  if (data.length === 0) return <span className="text-text-muted text-xs">—</span>

  const barW = Math.max(2, Math.floor(width / data.length) - 1)
  const total = data.length

  return (
    <svg width={width} height={height} className="inline-block" aria-hidden>
      {data.map((win, i) => (
        <rect
          key={i}
          x={i * (barW + 1)}
          y={0}
          width={barW}
          height={height}
          rx={1}
          fill={win ? '#2ea043' : '#f85149'}
          opacity={0.7}
        />
      ))}
    </svg>
  )
}
