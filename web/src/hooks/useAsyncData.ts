import { useState, useEffect, useCallback, useRef } from 'react'

interface AsyncState<T> {
  data: T | null
  loading: boolean
  error: string | null
}

export function useAsyncData<T>(
  fetcher: () => Promise<T>,
  deps: unknown[],
): AsyncState<T> & { refetch: () => void; setData: (data: T | null) => void } {
  const [state, setState] = useState<AsyncState<T>>({ data: null, loading: true, error: null })
  const mountedRef = useRef(true)
  const epochRef = useRef(0)

  const fetch = useCallback(async () => {
    const epoch = ++epochRef.current
    setState(prev => ({ ...prev, loading: true, error: null }))
    try {
      const data = await fetcher()
      if (mountedRef.current && epoch === epochRef.current) setState({ data, loading: false, error: null })
    } catch (err) {
      if (mountedRef.current && epoch === epochRef.current) setState({ data: null, loading: false, error: err instanceof Error ? err.message : 'Unknown error' })
    }
  }, deps) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    mountedRef.current = true
    fetch()
    return () => { mountedRef.current = false }
  }, [fetch])

  const setData = useCallback((data: T | null) => {
    setState(prev => ({ ...prev, data }))
  }, [])

  return { ...state, refetch: fetch, setData }
}
