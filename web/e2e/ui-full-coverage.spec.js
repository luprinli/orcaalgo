// @ts-check
import { test, expect } from '@playwright/test';

const BASE = 'http://127.0.0.1:8080';

function headers() {
  return { 'Content-Type': 'application/json' }
}

test.describe('Auth Flow', () => {
  let authToken = '';

  test.beforeAll(async ({ request }) => {
    try {
      const r = await request.post(`${BASE}/api/v1/auth/register`, {
        data: { username: 'e2e_tester', password: 'Test123!', email: 'e2e@orca.test', roles: ['trader'] },
      });
      const d = await r.json();
      if (d.access_token) authToken = d.access_token;
    } catch {}
    if (!authToken) {
      try {
        const r = await request.post(`${BASE}/api/v1/auth/login`, {
          data: { username: 'e2e_tester', password: 'Test123!' },
        });
        const d = await r.json();
        if (d.access_token) authToken = d.access_token;
      } catch {}
    }
  });

  test('Login with valid credentials', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/auth/login`, {
      data: { username: 'e2e_tester', password: 'Test123!' },
    });
    expect(r.status()).toBeLessThan(500);
    const d = await r.json();
    expect(d.access_token || d.token).toBeTruthy();
  });

  test('Login with invalid credentials returns error', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/auth/login`, {
      data: { username: 'nonexistent', password: 'wrong' },
    });
    expect(r.status()).toBeGreaterThanOrEqual(400);
  });
});

test.describe('Dashboard & Health', () => {
  test('Backtest health endpoint', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/backtests/health`);
    expect(r.status()).toBe(200);
    const d = await r.json();
    expect(d.overall).toBe('ok');
    expect(d.checks.length).toBe(3);
  });

  test('Risk status endpoint', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/risk/status`);
    expect(r.status()).toBeLessThan(500);
  });

  test('Admin health endpoint', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/admin/health`);
    expect(r.status()).toBeLessThan(500);
  });
});

test.describe('Backtest Page', () => {
  test('Submit backtest with 3 strategies', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/backtests`, {
      data: {
        strategy_ids: ['ma_crossover', 'rsi2_reversion', 'donchian_breakout'],
        symbols: ['SPX500'],
        start_date: '2024-01-01', end_date: '2024-06-30',
        capital: 100000,
      },
      timeout: 60000,
    });
    expect(r.status()).toBeLessThan(500);
    const d = await r.json();
    expect(d.status).toBe('completed');
    expect(typeof d.sharpe_ratio).toBe('number');
    expect(typeof d.num_trades).toBe('number');
  });

  test('Submit matrix backtest with all 11 strategies', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/backtests`, {
      data: {
        strategy_ids: ['trend_following','opening_range_breakout','grid_trading','session_scalp','mean_reversion','intraday_mr','ma_crossover','rsi2_reversion','donchian_breakout','keltner_macd','ichimoku_cloud'],
        symbols: ['SPX500'],
        start_date: '2024-01-01', end_date: '2024-03-31',
        capital: 100000, mode: 'matrix',
      },
      timeout: 120000,
    });
    expect(r.status()).toBeLessThan(500);
  });

  test('Backtest metrics endpoint', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/backtests/bt-test/metrics`);
    expect(r.status()).toBeLessThan(500);
  });
});

test.describe('Backtest Detail & Promote Wizard', () => {
  let backtestId = '';

  test.beforeAll(async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/backtests`, {
      data: {
        strategy_ids: ['ma_crossover'],
        symbols: ['SPX500'],
        start_date: '2024-01-01', end_date: '2024-01-31',
        capital: 100000,
      },
    });
    const d = await r.json();
    backtestId = d.id || '';
  });

  test('Fetch backtest metrics', async ({ request }) => {
    if (!backtestId) { test.skip(true, 'no backtest ID'); return; }
    const r = await request.get(`${BASE}/api/v1/backtests/${backtestId}/metrics`);
    expect(r.status()).toBeLessThan(500);
  });

  test('Fetch backtest equity', async ({ request }) => {
    if (!backtestId) { test.skip(true, 'no backtest ID'); return; }
    const r = await request.get(`${BASE}/api/v1/backtests/${backtestId}/equity?resolution=1d`);
    expect(r.status()).toBeLessThan(500);
  });

  test('Live comparison endpoint', async ({ request }) => {
    if (!backtestId) { test.skip(true, 'no backtest ID'); return; }
    const r = await request.get(`${BASE}/api/v1/backtests/${backtestId}/live-comparison`);
    expect(r.status()).toBeLessThan(500);
  });

  test('Backtest trades endpoint', async ({ request }) => {
    if (!backtestId) { test.skip(true, 'no backtest ID'); return; }
    const r = await request.get(`${BASE}/api/v1/backtests/${backtestId}/trades`);
    expect(r.status()).toBeLessThan(500);
  });
});

test.describe('Strategies Page', () => {
  test('List strategies', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/strategies`);
    expect(r.status()).toBeLessThan(500);
    const d = await r.json();
    const items = d.strategies || d || [];
    expect(Array.isArray(items)).toBe(true);
    expect(items.length).toBeGreaterThan(0);
  });

  test('Validate strategy', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/strategies/validate`, {
      data: { name: 'test_strategy', type: 'trend', parameters: {} },
    });
    expect(r.status()).toBeLessThan(500);
  });
});

test.describe('Execution Page', () => {
  test('Place market order', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/orders`, {
      data: { symbol: 'SPX500', side: 'BUY', type: 'MARKET', quantity: 10, timeInForce: 'DAY' },
    });
    expect(r.status()).toBeLessThan(500);
    const d = await r.json();
    expect(d.status).toBeTruthy();
  });

  test('Place limit order with camelCase fields', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/orders`, {
      data: { symbol: 'SPX500', side: 'SELL', type: 'LIMIT', quantity: 5, limitPrice: 8500, timeInForce: 'DAY' },
    });
    expect(r.status()).toBeLessThan(500);
  });

  test('List positions', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/positions`);
    expect(r.status()).toBeLessThan(500);
  });

  test('List orders', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/orders`);
    expect(r.status()).toBeLessThan(500);
    const d = await r.json();
    expect(d.orders).toBeDefined();
  });
});

test.describe('Edge Cases & Error Handling', () => {
  test('Empty strategy list in backtest', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/backtests`, {
      data: {
        strategy_ids: [],
        symbols: ['SPX500'],
        start_date: '2024-01-01', end_date: '2024-01-31',
        capital: 100000,
      },
    });
    expect(r.status()).toBeLessThan(500);
  });

  test('Invalid date range (end before start)', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/backtests`, {
      data: {
        strategy_ids: ['ma_crossover'],
        symbols: ['SPX500'],
        start_date: '2024-06-30', end_date: '2024-01-01',
        capital: 100000,
      },
    });
    expect(r.status()).toBeLessThan(500);
  });

  test('Order with zero quantity', async ({ request }) => {
    const r = await request.post(`${BASE}/api/v1/orders`, {
      data: { symbol: 'SPX500', side: 'BUY', type: 'MARKET', quantity: 0, timeInForce: 'DAY' },
    });
    expect(r.status()).toBeLessThan(500);
  });

  test('404 on unknown route', async ({ request }) => {
    const r = await request.get(`${BASE}/api/v1/nonexistent`);
    expect(r.status()).toBe(404);
  });
});
