import type { EvalStatus } from '../../../types/history'

interface StatusBadgeProps {
  status?: EvalStatus | null
}

export function StatusBadge({ status }: StatusBadgeProps) {
  if (!status) return <span className="text-text-muted">—</span>

  const map = {
    PENDING:   { icon: '⏳', label: 'Pending', color: '#d29922' },
    CORRECT:   { icon: '✅', label: 'Correct',   color: '#2ea043' },
    INCORRECT: { icon: '❌', label: 'Incorrect', color: '#f85149' },
    SKIPPED:   { icon: '—',  label: 'Skip',      color: '#484f58' },
  }

  const { icon, label, color } = map[status] ?? map.PENDING
  return (
    <span className="text-xs" style={{ color }} title={label}>
      {icon} {label}
    </span>
  )
}
