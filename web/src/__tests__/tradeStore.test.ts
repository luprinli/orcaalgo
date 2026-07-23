import { describe, it, expect, beforeEach } from 'vitest'
import { useTradeStore } from '../stores/tradeStore'
import type { TradeSummary } from '../types/api'

function makeTrade(id: string, pnl = 100.0): TradeSummary {
  return {
    id,
    symbol: 'USDEUR',
    side: 'BUY',
    quantity: 1000,
    entry_price: 1.0500,
    exit_price: 1.0510,
    pnl,
    pnl_pct: 0.1,
    entry_time: '2024-01-01T00:00:00Z',
    exit_time: '2024-01-02T00:00:00Z',
    hold_duration: 24,
    mae: -0.5,
    mfe: 0.2,
    strategy_id: 'grid_trading',
    exit_reason: 'take_profit',
    commission: 1.0,
  }
}

describe('tradeStore', () => {
  beforeEach(() => {
    useTradeStore.getState().clearTrades()
  })

  it('starts with empty trades', () => {
    expect(useTradeStore.getState().trades).toHaveLength(0)
  })

  it('setTrades replaces the trade list', () => {
    const trades = [makeTrade('a'), makeTrade('b')]
    useTradeStore.getState().setTrades(trades)
    expect(useTradeStore.getState().trades).toHaveLength(2)
  })

  it('addTrade prepends a new trade', () => {
    useTradeStore.getState().addTrade(makeTrade('first'))
    useTradeStore.getState().addTrade(makeTrade('second'))
    expect(useTradeStore.getState().trades).toHaveLength(2)
    expect(useTradeStore.getState().trades[0].id).toBe('first')
  })

  it('addTrade caps at 200 entries', () => {
    for (let i = 0; i < 300; i++) {
      useTradeStore.getState().addTrade(makeTrade(`trade-${i}`))
    }
    expect(useTradeStore.getState().trades).toHaveLength(200)
    // First 100 should be trimmed, so trades[0] should be trade-100
    expect(useTradeStore.getState().trades[0].id).toBe('trade-100')
  })

  it('clearTrades empties the list', () => {
    useTradeStore.getState().addTrade(makeTrade('test'))
    useTradeStore.getState().clearTrades()
    expect(useTradeStore.getState().trades).toHaveLength(0)
  })
})
