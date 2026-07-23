import { useEffect } from 'react'

export function useGlobalShortcut(key: string, handler: () => void, enabled = true) {
  useEffect(() => {
    if (!enabled) return
    const cb = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === key) {
        e.preventDefault()
        handler()
      }
    }
    window.addEventListener('keydown', cb)
    return () => window.removeEventListener('keydown', cb)
  }, [key, handler, enabled])
}
