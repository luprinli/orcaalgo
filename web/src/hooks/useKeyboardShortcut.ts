import { useEffect } from 'react'

export function useKeyboardShortcut(key: string, handler: () => void, options?: { metaKey?: boolean; ctrlKey?: boolean; enabled?: boolean }) {
  useEffect(() => {
    if (options?.enabled === false) return

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === key || e.code === key) {
        const metaRequired = options?.metaKey ?? false
        const ctrlRequired = options?.ctrlKey ?? false
        if (metaRequired && !e.metaKey) return
        if (ctrlRequired && !e.ctrlKey) return
        if (metaRequired || ctrlRequired) {
          e.preventDefault()
        }
        handler()
      }
    }

    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [key, handler, options?.metaKey, options?.ctrlKey, options?.enabled])
}
