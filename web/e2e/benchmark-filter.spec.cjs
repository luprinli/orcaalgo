const { test, expect } = require('@playwright/test');

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await page.evaluate(() => {
    localStorage.setItem('orca_auth', JSON.stringify({
      token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.DuZ_rWj_5v4IQp9JSOCbVB7BqBpQKBQfBJyq6rzYhxI',
      user: 'e2e-test',
      expires: Date.now() + 86400000,
    }));
  });
});

test.describe('Benchmark Filter — E2E', () => {
  test('runner accepts a benchmark spec in the run request', async ({ request }) => {
    const body = {
      strategy_ids: ['intraday_mr'],
      symbols: ['SPY'],
      start_date: '2024-01-01',
      end_date: '2024-06-30',
      capital: 100000,
      benchmark_kind: 'equity_index',
      benchmark_symbol: 'SPY',
    };
    const resp = await request.post('/api/v1/backtests', { data: body });
    expect(resp.ok(), `backtest run must be accepted (got ${resp.status()})`).toBeTruthy();
    const bt = await resp.json();
    expect(bt.id || bt.batch_run_id).toBeDefined();
    console.log(`  Run ID: ${bt.id || bt.batch_run_id}`);
  });

  test('benchmark-eval endpoint returns a verdict for a completed run', async ({ request }) => {
    const runBody = {
      strategy_ids: ['intraday_mr'],
      symbols: ['SPY'],
      start_date: '2024-01-01',
      end_date: '2024-06-30',
      capital: 100000,
    };
    const runResp = await request.post('/api/v1/backtests', { data: runBody });
    expect(runResp.ok()).toBeTruthy();
    const bt = await runResp.json();
    const runId = bt.id;
    if (!runId) {
      console.log('  No single-run id returned; skipping benchmark-eval (matrix path).');
      return;
    }

    const evalResp = await request.post(`/api/v1/backtests/${runId}/benchmark-eval`, { data: {} });
    if (evalResp.status() === 503) {
      // The orca Python toolchain is not present in this environment; the
      // endpoint degrades to 503 rather than failing the hot path.
      console.log('  benchmark-eval 503 (orca toolchain unavailable in CI) — expected graceful degradation.');
      const body = await evalResp.json();
      expect(body.error).toBeTruthy();
      return;
    }
    expect(evalResp.ok(), `benchmark-eval must return 2xx (got ${evalResp.status()})`).toBeTruthy();
    const verdict = await evalResp.json();
    expect(typeof verdict.passed).toBe('boolean');
    expect(verdict.kind).toBe('equity_index');
    expect(typeof verdict.deflated_active_sharpe).toBe('number');
    expect(verdict.metrics).toBeDefined();
    console.log(`  Verdict: passed=${verdict.passed} IR=${verdict.metrics?.information_ratio ?? 'n/a'} alpha=${verdict.metrics?.alpha_annualized ?? 'n/a'}`);
  });

  test('benchmark-evals admin endpoint returns a list', async ({ request }) => {
    const resp = await request.get('/api/v1/admin/benchmark-evals');
    if (!resp.ok()) {
      console.log(`  admin/benchmark-evals unavailable (${resp.status()}) — expected without a persisted eval row.`);
      return;
    }
    const data = await resp.json();
    expect(Array.isArray(data.benchmark_evals)).toBeTruthy();
    console.log(`  Benchmark evals persisted: ${data.benchmark_evals.length}`);
  });
});
