import { useRef, useCallback, useEffect } from 'react'

export function useChartUpdate() {
  const rafRef = useRef<number | null>(null)
  const queueRef = useRef<Array<() => void>>([])

  const flush = useCallback(() => {
    const updates = queueRef.current
    queueRef.current = []
    for (const update of updates) {
      update()
    }
    rafRef.current = null
  }, [])

  const enqueue = useCallback((update: () => void) => {
    queueRef.current.push(update)
    if (rafRef.current === null) {
      rafRef.current = requestAnimationFrame(flush)
    }
  }, [flush])

  useEffect(() => {
    return () => {
      if (rafRef.current !== null) {
        cancelAnimationFrame(rafRef.current)
      }
    }
  }, [])

  return { enqueue }
}
