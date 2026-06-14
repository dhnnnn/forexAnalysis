import { useConnectionStore } from '../../stores/connectionStore'

export function ConnectionStatus() {
  const status = useConnectionStore((s) => s.status)
  const lastMessage = useConnectionStore((s) => s.lastMessage)

  const label = {
    connected:    'Connected',
    reconnecting: 'Reconnecting…',
    disconnected: 'Disconnected',
  }[status]

  const dotClass = {
    connected:    'dot-connected',
    reconnecting: 'dot-reconnecting',
    disconnected: 'dot-disconnected',
  }[status]

  return (
    <div
      className="flex items-center gap-1.5 text-xs"
      title={lastMessage ? `Last update: ${lastMessage}` : undefined}
    >
      <span className={dotClass} aria-label={`Status: ${label}`} />
      <span className="text-text-secondary font-medium">{label}</span>
    </div>
  )
}
