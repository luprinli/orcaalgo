import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { useIndicatorStore, restoreIndicators } from '../stores/indicatorStore'
import type { IndicatorSpec, IndicatorResult } from '../types/api'

function makeSpec(name: string, overlay = false): IndicatorSpec {
  return {
    id: name.toLowerCase(),
    name,
    description: `${name} indicator`,
    overlay,
    parameters: [],
    outputs: [],
    warmup: 0,
  }
}

function makeResult(name: string, overlay = false): IndicatorResult {
  return {
    id: name.toLowerCase(),
    name,
    overlay,
    outputs: [],
    data: [],
  }
}

function params(period = 14): Record<string, number> {
  return { period }
}

describe('indicatorStore', () => {
  beforeEach(() => {
    useIndicatorStore.getState().clearAll()
  })

  it('starts empty', () => {
    const state = useIndicatorStore.getState()
    expect(state.all()).toHaveLength(0)
    expect(state.paneCount).toBe(1)
  })

  it('addIndicator creates a new indicator and returns its id', () => {
    const spec = makeSpec('RSI')
    const id = useIndicatorStore.getState().addIndicator(spec, params())
    expect(id).toMatch(/^ind_\d+_\d+$/)
    expect(useIndicatorStore.getState().all()).toHaveLength(1)

    const ind = useIndicatorStore.getState().getById(id)
    expect(ind).toBeDefined()
    expect(ind!.spec.name).toBe('RSI')
    expect(ind!.loading).toBe(true)
  })

  it('setResult updates the indicator with data', () => {
    const spec = makeSpec('SMA')
    const id = useIndicatorStore.getState().addIndicator(spec, params())

    useIndicatorStore.getState().setResult(id, makeResult('SMA'))

    const ind = useIndicatorStore.getState().getById(id)
    expect(ind!.result).not.toBeNull()
    expect(ind!.loading).toBe(false)
    expect(ind!.dataVersion).toBe(1)
  })

  it('setError marks indicator as errored', () => {
    const spec = makeSpec('MACD')
    const id = useIndicatorStore.getState().addIndicator(spec, params())
    useIndicatorStore.getState().setError(id, 'fetch failed')

    const ind = useIndicatorStore.getState().getById(id)
    expect(ind!.error).toBe('fetch failed')
    expect(ind!.loading).toBe(false)
  })

  it('removeIndicator deletes and shifts pane indices', () => {
    const s1 = makeSpec('A', false)
    const s2 = makeSpec('B', false)
    const id1 = useIndicatorStore.getState().addIndicator(s1, params())
    useIndicatorStore.getState().addIndicator(s2, params())

    expect(useIndicatorStore.getState().paneCount).toBe(3) // base 1 + 2 non-overlay

    useIndicatorStore.getState().removeIndicator(id1)
    expect(useIndicatorStore.getState().getById(id1)).toBeUndefined()
    expect(useIndicatorStore.getState().paneCount).toBe(2)
  })

  it('overlay indicators do not increase paneCount', () => {
    const overlay = makeSpec('VWAP', true)
    useIndicatorStore.getState().addIndicator(overlay, params())
    expect(useIndicatorStore.getState().paneCount).toBe(1)
  })

  it('updateParameters sets loading and merges params', () => {
    const spec = makeSpec('RSI')
    const id = useIndicatorStore.getState().addIndicator(spec, { period: 7 })

    useIndicatorStore.getState().updateParameters(id, { period: 21 })
    const ind = useIndicatorStore.getState().getById(id)
    expect(ind!.parameters.period).toBe(21)
    expect(ind!.loading).toBe(true)
  })

  it('clearAll resets everything', () => {
    useIndicatorStore.getState().addIndicator(makeSpec('RSI'), params())
    useIndicatorStore.getState().addIndicator(makeSpec('SMA'), params())
    useIndicatorStore.getState().clearAll()
    expect(useIndicatorStore.getState().all()).toHaveLength(0)
    expect(useIndicatorStore.getState().paneCount).toBe(1)
  })

  it('getById returns undefined for unknown id', () => {
    expect(useIndicatorStore.getState().getById('nonexistent')).toBeUndefined()
  })
})

describe('indicatorStore localStorage persistence', () => {
  const STORAGE_KEY = 'orca_indicators_preset'

  beforeEach(() => {
    useIndicatorStore.getState().clearAll()
    const store: Record<string, string> = {}
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => store[key] ?? null),
      setItem: vi.fn((key: string, value: string) => { store[key] = value }),
      removeItem: vi.fn((key: string) => { delete store[key] }),
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('clearAll removes localStorage key', () => {
    const spec = makeSpec('RSI')
    const id = useIndicatorStore.getState().addIndicator(spec, params())
    useIndicatorStore.getState().setResult(id, makeResult('RSI'))

    expect(localStorage.getItem(STORAGE_KEY)).toBeTruthy()

    useIndicatorStore.getState().clearAll()

    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })

  it('setResult persists indicator spec ID and parameters', () => {
    const spec = makeSpec('RSI')
    const id = useIndicatorStore.getState().addIndicator(spec, { period: 14, smoothing: 'ema' })
    useIndicatorStore.getState().setResult(id, makeResult('RSI'))

    const raw = localStorage.getItem(STORAGE_KEY)
    expect(raw).toBeTruthy()
    const parsed = JSON.parse(raw!)
    expect(Array.isArray(parsed)).toBe(true)
    expect(parsed).toHaveLength(1)
    expect(parsed[0].specId).toBe('rsi')
    expect(parsed[0].parameters).toEqual({ period: 14, smoothing: 'ema' })
    expect(parsed[0].paneIndex).toBe(1)
  })

  it('only persists non-loading indicators', () => {
    const spec1 = makeSpec('RSI')
    const spec2 = makeSpec('SMA')
    const id1 = useIndicatorStore.getState().addIndicator(spec1, params(14))
    useIndicatorStore.getState().addIndicator(spec2, params(20))

    useIndicatorStore.getState().setResult(id1, makeResult('RSI'))

    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = JSON.parse(raw!)
    expect(parsed).toHaveLength(1)
    expect(parsed[0].specId).toBe('rsi')
  })

  it('restoreIndicators returns false when localStorage is empty', () => {
    const result = restoreIndicators([makeSpec('RSI')])
    expect(result).toBe(false)
    expect(useIndicatorStore.getState().all()).toHaveLength(0)
  })

  it('restoreIndicators returns false when localStorage has invalid JSON', () => {
    localStorage.setItem(STORAGE_KEY, 'not valid json')

    const result = restoreIndicators([makeSpec('RSI')])
    expect(result).toBe(false)
  })

  it('restoreIndicators restores previously persisted indicators', () => {
    const persisted = [
      { specId: 'rsi', parameters: { period: 14 }, paneIndex: 1 },
      { specId: 'sma', parameters: { period: 50 }, paneIndex: 2 },
    ]
    localStorage.setItem(STORAGE_KEY, JSON.stringify(persisted))

    const specs: IndicatorSpec[] = [makeSpec('RSI'), makeSpec('SMA')]
    const result = restoreIndicators(specs)

    expect(result).toBe(true)
    const all = useIndicatorStore.getState().all()
    expect(all).toHaveLength(2)

    const rsi = all.find(i => i.spec.id === 'rsi')
    expect(rsi).toBeDefined()
    expect(rsi!.parameters).toEqual({ period: 14 })
    expect(rsi!.paneIndex).toBe(1)

    const sma = all.find(i => i.spec.id === 'sma')
    expect(sma).toBeDefined()
    expect(sma!.parameters).toEqual({ period: 50 })
    expect(sma!.paneIndex).toBe(2)
  })

  it('restoreIndicators skips unknown specIds', () => {
    const persisted = [
      { specId: 'rsi', parameters: { period: 14 }, paneIndex: 1 },
      { specId: 'unknown_indicator', parameters: {}, paneIndex: 2 },
    ]
    localStorage.setItem(STORAGE_KEY, JSON.stringify(persisted))

    const specs: IndicatorSpec[] = [makeSpec('RSI')]
    const result = restoreIndicators(specs)

    expect(result).toBe(true)
    const all = useIndicatorStore.getState().all()
    expect(all).toHaveLength(1)
    expect(all[0].spec.id).toBe('rsi')
  })

  it('multiple add/remove sequences only persist active indicators', () => {
    const specRsi = makeSpec('RSI')
    const specSma = makeSpec('SMA')
    const specMacd = makeSpec('MACD')

    const idRsi = useIndicatorStore.getState().addIndicator(specRsi, params(14))
    useIndicatorStore.getState().setResult(idRsi, makeResult('RSI'))

    const idSma = useIndicatorStore.getState().addIndicator(specSma, params(20))
    useIndicatorStore.getState().setResult(idSma, makeResult('SMA'))

    const idMacd = useIndicatorStore.getState().addIndicator(specMacd, params(12))
    useIndicatorStore.getState().setResult(idMacd, makeResult('MACD'))

    // Remove SMA
    useIndicatorStore.getState().removeIndicator(idSma)

    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = JSON.parse(raw!)
    expect(parsed).toHaveLength(2)

    const specIds = parsed.map((p: { specId: string }) => p.specId)
    expect(specIds).toContain('rsi')
    expect(specIds).toContain('macd')
    expect(specIds).not.toContain('sma')
  })

  it('overlay indicators persist with correct paneIndex', () => {
    const overlaySpec = makeSpec('vwap', true)
    const id = useIndicatorStore.getState().addIndicator(overlaySpec, { anchor: 'session' })
    useIndicatorStore.getState().setResult(id, makeResult('VWAP', true))

    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = JSON.parse(raw!)
    expect(parsed[0].paneIndex).toBe(0)
    expect(parsed[0].specId).toBe('vwap')
  })
})
