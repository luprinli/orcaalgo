import { create } from 'zustand'
import type { IndicatorSpec, IndicatorWithData, IndicatorResult } from '../types/api'

const STORAGE_KEY = 'orca_indicators_preset'

interface PersistedIndicator {
  specId: string
  parameters: Record<string, number | string>
  paneIndex: number
}

function persist(indicators: Record<string, IndicatorWithData>) {
  const data: PersistedIndicator[] = Object.values(indicators)
    .filter(i => !i.loading)
    .map(i => ({
      specId: i.spec.id,
      parameters: { ...i.parameters },
      paneIndex: i.paneIndex,
    }))
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(data)) } catch { /* quota exceeded */ }
}

function loadPersisted(): PersistedIndicator[] | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return null
    return parsed
  } catch { return null }
}

function hydrateIndicators(
  persisted: PersistedIndicator[],
  specs: IndicatorSpec[],
): Record<string, IndicatorWithData> {
  let nextId = 0
  let paneCount = 1
  const indicators: Record<string, IndicatorWithData> = {}

  for (const p of persisted) {
    const spec = specs.find(s => s.id === p.specId)
    if (!spec) continue
    const id = `ind_${++nextId}_${Date.now()}`
    const paneIndex = spec.overlay ? 0 : p.paneIndex > 0 ? paneCount++ : 0
    indicators[id] = {
      _id: id,
      spec,
      result: null,
      parameters: { ...p.parameters },
      paneIndex: spec.overlay ? 0 : paneIndex,
      dataVersion: 0,
      loading: false,
      error: null,
    }
  }

  return indicators
}

interface IndicatorState {
  indicators: Record<string, IndicatorWithData>
  paneCount: number

  addIndicator: (spec: IndicatorSpec, params: Record<string, number | string>) => string
  setResult: (id: string, result: IndicatorResult) => void
  setLoading: (id: string, loading: boolean) => void
  setError: (id: string, error: string | null) => void
  updateParameters: (id: string, params: Record<string, number | string>) => void
  removeIndicator: (id: string) => void
  clearAll: () => void
  getById: (id: string) => IndicatorWithData | undefined
  all: () => IndicatorWithData[]
}

let nextId = 0

export const useIndicatorStore = create<IndicatorState>((set, get) => ({
  indicators: {},
  paneCount: 1,

  addIndicator: (spec, params) => {
    const id = `ind_${++nextId}_${Date.now()}`
    const paneIndex = spec.overlay ? 0 : get().paneCount

    const indicator: IndicatorWithData = {
      _id: id,
      spec,
      result: null,
      parameters: { ...params },
      paneIndex,
      dataVersion: 0,
      loading: true,
      error: null,
    }

    set({
      indicators: { ...get().indicators, [id]: indicator },
      paneCount: spec.overlay ? get().paneCount : get().paneCount + 1,
    })

    return id
  },

  setResult: (id, result) => {
    const current = get().indicators[id]
    if (!current) return

    const next = {
      ...get().indicators,
      [id]: { ...current, result, loading: false, dataVersion: current.dataVersion + 1, error: null },
    }

    set({ indicators: next })
    persist(next)
  },

  setLoading: (id, loading) => {
    const current = get().indicators[id]
    if (!current) return

    set({
      indicators: { ...get().indicators, [id]: { ...current, loading } },
    })
  },

  setError: (id, error) => {
    const current = get().indicators[id]
    if (!current) return

    const next = {
      ...get().indicators,
      [id]: { ...current, error, loading: false },
    }

    set({ indicators: next })
    persist(next)
  },

  updateParameters: (id, params) => {
    const current = get().indicators[id]
    if (!current) return

    set({
      indicators: {
        ...get().indicators,
        [id]: { ...current, parameters: { ...current.parameters, ...params }, loading: true },
      },
    })
  },

  removeIndicator: (id) => {
    const indicator = get().indicators[id]
    if (!indicator) return

    const next = { ...get().indicators }
    delete next[id]

    if (!indicator.spec.overlay && indicator.paneIndex > 0) {
      for (const key of Object.keys(next)) {
        if (next[key].paneIndex > indicator.paneIndex) {
          next[key] = { ...next[key], paneIndex: next[key].paneIndex - 1 }
        }
      }
      set({ indicators: next, paneCount: Math.max(1, get().paneCount - 1) })
    } else {
      set({ indicators: next })
    }

    persist(next)
  },

  clearAll: () => {
    set({ indicators: {}, paneCount: 1 })
    try { localStorage.removeItem(STORAGE_KEY) } catch { /* ignore */ }
  },

  getById: (id) => get().indicators[id],

  all: () => Object.values(get().indicators),
}))

export function restoreIndicators(specs: IndicatorSpec[]): boolean {
  const persisted = loadPersisted()
  if (!persisted || persisted.length === 0) return false

  const indicators = hydrateIndicators(persisted, specs)
  if (Object.keys(indicators).length === 0) return false

  const paneCount = Math.max(1, ...Object.values(indicators).map(i => i.paneIndex)) + 1

  useIndicatorStore.setState({ indicators, paneCount })
  nextId = Object.keys(indicators).length
  return true
}
