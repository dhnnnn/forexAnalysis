import { useCallback, useRef, useEffect } from 'react'

interface UseResizableOptions {
  initialHeight: number
  minHeight: number
  maxHeight: number
  onHeightChange: (height: number) => void
}

export function useResizable({ initialHeight, minHeight, maxHeight, onHeightChange }: UseResizableOptions) {
  const startY       = useRef(0)
  const startHeight  = useRef(initialHeight)
  const isDragging   = useRef(false)

  const onMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault()
      isDragging.current  = true
      startY.current      = e.clientY
      startHeight.current = initialHeight
      document.body.style.cursor = 'ns-resize'
      document.body.style.userSelect = 'none'
    },
    [initialHeight]
  )

  useEffect(() => {
    const onMouseMove = (e: MouseEvent) => {
      if (!isDragging.current) return
      const delta  = startY.current - e.clientY
      const newH   = Math.min(maxHeight, Math.max(minHeight, startHeight.current + delta))
      onHeightChange(newH)
    }
    const onMouseUp = () => {
      if (!isDragging.current) return
      isDragging.current = false
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    document.addEventListener('mousemove', onMouseMove)
    document.addEventListener('mouseup', onMouseUp)
    return () => {
      document.removeEventListener('mousemove', onMouseMove)
      document.removeEventListener('mouseup', onMouseUp)
    }
  }, [minHeight, maxHeight, onHeightChange])

  return { onMouseDown }
}
