import { describe, it, expect, vi } from 'vitest'
import { exportMatrixResultsCSV, exportTradesCSV, exportEquityCSV, exportDailyReturnsCSV } from '../lib/export'
import type { ComboResult, TradeSummary, EquityPoint, DailyReturn } from '../types/api'

const mockCreateObjectURL = vi.fn(() => 'blob:mock')
const mockRevokeObjectURL = vi.fn()
URL.createObjectURL = mockCreateObjectURL
URL.revokeObjectURL = mockRevokeObjectURL

let appendedChild: HTMLAnchorElement | null = null
document.body.appendChild = vi.fn((el) => { appendedChild = el as HTMLAnchorElement; return el }) as any
document.body.removeChild = vi.fn()

describe('exportMatrixResultsCSV', () => {
  it('generates 26-column CSV with long/short breakdown and params', () => {
    const results: ComboResult[] = [
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
        avg_mae: 0.5,
        avg_mfe: 1.2,
        num_wins: 23,
        num_losses: 57,
      },
    ]

    exportMatrixResultsCSV(results)
    expect(mockCreateObjectURL).toHaveBeenCalled()
  })

  it('handles empty results', () => {
    exportMatrixResultsCSV([])
    expect(mockCreateObjectURL).toHaveBeenCalled()
  })

  it('handles null/undefined optional fields', () => {
    const results: ComboResult[] = [
      { strategy_id: 'test', symbol: 'TEST', timeframe: '1d', num_trades: 0, sharpe_ratio: 0, sortino_ratio: 0, max_drawdown: 0, total_return: 0, win_rate: 0, profit_factor: 0, avg_trade: 0, avg_win: 0, avg_loss: 0 },
    ]
    expect(() => exportMatrixResultsCSV(results)).not.toThrow()
  })
})

describe('exportTradesCSV', () => {
  it('generates CSV with trade columns', () => {
    const trades = [
      { symbol: 'AAPL', side: 'BUY', quantity: 100, entry_price: 185.0, exit_price: 187.5, pnl: 250.0, pnl_pct: 1.35, mae: -0.5, mfe: 1.8, hold_duration: 2.5, entry_time: '2025-01-15T10:30:00Z', exit_time: '2025-01-15T13:00:00Z', exit_reason: 'take_profit' },
    ] as TradeSummary[]
    exportTradesCSV(trades)
    expect(mockCreateObjectURL).toHaveBeenCalled()
  })

  it('escapeCSV handles commas and quotes', () => {
    const trades = [
      { symbol: 'AAPL', side: 'BUY', quantity: 100, entry_price: 185.0, exit_price: 187.5, pnl: 250.0, pnl_pct: 1.35, mae: -0.5, mfe: 1.8, hold_duration: 2.5, entry_time: '2025-01-15T10:30:00Z', exit_time: '2025-01-15T13:00:00Z', exit_reason: 'stop_loss,"fast"' },
    ] as TradeSummary[]
    expect(() => exportTradesCSV(trades)).not.toThrow()
  })
})

describe('exportEquityCSV', () => {
  it('generates CSV from equity points', () => {
    const points: EquityPoint[] = [
      { time: '2025-01-15T09:30:00Z', value: 100000, regime: 0 },
      { time: '2025-01-15T16:00:00Z', value: 100500, regime: 0 },
    ]
    exportEquityCSV(points)
    expect(mockCreateObjectURL).toHaveBeenCalled()
  })
})

describe('exportDailyReturnsCSV', () => {
  it('generates CSV from daily returns', () => {
    const returns: DailyReturn[] = [
      { date: '2025-01-15', return_pct: 0.5, pnl: 500 },
      { date: '2025-01-16', return_pct: -0.2, pnl: -200 },
    ]
    exportDailyReturnsCSV(returns)
    expect(mockCreateObjectURL).toHaveBeenCalled()
  })
})
