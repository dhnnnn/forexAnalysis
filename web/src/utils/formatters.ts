// ── Price & Pips ──────────────────────────────────────────────────────────
export function formatPrice(price: number, decimals = 5): string {
  return price.toFixed(decimals)
}

export function formatPips(pips: number): string {
  const sign = pips > 0 ? '+' : ''
  return `${sign}${pips.toFixed(1)} pips`
}

export function formatPercent(value: number, decimals = 1): string {
  return `${(value * 100).toFixed(decimals)}%`
}

export function formatConfidence(value: number): string {
  return `${Math.round(value * 100)}%`
}

// ── Time ─────────────────────────────────────────────────────────────────
export function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString('en-US', { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

export function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}

export function formatDateTime(iso: string): string {
  return `${formatDate(iso)} ${formatTime(iso)}`
}

export function formatTimeAgo(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime()
  const diffSec = Math.floor(diffMs / 1000)
  if (diffSec < 60) return `${diffSec}s ago`
  const diffMin = Math.floor(diffSec / 60)
  if (diffMin < 60) return `${diffMin}m ago`
  const diffHr = Math.floor(diffMin / 60)
  return `${diffHr}h ago`
}

export function formatTTL(expiresAt: string): string {
  const remaining = new Date(expiresAt).getTime() - Date.now()
  if (remaining <= 0) return 'expired'
  const hours = Math.floor(remaining / 3_600_000)
  const minutes = Math.floor((remaining % 3_600_000) / 60_000)
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

export function toUnixTimestamp(iso: string): number {
  return Math.floor(new Date(iso).getTime() / 1000)
}

// ── Number ────────────────────────────────────────────────────────────────
export function formatWeight(w: number): string {
  return w.toFixed(2)
}

export function formatADX(adx: number): string {
  return adx.toFixed(1)
}

// ── Pair ──────────────────────────────────────────────────────────────────
export function formatPairDisplay(pair: string): string {
  return pair.replace('_', '/')
}
