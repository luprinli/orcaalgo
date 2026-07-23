import { useMemo } from 'react'
import type { MatrixResultsResponse } from '../types/api'

export interface SensitivityEntry {
  key: string
  bestSharpe: number
  bestParams: string
}

export function useParameterSensitivity(matrixResult: MatrixResultsResponse | null) {
  return useMemo(() => {
    if (!matrixResult?.results?.length) return { entries: [], colorFor: () => '', minS: 0, maxS: 0 }

    const paramsMap = new Map<string, Map<string, { sharpe: number; count: number }>>()
    for (const r of matrixResult.results) {
      const key = `${r.strategy_id}|${r.symbol}|${r.timeframe}`
      if (!paramsMap.has(key)) paramsMap.set(key, new Map())
      const pk = r.optimized && r.best_params ? JSON.stringify(r.best_params) : 'default'
      const m = paramsMap.get(key)!
      const prev = m.get(pk)
      m.set(pk, { sharpe: (prev?.sharpe ?? 0) + r.sharpe_ratio, count: (prev?.count ?? 0) + 1 })
    }

    const entries: SensitivityEntry[] = Array.from(paramsMap.entries())
      .map(([key, pMap]) => {
        const best = Array.from(pMap.entries()).sort((a, b) => (b[1].sharpe / b[1].count) - (a[1].sharpe / a[1].count))[0]
        return { key, bestSharpe: best[1].sharpe / best[1].count, bestParams: best[0] }
      })
      .sort((a, b) => b.bestSharpe - a.bestSharpe)
      .slice(0, 12)

    if (entries.length <= 1) return { entries: [], colorFor: () => '', minS: 0, maxS: 0 }

    const allSharpes = entries
      .map(e => {
        const pMap = paramsMap.get(e.key)
        if (!pMap) return []
        return Array.from(pMap.values()).map(v => v.sharpe / v.count)
      })
      .flat()
      .filter(s => isFinite(s))

    if (allSharpes.length === 0) return { entries: [], colorFor: () => '', minS: 0, maxS: 0 }

    const minS = Math.min(...allSharpes)
    const maxS = Math.max(...allSharpes)
    const range = maxS - minS || 1

    const colorFor = (s: number) => {
      const t = Math.max(0, Math.min(1, (s - minS) / range))
      const r = Math.round(239 * (1 - t) + 63 * t)
      const g = Math.round(83 * (1 - t) + 185 * t)
      const b = Math.round(80 * (1 - t) + 80 * t)
      return `rgb(${r},${g},${b})`
    }

    return { entries, colorFor, minS, maxS }
  }, [matrixResult])
}
