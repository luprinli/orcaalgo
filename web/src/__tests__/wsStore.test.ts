import { describe, it, expect } from 'vitest'
import { useWSStore } from '../stores/wsStore'

describe('wsStore', () => {
  it('starts disconnected with null data', () => {
    const state = useWSStore.getState()
    expect(state.connected).toBe(false)
    expect(state.risk).toBeNull()
    expect(state.performance).toBeNull()
  })

  it('setConnected updates connection status', () => {
    useWSStore.getState().setConnected(true)
    expect(useWSStore.getState().connected).toBe(true)
  })

  it('setRisk stores risk data', () => {
    const riskData = { halted: false, equity: 100000, daily_pnl_pct: 0.5, regime: 1, reason: '', connections: 0, confidence: 0.8, vix: 15, sentiment: 0.2, daily_loss_used: 10, drawdown_used: 5, daily_limit_pct: 5, max_dd_pct: 10, consistency_multiplier: 1, tick_count: 100, balance: 100000 }
    useWSStore.getState().setRisk(riskData)
    expect(useWSStore.getState().risk?.equity).toBe(100000)
    expect(useWSStore.getState().risk?.halted).toBe(false)
  })

  it('setPerformance stores performance data', () => {
    const perfData = { timestamp: '2024-01-01T00:00:00Z', equity: 100000, balance: 100000, daily_pnl: 500, daily_pnl_pct: 0.5, drawdown_pct: 1, max_drawdown_pct: 5, sharpe: 1.5, sortino: 2.0, win_rate: 0.6, profit_factor: 1.8, num_trades: 50 }
    useWSStore.getState().setPerformance(perfData)
    expect(useWSStore.getState().performance?.sharpe).toBe(1.5)
  })
})
