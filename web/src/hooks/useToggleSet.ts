import { useState, useCallback } from 'react'

export function useToggleSet(initial: string[] = []) {
  const [set, setSet] = useState<Set<string>>(() => new Set(initial))
  const toggle = useCallback((key: string) => {
    setSet(prev => { const n = new Set(prev); if (n.has(key)) n.delete(key); else n.add(key); return n })
  }, [])
  const selectAll = useCallback((keys: string[]) => {
    setSet(prev => new Set([...prev, ...keys]))
  }, [])
  const deselectAll = useCallback(() => {
    setSet(new Set())
  }, [])
  const replace = useCallback((keys: string[]) => {
    setSet(new Set(keys))
  }, [])
  return { set, toggle, selectAll, deselectAll, replace }
}
