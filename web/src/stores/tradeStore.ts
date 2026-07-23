import { create } from 'zustand'
import type { TradeSummary } from '../types/api'

interface TradeState {
  trades: TradeSummary[]
  setTrades: (trades: TradeSummary[]) => void
  addTrade: (trade: TradeSummary) => void
  clearTrades: () => void
}

export const useTradeStore = create<TradeState>((set) => ({
  trades: [],
  setTrades: (trades) => set({ trades }),
  addTrade: (trade) => set((s) => ({
    trades: [...s.trades, trade].slice(-200),
  })),
  clearTrades: () => set({ trades: [] }),
}))
