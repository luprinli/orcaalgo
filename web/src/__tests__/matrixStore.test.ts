import { describe, it, expect, beforeEach } from 'vitest'
import { useMatrixStore } from '../stores/matrixStore'
import type { MatrixResultsResponse, ComboResult } from '../types/api'

function makeResult(overrides: Partial<ComboResult> = {}): ComboResult {
  return {
    batch_id: 'run-1',
    symbol: 'SPX500',
    strategy_id: 'ma_crossover',
    timeframe: '1d',
    sharpe_ratio: 1.5,
    sortino_ratio: 2.0,
    max_drawdown: -15,
    total_return: 25,
    win_rate: 55,
    profit_factor: 1.8,
    avg_trade: 0,
    avg_win: 0,
    avg_loss: 0,
    num_trades: 42,
    gate_passed: true,
    ...overrides,
  }
}

function makeResponse(results: ComboResult[], summaryOverrides: Partial<MatrixResultsResponse['summary']> = {}): MatrixResultsResponse {
  return {
    summary: {
      total_combos: 10,
      passed: 3,
      failed: 1,
      total_trades: 84,
      best_sharpe: 2.1,
      best_strategy: 'ichimoku_cloud',
      best_symbol: 'NAS100',
      status: 'running',
      completed: 3,
      running: 2,
      percent: 30,
      throughput_per_min: 5.5,
      eta_seconds: 76.4,
      seq: 3,
      ...summaryOverrides,
    },
    results,
    batch_id: 'batch-1',
    status: 'running',
    seq: 3,
  }
}

describe('matrixStore', () => {
  beforeEach(() => {
    const { getState } = useMatrixStore
    getState().reset()
  })

  it('begins a new batch', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-abc', 20)

    const s = getState()
    expect(s.batchId).toBe('batch-abc')
    expect(s.status).toBe('running')
    expect(s.telemetry.total).toBe(20)
    expect(s.seq).toBe(0)
    expect(s.order).toEqual([])
  })

  it('applyDelta upserts results by combo key', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-1', 5)

    const r1 = makeResult({ strategy_id: 'intraday_mr', symbol: 'EURUSD', timeframe: '1h', sharpe_ratio: 1.2 })
    const resp = makeResponse([r1])
    getState().applyDelta(resp)

    const s = getState()
    expect(Object.keys(s.byKey)).toHaveLength(1)
    expect(s.order).toHaveLength(1)
    expect(s.telemetry.completed).toBe(3)
  })

  it('applyDelta appends new results, does not duplicate existing', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-2', 5)

    const r1 = makeResult({ strategy_id: 'a', symbol: 'X', timeframe: '1d' })
    const r2 = makeResult({ strategy_id: 'b', symbol: 'Y', timeframe: '1h' })

    getState().applyDelta(makeResponse([r1]))
    expect(Object.keys(getState().byKey)).toHaveLength(1)

    getState().applyDelta(makeResponse([r2]))
    expect(Object.keys(getState().byKey)).toHaveLength(2)
    expect(getState().order).toHaveLength(2)
  })

  it('applyDelta updates existing result with same combo key', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-3', 5)

    const r1 = makeResult({ strategy_id: 'grid_trading', symbol: 'BTCUSD', timeframe: '5m', sharpe_ratio: 0.5 })
    getState().applyDelta(makeResponse([r1]))

    const r2 = makeResult({ strategy_id: 'grid_trading', symbol: 'BTCUSD', timeframe: '5m', sharpe_ratio: 1.8 })
    getState().applyDelta(makeResponse([r2]))

    const key = 'grid_trading|BTCUSD|5m'
    expect(getState().byKey[key].sharpe_ratio).toBe(1.8)
    expect(getState().order).toHaveLength(1)
  })

  it('applyDelta handles empty results gracefully', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-4', 5)

    getState().applyDelta(makeResponse([]))
    expect(getState().order).toEqual([])
    expect(getState().status).toBe('running')
  })

  it('applyDelta updates telemetry from summary', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-5', 10)

    const r1 = makeResult({ sharpe_ratio: 0.8 })
    getState().applyDelta(makeResponse([r1], {
      completed: 4,
      percent: 40,
      throughput_per_min: 12,
      best_sharpe: 2.5,
      best_strategy: 'trend_following',
      best_symbol: 'US30',
    }))

    const t = getState().telemetry
    expect(t.completed).toBe(4)
    expect(t.percent).toBe(40)
    expect(t.throughputPerMin).toBe(12)
    expect(t.bestSharpe).toBe(2.5)
    expect(t.bestStrategy).toBe('trend_following')
    expect(t.bestSymbol).toBe('US30')
  })

  it('applyDelta sets completed status in telemetry from order length', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-6', 5)

    const results = [
      makeResult({ strategy_id: 'a', symbol: 'A', timeframe: '1d' }),
      makeResult({ strategy_id: 'b', symbol: 'B', timeframe: '1d' }),
    ]
    getState().applyDelta(makeResponse(results, { completed: undefined as unknown as number }))

    expect(getState().telemetry.completed).toBe(2)
  })

  it('results() returns in insertion order', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-7', 3)

    const r1 = makeResult({ strategy_id: 'a', symbol: 'X', timeframe: '1d' })
    const r2 = makeResult({ strategy_id: 'b', symbol: 'Y', timeframe: '1h' })

    getState().applyDelta(makeResponse([r1]))
    getState().applyDelta(makeResponse([r2]))

    const results = getState().results()
    expect(results).toHaveLength(2)
    expect(results[0].strategy_id).toBe('a')
    expect(results[1].strategy_id).toBe('b')
  })

  it('setStatus updates status', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-8', 5)
    getState().setStatus('completed')
    expect(getState().status).toBe('completed')
  })

  it('reset clears all state', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-9', 10)
    getState().applyDelta(makeResponse([makeResult()]))
    getState().reset()

    expect(getState().batchId).toBeNull()
    expect(getState().status).toBe('idle')
    expect(getState().order).toEqual([])
    expect(Object.keys(getState().byKey)).toHaveLength(0)
    expect(getState().telemetry.total).toBe(0)
  })

  it('applyDelta infers completed status correctly', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-10', 2)

    getState().applyDelta(makeResponse([], { status: 'completed', completed: 2, percent: 100 }))
    expect(getState().status).toBe('completed')
  })

  it('applyDelta returns cancelled status when cancelled flag set', () => {
    const { getState } = useMatrixStore
    getState().begin('batch-11', 2)

    const resp = makeResponse([], { cancelled: true } as unknown as MatrixResultsResponse['summary'])
    getState().applyDelta(resp)
    expect(getState().status).toBe('cancelled')
  })
})
