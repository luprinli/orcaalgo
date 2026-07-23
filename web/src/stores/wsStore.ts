import { create } from 'zustand'
import type { WSRiskData, WSPerformanceData, WSOrderData } from '../types/ws'
import type { Position } from '../types/api'

interface WSState {
  connected: boolean
  risk: WSRiskData | null
  performance: WSPerformanceData | null
  positions: Position[] | null
  orders: WSOrderData | null
  openPositionCount: number
  activeBacktestRuns: number
  setConnected: (connected: boolean) => void
  setRisk: (data: WSRiskData) => void
  setPerformance: (data: WSPerformanceData) => void
  setPositions: (data: Position[] | null) => void
  setOrders: (data: WSOrderData | null) => void
  setOpenPositionCount: (n: number) => void
  setActiveBacktestRuns: (n: number) => void
}

export const useWSStore = create<WSState>((set) => ({
  connected: false,
  risk: null,
  performance: null,
  positions: null,
  orders: null,
  openPositionCount: 0,
  activeBacktestRuns: 0,
  setConnected: (connected) => set({ connected }),
  setRisk: (risk) => set({ risk }),
  setPerformance: (performance) => set({ performance }),
  setPositions: (positions) => set({ positions }),
  setOrders: (orders) => set({ orders }),
  setOpenPositionCount: (n) => set({ openPositionCount: n }),
  setActiveBacktestRuns: (n) => set({ activeBacktestRuns: n }),
}))
