import { describe, it, expect, vi } from 'vitest'
import { useCacheStore } from '../stores/cacheStore'

describe('cacheStore', () => {
  it('starts with null values', () => {
    const state = useCacheStore.getState()
    expect(state.strategies).toBeNull()
    expect(state.symbols).toBeNull()
    expect(state.accounts).toBeNull()
  })

  it('fetchStrategies calls fetcher and caches result', async () => {
    const mock = vi.fn().mockResolvedValue([{ id: 's1', name: 'Test' }])
    const store = useCacheStore.getState()
    const result = await store.fetchStrategies(mock)
    expect(mock).toHaveBeenCalledOnce()
    expect(result).toEqual([{ id: 's1', name: 'Test' }])
    expect(useCacheStore.getState().strategies).toEqual([{ id: 's1', name: 'Test' }])
  })

  it('fetchStrategies returns cached on second call (stale-while-revalidate)', async () => {
    useCacheStore.setState({ strategies: [{ id: 'cached', name: 'Cached' }] })
    const mock = vi.fn().mockResolvedValue([])
    const store = useCacheStore.getState()
    const result = await store.fetchStrategies(mock)
    expect(result).toEqual([{ id: 'cached', name: 'Cached' }])
  })

  it('fetchSymbols caches and returns', async () => {
    const mock = vi.fn().mockResolvedValue([{ id: 'sym1', ticker: 'SPY' }])
    const store = useCacheStore.getState()
    const result = await store.fetchSymbols(mock)
    expect(result).toHaveLength(1)
  })

  it('fetchAccounts caches and returns', async () => {
    const mock = vi.fn().mockResolvedValue([{ id: 'acc1', name: 'Main' }])
    const store = useCacheStore.getState()
    const result = await store.fetchAccounts(mock)
    expect(result).toHaveLength(1)
  })

  it('invalidate clears specific key', () => {
    useCacheStore.setState({ strategies: [{ id: 's1' }], symbols: [{ id: 'sym1' }] })
    useCacheStore.getState().invalidate('strategies')
    expect(useCacheStore.getState().strategies).toBeNull()
    expect(useCacheStore.getState().symbols).not.toBeNull()
  })
})
