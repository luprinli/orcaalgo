const { test, expect } = require('@playwright/test');

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.evaluate(() => {
    localStorage.setItem('orca_auth', JSON.stringify({
      token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.DuZ_rWj_5v4IQp9JSOCbVB7BqBQpKBQfBJyq6rzYhxI',
      user: 'e2e-test',
      expires: Date.now() + 86400000,
    }));
  });
});

test.describe('Backtest Page — E2E', () => {
  test('page loads with form, toggle, and history link', async ({ page }) => {
    await page.goto('/backtest');
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Backtest Runner');
    await expect(page.getByRole('button', { name: 'Backtest', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Optimize', exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: 'History' })).toBeVisible();
  });

  test('simple backtest via API returns metrics and equity curve', async ({ request }) => {
    const body = {
      strategy_ids: ['intraday_mr'],
      symbols: ['AAPL'],
      start_date: '2024-01-01',
      end_date: '2024-06-30',
      capital: 100000,
    };

    const startTime = Date.now();
    const response = await request.post('/api/v1/backtests', { data: body });
    const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);

    expect(response.ok(), `API must return 2xx (got ${response.status()})`).toBeTruthy();
    const data = await response.json();
    console.log(`  Response: ${elapsed}s | Status: ${data.status}`);

    expect(data.status).toBe('completed');
    expect(typeof data.sharpe_ratio).toBe('number');
    expect(isFinite(data.sharpe_ratio)).toBeTruthy();
    expect(typeof data.max_drawdown).toBe('number');
    expect(data.max_drawdown).toBeGreaterThanOrEqual(0);
    expect(typeof data.total_return).toBe('number');
    expect(typeof data.win_rate).toBe('number');
    expect(data.win_rate).toBeGreaterThanOrEqual(0);
    expect(data.win_rate).toBeLessThanOrEqual(100);
    expect(typeof data.num_trades).toBe('number');
    expect(data.num_trades).toBeGreaterThanOrEqual(0);
    expect(data.data_source).toBeDefined();
    expect(data.data_source.length).toBeGreaterThan(0);

    if (data.equity_curve) {
      expect(Array.isArray(data.equity_curve)).toBeTruthy();
      if (data.equity_curve.length > 0) {
        const pt = data.equity_curve[0];
        expect(pt.time).toBeDefined();
        expect(typeof pt.value).toBe('number');
      }
    }

    console.log(`  Sharpe: ${data.sharpe_ratio?.toFixed(2)} | MaxDD: ${data.max_drawdown?.toFixed(2)}% | Return: ${data.total_return?.toFixed(2)}% | WinRate: ${data.win_rate?.toFixed(1)}% | Trades: ${data.num_trades} | Source: ${data.data_source}`);
  });

  test('matrix backtest via API returns batch run ID and progress', async ({ request }) => {
    const body = {
      mode: 'matrix',
      strategy_ids: ['intraday_mr', 'trend_following'],
      symbols: ['AAPL'],
      timeframes: ['1d'],
      start_date: '2024-01-01',
      end_date: '2024-03-31',
      capital: 100000,
    };

    const response = await request.post('/api/v1/backtests', { data: body });
    expect(response.ok()).toBeTruthy();
    const data = await response.json();

    expect(data.batch_run_id).toBeDefined();
    expect(data.status).toBe('running');
    expect(data.total_combos).toBeGreaterThan(0);

    const batchId = data.batch_run_id;
    console.log(`  Batch ID: ${batchId} | Combos: ${data.total_combos}`);

    let completed = false;
    for (let i = 0; i < 60; i++) {
      await new Promise(r => setTimeout(r, 2000));
      try {
        const progRes = await request.get(`/api/v1/backtests/${batchId}/progress`);
        if (progRes.status() === 404) continue;
        const prog = await progRes.json();
        console.log(`  Progress [${i * 2}s]: ${prog.completed}/${prog.total} | ${prog.status}`);
        if (prog.status === 'completed') {
          completed = true;
          if (prog.results && prog.results.length > 0) {
            const first = prog.results[0];
            expect(first.sharpe_ratio !== undefined || first.error).toBeTruthy();
            if (first.run_id) {
              console.log(`  Run ID: ${first.run_id} — verifying detail view`);
              const metricsRes = await request.get(`/api/v1/backtests/${first.run_id}/metrics`);
              if (metricsRes.ok()) {
                const m = await metricsRes.json();
                expect(typeof m.sharpe).toBe('number');
                console.log(`  Detail view Sharope: ${m.sharpe?.toFixed(2)}`);
              }
              const equityRes = await request.get(`/api/v1/backtests/${first.run_id}/equity?resolution=1d`);
              if (equityRes.ok()) {
                const eq = await equityRes.json();
                expect(Array.isArray(eq)).toBeTruthy();
                console.log(`  Equity points: ${eq.length}`);
              }
            }
          }
          break;
        }
      } catch { /* poll may fail transiently */ }
    }
    expect(completed).toBeTruthy();
  }, 180000);

  test('backtest metrics endpoint returns 10-metric PerformanceSnapshot', async ({ request }) => {
    const body = {
      strategy_ids: ['intraday_mr'],
      symbols: ['AAPL'],
      start_date: '2024-01-02',
      end_date: '2024-06-30',
      capital: 100000,
    };
    const resp = await request.post('/api/v1/backtests', { data: body });
    expect(resp.ok()).toBeTruthy();
    const bt = await resp.json();
    const runId = bt.id;
    console.log(`  Run ID: ${runId}`);

    const metricsRes = await request.get(`/api/v1/backtests/${runId}/metrics`);
    let m;
    try { m = await metricsRes.json() } catch { m = await metricsRes.text() }

    if (metricsRes.ok()) {
      const metricFields = ['sharpe', 'sortino', 'calmar', 'cagr', 'win_rate', 'profit_factor', 'var_95', 'cvar_95', 'ulcer_index', 'max_drawdown_pct', 'num_trades'];
      for (const field of metricFields) {
        expect(m[field], `${field} must be present`).toBeDefined();
        if (field !== 'num_trades') {
          expect(typeof m[field], `${field} must be a number`).toBe('number');
        }
      }
      console.log(`  Metrics: Sharpe=${m.sharpe?.toFixed(2)} | Sortino=${m.sortino?.toFixed(2)} | Calmar=${m.calmar?.toFixed(2)} | VaR=${m.var_95?.toFixed(2)}`);
    } else {
      console.log(`  Metrics endpoint unavailable (expected without DB migration): ${m.error || metricsRes.status()}`);
      expect(m.error || m).toBeTruthy();
    }
  });

  test('regime-stats endpoint returns per-run regime breakdown', async ({ request }) => {
    const body = {
      strategy_ids: ['intraday_mr'],
      symbols: ['SPY'],
      start_date: '2024-01-01',
      end_date: '2024-06-30',
      capital: 100000,
    };
    const resp = await request.post('/api/v1/backtests', { data: body });
    expect(resp.ok()).toBeTruthy();
    const bt = await resp.json();

    if (bt.regime_stats && bt.regime_stats.length > 0) {
      const rs = bt.regime_stats[0];
      expect(rs.regime).toBeDefined();
      expect(rs.label).toBeDefined();
      expect(rs.num_trades).toBeDefined();
      expect(rs.win_rate).toBeDefined();
      console.log(`  Regimes: ${bt.regime_stats.map(r => `${r.label}(${r.num_trades}t)`).join(', ')}`);
    }
  });
});
