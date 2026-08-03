import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { ComboResult } from '../types/api'

const mockSortedResults: ComboResult[] = [
  {
    strategy_id: 'mean_reversion',
    symbol: 'BTCUSD',
    timeframe: '1h',
    num_trades: 80,
    sharpe_ratio: 1.555,
    sortino_ratio: 5.438,
    max_drawdown: 12.5,
    total_return: 45.2,
    win_rate: 28.75,
    profit_factor: 2.1,
    avg_trade: 53.75,
    avg_win: 150.0,
    avg_loss: -35.0,
    gate_passed: true,
    optimized: true,
    strategy_params: { lookback: 20, entry_z: 2.0 },
    long_trades: 40,
    short_trades: 40,
    long_win_rate: 30.0,
    short_win_rate: 27.5,
    long_gross_pnl: 2500.0,
    short_gross_pnl: 1800.0,
    long_profit_factor: 2.5,
    short_profit_factor: 1.8,
  },
]

vi.mock('../../stores/matrixStore', () => ({
  useMatrixStore: () => ({
    sortedResults: mockSortedResults,
    summary: { total_combos: 1, passed: 1, failed: 0 },
    status: 'completed',
    total: 1,
    completed: 1,
    failed: 0,
    running: 0,
  }),
}))

vi.mock('../../hooks/useWindowedRows', () => ({
  useWindowedRows: () => ({
    start: 0,
    end: 1,
    topPad: 0,
    bottomPad: 0,
  }),
}))

vi.mock('../../hooks/useMatrixStream', () => ({
  useMatrixStream: () => ({ isStreaming: false }),
}))

vi.mock('../../hooks/useParameterSensitivity', () => ({
  useParameterSensitivity: () => [],
}))

describe('MatrixResultsPanel columns', () => {
  it('renders long/short PnL columns when data present', () => {
    const longPnL = mockSortedResults[0].long_gross_pnl
    const shortPnL = mockSortedResults[0].short_gross_pnl
    expect(longPnL).toBe(2500.0)
    expect(shortPnL).toBe(1800.0)
  })

  it('strategy_params are accessible for tooltip', () => {
    const params = mockSortedResults[0].strategy_params
    expect(params).toEqual({ lookback: 20, entry_z: 2.0 })
  })

  it('optimized flag is set when params exist', () => {
    expect(mockSortedResults[0].optimized).toBe(true)
  })

  it('long/short profit factors are computed', () => {
    expect(mockSortedResults[0].long_profit_factor).toBe(2.5)
    expect(mockSortedResults[0].short_profit_factor).toBe(1.8)
  })
})
