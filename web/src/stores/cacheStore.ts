import { create } from 'zustand'

interface CacheStore {
  strategies: any[] | null
  symbols: any[] | null
  accounts: any[] | null
  fetchStrategies: (fetcher: () => Promise<any[]>) => Promise<any[]>
  fetchSymbols: (fetcher: () => Promise<any[]>) => Promise<any[]>
  fetchAccounts: (fetcher: () => Promise<any[]>) => Promise<any[]>
  invalidate: (key: 'strategies' | 'symbols' | 'accounts') => void
}

export const useCacheStore = create<CacheStore>((set, get) => ({
  strategies: null,
  symbols: null,
  accounts: null,

  fetchStrategies: async (fetcher) => {
    const cached = get().strategies
    if (cached) {
      fetcher().then(s => set({ strategies: s })).catch(() => {})
      return cached
    }
    const s = await fetcher()
    set({ strategies: s })
    return s
  },

  fetchSymbols: async (fetcher) => {
    const cached = get().symbols
    if (cached) {
      fetcher().then(s => set({ symbols: s })).catch(() => {})
      return cached
    }
    const s = await fetcher()
    set({ symbols: s })
    return s
  },

  fetchAccounts: async (fetcher) => {
    const cached = get().accounts
    if (cached) {
      fetcher().then(a => set({ accounts: a })).catch(() => {})
      return cached
    }
    const a = await fetcher()
    set({ accounts: a })
    return a
  },

  invalidate: (key) => set({ [key]: null }),
}))
