import { useCallback } from 'react'
import { useIndicatorStore } from '../stores/indicatorStore'
import { indicators } from '../api/client'
import type { Candle } from '../types/api'

export function useIndicatorCompute() {
  const store = useIndicatorStore()

  const compute = useCallback(async (id: string, candles: Candle[]) => {
    const indicator = store.getById(id)
    if (!indicator) return

    store.setLoading(id, true)

    try {
      const res = await indicators.compute(indicator.spec.id, {
        parameters: indicator.parameters,
        candles,
      })

      store.setResult(id, res.indicator)
    } catch (err) {
      store.setError(id, err instanceof Error ? err.message : 'Computation failed')
    }
  }, [store])

  return { compute }
}
