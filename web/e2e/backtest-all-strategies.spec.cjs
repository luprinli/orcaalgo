// @ts-check
const { test, expect } = require('@playwright/test');

const STRATEGIES = [
  'trend_following', 'opening_range_breakout', 'grid_trading',
  'session_scalp', 'mean_reversion', 'intraday_mr',
  'ma_crossover', 'rsi2_reversion', 'donchian_breakout',
  'keltner_macd', 'ichimoku_cloud',
]

test.describe('Backtest E2E — API Validation', () => {
  test('Backtest health check', async ({ request }) => {
    const r = await request.get('http://localhost:8080/api/v1/backtests/health')
    expect(r.status()).toBe(200)
    const data = await r.json()
    expect(data.overall).toBe('ok')
    expect(data.checks[2].component).toBe('engine')
    expect(data.checks[1].candle_count).toBeGreaterThan(0)
  })

  test('Submit single-strategy backtest', async ({ request }) => {
    const r = await request.post('http://localhost:8080/api/v1/backtests', {
      data: { strategy_ids: ['ma_crossover'], symbols: ['SPX500'], start_date: '2024-01-01', end_date: '2024-03-31', capital: 100000 }
    })
    expect(r.status()).toBeLessThan(500)
    const d = await r.json()
    expect(d.status).toBe('completed')
    expect(typeof d.sharpe_ratio).toBe('number')
    expect(typeof d.max_drawdown).toBe('number')
    expect(typeof d.win_rate).toBe('number')
    expect(typeof d.num_trades).toBe('number')
  })

  test('List strategies via API', async ({ request }) => {
    const r = await request.get('http://localhost:8080/api/v1/strategies')
    expect(r.status()).toBeLessThan(500)
    const d = await r.json()
    expect(Array.isArray(d.strategies) || Array.isArray(d)).toBe(true)
  })

  test('Place order via API', async ({ request }) => {
    const r = await request.post('http://localhost:8080/api/v1/orders', {
      data: { symbol: 'SPX500', side: 'BUY', type: 'MARKET', quantity: 10, timeInForce: 'DAY' }
    })
    expect(r.status()).toBeLessThan(500)
  })

  test('Live comparison endpoint exists', async ({ request }) => {
    const r = await request.get('http://localhost:8080/api/v1/backtests/bt-test/live-comparison')
    expect(r.status()).toBeLessThan(500)
  })
})

test.describe('Matrix Backtest', () => {
  test('Run 11-strategy matrix', async ({ request }) => {
    const r = await request.post('http://localhost:8080/api/v1/backtests', {
      data: { strategy_ids: STRATEGIES, symbols: ['SPX500'], start_date: '2024-01-01', end_date: '2024-03-31', capital: 100000, mode: 'matrix' }
    })
    expect(r.status()).toBeLessThan(500)
    const d = await r.json()
    expect(d.status || d.id).toBeTruthy()
  })
})
