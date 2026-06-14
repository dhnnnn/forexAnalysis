import { useEffect, useRef } from 'react'

export function useAutoScroll(deps: unknown[]) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    // Only auto-scroll if user is near the bottom (within 80px)
    const isNearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
    if (isNearBottom) {
      el.scrollTop = el.scrollHeight
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  return ref
}
