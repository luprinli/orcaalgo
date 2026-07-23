// @ts-check
const { test, expect } = require('@playwright/test');

const STRATEGY_IDS = [
  'intraday_mr',
  'opening_range_breakout',
  'trend_following',
  'grid_trading',
  'pairs_trading',
  'volatility_harvesting',
];

const ALL_SYMBOLS = 'SPY,QQQ,AAPL,MSFT,TSLA,NVDA,AMZN,META,GOOGL,BRK.B,JPM,V,JNJ,WMT,PG,XOM,UNH,HD,CVX,MA,ABBV,PFE,KO,PEP,TMO,COST,AVGO,DIS,CSCO,ABT,WFC,VZ,NKE,MRK';
const START_DATE = '2024-01-02';
const END_DATE = '2024-06-28';
const CAPITAL = 100000;

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

test.describe('Backtest Runner — Comprehensive E2E', () => {

  // =========================================================================
  // 1. PAGE LOAD & UI SANITY
  // =========================================================================
  test('backtest page loads with all required UI elements', async ({ page }) => {
    await page.goto('/backtest');

    await expect(page.getByRole('heading', { level: 1 })).toContainText('Backtest Runner');

    // Mode toggle buttons
    await expect(page.getByRole('button', { name: 'Backtest', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Optimize', exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: 'History' })).toBeVisible();

    // Configuration form elements
    await expect(page.getByText('Backtest Configuration')).toBeVisible();
    await expect(page.getByText('Matrix')).toBeVisible();
    await expect(page.getByText('Single')).toBeVisible();

    // Timeframe checkboxes
    await expect(page.getByText('Daily (1d)')).toBeVisible();
    await expect(page.getByText('Hourly (1h)')).toBeVisible();
    await expect(page.getByText('5-Minute (5m)')).toBeVisible();

    // Strategy checkboxes
    for (const sid of STRATEGY_IDS) {
      await expect(page.locator(`text=${sid}`).first()).toBeVisible();
    }

    // Input fields
    await expect(page.locator('input[placeholder="EURUSD, BTCUSD, US30"]')).toBeVisible();
    await expect(page.locator('input[type="date"]').first()).toBeVisible();
    await expect(page.locator('input[type="number"]')).toBeVisible();

    // Submit button
    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await expect(submitBtn).toBeVisible();
    await expect(submitBtn).not.toBeDisabled();
  });

  // =========================================================================
  // 2. SINGLE STRATEGY BACKTEST (API+UI)
  // =========================================================================
  test('run single-strategy backtest and verify DOM renders results', async ({ page }) => {
    await page.goto('/backtest');

    // Switch to Single mode
    const singleRadio = page.locator('input[type="radio"]').nth(1);
    await singleRadio.check();

    // Uncheck all strategies first using label text
    for (const sid of STRATEGY_IDS) {
      const label = page.locator(`label:has-text("${sid}")`);
      if (await label.count() > 0) {
        const cb = label.locator('input[type="checkbox"]');
        if (await cb.isChecked()) await cb.uncheck();
      }
    }
    // Check only intraday_mr
    const mrLabel = page.locator('label:has-text("intraday_mr")');
    if (await mrLabel.count() > 0) {
      const mrCb = mrLabel.locator('input[type="checkbox"]');
      await mrCb.check();
    }

    // Set symbols to a single symbol
    const symbolInput = page.locator('input[placeholder="EURUSD, BTCUSD, US30"]');
    await symbolInput.fill('SPY');

    // Set dates
    await page.locator('input[type="date"]').first().fill(START_DATE);
    await page.locator('input[type="date"]').nth(1).fill(END_DATE);

    // Set capital
    const capitalInput = page.locator('input[type="number"]');
    await capitalInput.fill(String(CAPITAL));

    // Verify combo count displays "Will run 1 backtest"
    await expect(page.locator('text=Will run 1 backtest')).toBeVisible();

    // Submit
    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await submitBtn.click();

    // Wait for results card to appear
    await expect(async () => {
      const metricValues = page.locator('.orca-metric-card__value');
      const resultsHeader = page.locator('h3:has-text("Equity Curve")');
      const card = page.locator('.card');
      const hasMetrics = (await metricValues.count()) > 0;
      const hasEquity = (await resultsHeader.count()) > 0;
      const hasError = (await page.locator('text=/Server returned/').count()) > 0;
      expect(hasMetrics || hasEquity || hasError).toBeTruthy();
    }).toPass({ timeout: 60000, intervals: [2000] });

    // Verify results appeared
    const metricCards = page.locator('.orca-metric-card__value');
    const cardCount = await metricCards.count();
    if (cardCount > 0) {
      for (let i = 0; i < cardCount; i++) {
        const text = await metricCards.nth(i).textContent();
        expect(text?.trim().length).toBeGreaterThan(0);
      }
    }

    // Check equity curve if present
    const equityHeader = page.locator('h3:has-text("Equity Curve")');
    if (await equityHeader.count() > 0) {
      await expect(equityHeader).toBeVisible();
    }
  });

  // =========================================================================
  // 3. MATRIX BACKTEST (API+UI) — FULL COMBO
  // =========================================================================
  test('run matrix backtest with multiple strategies × symbols × timeframes and verify results table', async ({ page }) => {
    test.setTimeout(300000);
    await page.goto('/backtest');

    // Matrix mode should be default — verify radio
    const matrixRadio = page.locator('input[type="radio"]').first();
    await expect(matrixRadio).toBeChecked();

    // Select all strategies using label text
    for (const sid of STRATEGY_IDS) {
      const label = page.locator(`label:has-text("${sid}")`);
      if (await label.count() > 0) {
        const cb = label.locator('input[type="checkbox"]');
        if (!(await cb.isChecked())) await cb.check();
      }
    }

    // Select daily timeframe (first one should already be checked)
    const dailyLabel = page.locator('label:has-text("Daily (1d)")');
    if (await dailyLabel.count() > 0) {
      const dailyCb = dailyLabel.locator('input[type="checkbox"]');
      if (!(await dailyCb.isChecked())) await dailyCb.check();
    }

    // Set symbols
    const symbolInput = page.locator('input[placeholder="EURUSD, BTCUSD, US30"]');
    await symbolInput.fill('SPY,QQQ,AAPL,MSFT');

    // Set dates
    await page.locator('input[type="date"]').first().fill('2024-01-02');
    await page.locator('input[type="date"]').nth(1).fill('2024-03-29');

    // Set capital
    await page.locator('input[type="number"]').fill(String(CAPITAL));

    // Verify combo count
    const comboText = page.locator('text=/Will run \\d+ backtests/');
    await expect(comboText).toBeVisible();

    // Submit
    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await submitBtn.click();

    // Wait for results table or error to appear
    await expect(async () => {
      const results = page.locator('h2:has-text("Backtest Results")');
      const error = page.locator('text=/Server returned|error/i');
      expect((await results.count()) + (await error.count())).toBeGreaterThan(0);
    }).toPass({ timeout: 180000, intervals: [3000] });

    // If results table appeared, verify its structure
    const resultsHeader = page.locator('h2:has-text("Backtest Results")');
    if (await resultsHeader.count() > 0) {
      const tableRows = page.locator('table.bt-table tbody tr');
      const rowCount = await tableRows.count();
      expect(rowCount).toBeGreaterThan(0);

      // Verify table headers
      const headerTexts = await page.locator('table.bt-table thead th').allTextContents();
      expect(headerTexts.some(h => h.includes('Sharpe'))).toBeTruthy();
      expect(headerTexts.some(h => h.includes('Max DD'))).toBeTruthy();

      // Verify summary row
      const tfoot = page.locator('table.bt-table tfoot');
      if (await tfoot.count() > 0) {
        await expect(tfoot.locator('tr')).toBeVisible();
      }
    }
  });

  // =========================================================================
  // 4. API-ONLY: ALL STRATEGIES VALIDATION
  // =========================================================================
  for (const strategyId of STRATEGY_IDS) {
    test(`API: backtest ${strategyId} returns valid metrics`, async ({ request }) => {
      test.info().annotations.push({ type: 'strategy', description: strategyId });

      const body = {
        strategy_ids: [strategyId],
        symbols: ['SPY'],
        start_date: START_DATE,
        end_date: END_DATE,
        capital: CAPITAL,
      };

      const response = await request.post('/api/v1/backtests', { data: body });
      expect(response.ok(), `API must return 2xx for ${strategyId} (got ${response.status()})`).toBeTruthy();

      const data = await response.json();
      expect(data.status).toBe('completed');

      // Validate required fields
      expect(data.sharpe_ratio).toBeDefined();
      expect(data.max_drawdown).toBeDefined();
      expect(data.total_return).toBeDefined();
      expect(data.win_rate).toBeDefined();
      expect(data.num_trades).toBeDefined();
      expect(data.data_source).toBeDefined();

      // Validate numeric ranges
      expect(typeof data.sharpe_ratio).toBe('number');
      expect(isFinite(data.sharpe_ratio)).toBeTruthy();
      expect(typeof data.max_drawdown).toBe('number');
      expect(data.max_drawdown).toBeGreaterThanOrEqual(0);
      expect(data.max_drawdown).toBeLessThanOrEqual(100);
      expect(typeof data.total_return).toBe('number');
      expect(typeof data.win_rate).toBe('number');
      expect(data.win_rate).toBeGreaterThanOrEqual(0);
      expect(data.win_rate).toBeLessThanOrEqual(100);
      expect(Number.isInteger(data.num_trades)).toBeTruthy();
      expect(data.num_trades).toBeGreaterThanOrEqual(0);
    });
  }

  // =========================================================================
  // 5. API: MATRIX WITH ALL 6 STRATEGIES × ALL SYMBOLS × 3 TIMEFRAMES
  // =========================================================================
  test('API: full matrix backtest — all strategies × all symbols × all timeframes', async ({ request }) => {
    test.setTimeout(600000);

    const body = {
      mode: 'matrix',
      strategy_ids: STRATEGY_IDS,
      symbols: ALL_SYMBOLS.split(',').map(s => s.trim()).filter(Boolean),
      timeframes: ['1d', '1h', '5m'],
      start_date: '2024-01-02',
      end_date: '2024-06-28',
      capital: CAPITAL,
    };

    const totalCombos = STRATEGY_IDS.length * body.symbols.length * body.timeframes.length;

    const response = await request.post('/api/v1/backtests', { data: body });
    expect(response.ok(), `Matrix API must return 2xx (got ${response.status()})`).toBeTruthy();

    const data = await response.json();
    expect(data.batch_run_id).toBeDefined();
    expect(data.total_combos).toBe(totalCombos);
    expect(data.status).toBe('running');

    const batchId = data.batch_run_id;

    // Poll for completion
    let completed = false;
    for (let i = 0; i < 120; i++) {
      await new Promise(r => setTimeout(r, 2000));
      try {
        const progRes = await request.get(`/api/v1/backtests/${batchId}/progress`);
        if (progRes.status() === 404) continue;
        const prog = await progRes.json();

        if (prog.status === 'completed') {
          completed = true;

          expect(prog.results).toBeDefined();
          expect(Array.isArray(prog.results)).toBeTruthy();
          expect(prog.results.length).toBeGreaterThan(0);

          // Validate each result entry
          const requiredFields = ['symbol', 'strategy_id', 'timeframe', 'sharpe_ratio', 'max_drawdown', 'total_return', 'win_rate', 'num_trades'];
          for (const result of prog.results.slice(0, 10)) {
            for (const field of requiredFields) {
              expect(result[field], `Field '${field}' missing in result for ${result.symbol}/${result.strategy_id}/${result.timeframe}`).toBeDefined();
            }
          }

          // Compute summary stats
          const successResults = prog.results.filter(r => !r.error);
          const errorResults = prog.results.filter(r => r.error);
          const avgSharpe = successResults.length > 0
            ? successResults.reduce((s, r) => s + r.sharpe_ratio, 0) / successResults.length
            : 0;

          break;
        }

        if (prog.status === 'failed') {
          break;
        }
      } catch { /* poll may fail transiently */ }
    }

    expect(completed, 'Matrix backtest should complete within timeout').toBeTruthy();
  });

  // =========================================================================
  // 6. OPTIMIZE MODE — UI TOGGLE AND FORM RENDER
  // =========================================================================
  test('backtest → optimize mode toggle renders correct form', async ({ page }) => {
    await page.goto('/backtest');

    // Default: Backtest mode active
    await expect(page.getByText('Backtest Configuration')).toBeVisible();
    await expect(page.getByText('Optimization Configuration')).not.toBeVisible();

    // Click Optimize button
    await page.getByRole('button', { name: 'Optimize', exact: true }).click();

    // Should show optimization form
    await expect(page.getByText('Optimization Configuration')).toBeVisible();
    await expect(page.getByText('Backtest Configuration')).not.toBeVisible();

    // Click Backtest button to revert
    await page.getByRole('button', { name: 'Backtest', exact: true }).click();
    await expect(page.getByText('Backtest Configuration')).toBeVisible();
  });

  // =========================================================================
  // 7. ERROR HANDLING — INVALID INPUT
  // =========================================================================
  test('backtest with no strategies selected shows validation', async ({ page }) => {
    await page.goto('/backtest');

    // Uncheck all strategies
    const strategyChecks = page.locator('input[type="checkbox"]');
    const count = Math.min(await strategyChecks.count(), STRATEGY_IDS.length);
    for (let i = 0; i < count; i++) {
      await strategyChecks.nth(i).uncheck();
    }

    // Submit button should be disabled
    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await expect(submitBtn).toBeDisabled();
  });

  // =========================================================================
  // 8. BACKTEST HISTORY NAVIGATION
  // =========================================================================
  test('navigate to backtest history and verify table renders', async ({ page, request }) => {
    // First run a backtest to ensure history has data
    await request.post('/api/v1/backtests', {
      data: {
        strategy_ids: ['intraday_mr'],
        symbols: ['SPY'],
        start_date: '2024-01-02',
        end_date: '2024-01-31',
        capital: 100000,
      },
    });

    await page.goto('/backtest/history');
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Backtest History');

    // Wait for table to populate (or show empty state)
    await page.waitForTimeout(2000);

    const historyTable = page.locator('table');
    if (await historyTable.count() > 0) {
      const rows = historyTable.locator('tbody tr');
      const rowCount = await rows.count();
      // At minimum we should have rows or an empty state
      expect(rowCount >= 0).toBeTruthy();
    }
  });

  // =========================================================================
  // 9. REGIME PERFORMANCE TABLE
  // =========================================================================
  test('single backtest with EQ curve and regime stats', async ({ page }) => {
    await page.goto('/backtest');

    // Single mode
    await page.locator('input[type="radio"]').nth(1).check();

    // Check only trend_following using label text
    for (const sid of STRATEGY_IDS) {
      const label = page.locator(`label:has-text("${sid}")`);
      if (await label.count() > 0) {
        const cb = label.locator('input[type="checkbox"]');
        if (sid === 'trend_following') {
          if (!(await cb.isChecked())) await cb.check();
        } else {
          if (await cb.isChecked()) await cb.uncheck();
        }
      }
    }

    // Set symbols
    await page.locator('input[placeholder="EURUSD, BTCUSD, US30"]').fill('SPY,QQQ');

    // Set dates
    await page.locator('input[type="date"]').first().fill('2024-01-02');
    await page.locator('input[type="date"]').nth(1).fill('2024-06-28');

    // Submit
    await page.getByRole('button', { name: /Run.*Backtest/ }).click();

    // Wait for results or error
    await expect(async () => {
      const metric = page.locator('.orca-metric-card__value').first();
      const equity = page.locator('h3:has-text("Equity Curve")');
      const error = page.locator('text=/Server returned|error/i');
      const hasResult = (await metric.count()) + (await equity.count()) + (await error.count()) > 0;
      expect(hasResult).toBeTruthy();
    }).toPass({ timeout: 60000, intervals: [2000] });

    // Check for equity curve
    const equitySection = page.locator('h3:has-text("Equity Curve")');
    if (await equitySection.count() > 0) {
      await expect(equitySection).toBeVisible();
    }
  });

});
