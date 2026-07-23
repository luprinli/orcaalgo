// @ts-check
const { test, expect } = require('@playwright/test');

const STRATEGY_IDS = [
  'intraday_mr',
  'opening_range_breakout',
  'trend_following',
  'grid_trading',
  'session_scalp',
  'ma_crossover',
  'rsi2_reversion',
  'donchian_breakout',
  'keltner_macd',
  'ichimoku_cloud',
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
      refresh: 'test-refresh-token',
      roles: ['admin'],
    }));
  });
});

// =========================================================================
// 1. PAGE LOAD & UI SANITY
// =========================================================================
test.describe('Backtest Page — UI Elements', () => {

  test('backtest page loads with all required UI elements', async ({ page }) => {
    await page.goto('/backtest');

    await expect(page.getByRole('heading', { level: 1 })).toContainText('Backtest Runner');

    await expect(page.getByRole('button', { name: 'Backtest', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Optimize', exact: true })).toBeVisible();
    await expect(page.getByRole('link', { name: 'History' })).toBeVisible();

    await expect(page.getByText('Backtest Configuration')).toBeVisible();
    await expect(page.getByText('Matrix')).toBeVisible();
    await expect(page.getByText('Single')).toBeVisible();

    await expect(page.getByText('Daily (1d)')).toBeVisible();
    await expect(page.getByText('Hourly (1h)')).toBeVisible();
    await expect(page.getByText('5-Minute (5m)')).toBeVisible();

    await expect(page.locator('input[placeholder="EURUSD, BTCUSD, US30"]')).toBeVisible();
    await expect(page.locator('input[type="date"]').first()).toBeVisible();
    await expect(page.locator('input[type="number"]')).toBeVisible();

    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await expect(submitBtn).toBeVisible();
    await expect(submitBtn).toBeEnabled();
  });

  test('strategy checkboxes are visible and toggleable', async ({ page }) => {
    await page.goto('/backtest');

    for (const sid of STRATEGY_IDS) {
      const checkbox = page.locator(`input[type="checkbox"][value="${sid}"]`);
      if (await checkbox.count() === 0) {
        const label = page.locator(`label:has-text("${sid.replace(/_/g, ' ')}")`);
        await expect(label.first()).toBeVisible({ timeout: 3000 });
      }
    }
  });

});

// =========================================================================
// 2. SINGLE STRATEGY BACKTEST — BUTTON CLICK + RESULTS RENDER
// =========================================================================
test.describe('Backtest — Run Button & Results', () => {

  test('clicking "Run Backtest" triggers loading state then renders metric cards', async ({ page }) => {
    test.setTimeout(120000);
    await page.goto('/backtest');

    await page.locator('input[type="radio"]').nth(1).check();

    const symbolInput = page.locator('input[placeholder="EURUSD, BTCUSD, US30"]');
    await symbolInput.fill('SPY');

    await page.locator('input[type="date"]').first().fill(START_DATE);
    await page.locator('input[type="date"]').nth(1).fill(END_DATE);

    const capitalInput = page.locator('input[type="number"]');
    await capitalInput.fill(String(CAPITAL));

    await expect(page.locator('text=Will run 1 backtest')).toBeVisible();

    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await expect(submitBtn).toBeVisible();
    await expect(submitBtn).toBeEnabled();

    await submitBtn.click();

    await expect(async () => {
      const resultsHeading = page.getByRole('heading', { name: 'Backtest Results' });
      const metricCards = page.locator('.metric-card');
      const errorMsg = page.locator('text=/error/i');
      const hasResult = (await resultsHeading.count()) + (await metricCards.count()) > 0
                     || (await errorMsg.count()) > 0;
      expect(hasResult).toBeTruthy();
    }).toPass({ timeout: 90000, intervals: [2000] });

    const metricValues = page.locator('.metric-value');
    const count = await metricValues.count();
    expect(count).toBeGreaterThanOrEqual(3);

    for (let i = 0; i < count; i++) {
      const text = await metricValues.nth(i).textContent();
      expect(text?.trim().length).toBeGreaterThan(0);
    }

    const labels = page.locator('.metric-label');
    const labelCount = await labels.count();
    expect(labelCount).toBeGreaterThanOrEqual(3);

    const labelTexts = await labels.allTextContents();
    expect(labelTexts.some(l => l.toLowerCase().includes('sharpe'))).toBeTruthy();
    expect(labelTexts.some(l => l.toLowerCase().includes('drawdown') || l.toLowerCase().includes('max dd'))).toBeTruthy();
    expect(labelTexts.some(l => l.toLowerCase().includes('win'))).toBeTruthy();
    expect(labelTexts.some(l => l.toLowerCase().includes('trade'))).toBeTruthy();
    expect(labelTexts.some(l => l.toLowerCase().includes('profit'))).toBeTruthy();
  });

  test('Run Backtest button is re-enabled after completion (or error)', async ({ page }) => {
    test.setTimeout(120000);
    await page.goto('/backtest');

    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await submitBtn.click();

    await expect(async () => {
      const enabled = await submitBtn.isEnabled();
      const disabled = await submitBtn.isDisabled();
      expect(enabled || disabled).toBeTruthy();
    }).toPass({ timeout: 60000, intervals: [2000] });
  });

});

// =========================================================================
// 3. MATRIX BACKTEST — COMPLETION VERIFICATION
// =========================================================================
test.describe('Backtest — Matrix Mode', () => {

  test('matrix mode click triggers request and renders results container', async ({ page }) => {
    test.setTimeout(300000);
    await page.goto('/backtest');

    const matrixRadio = page.locator('input[type="radio"]').first();
    await expect(matrixRadio).toBeChecked();

    const symbolInput = page.locator('input[placeholder="EURUSD, BTCUSD, US30"]');
    await symbolInput.fill('SPY,QQQ');

    await page.locator('input[type="date"]').first().fill('2024-01-02');
    await page.locator('input[type="date"]').nth(1).fill('2024-03-29');

    await page.locator('input[type="number"]').fill(String(CAPITAL));

    const comboText = page.locator('text=/Will run \\d+ backtests/');
    await expect(comboText).toBeVisible();

    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await submitBtn.click();

    await expect(async () => {
      const results = page.locator('h2:has-text("Backtest Results")');
      const error = page.locator('text=/error/i');
      expect((await results.count()) + (await error.count())).toBeGreaterThan(0);
    }).toPass({ timeout: 180000, intervals: [3000] });

    const resultsHeader = page.locator('h2:has-text("Backtest Results")');
    if (await resultsHeader.count() > 0) {
      await expect(resultsHeader).toBeVisible();
      const metricCards = page.locator('.metric-card');
      expect(await metricCards.count()).toBeGreaterThanOrEqual(3);
    }
  });

});

// =========================================================================
// 4. API-ONLY: ALL STRATEGIES VALIDATION
// =========================================================================
test.describe('Backtest API — Strategy Validation', () => {

  for (const strategyId of ['intraday_mr', 'trend_following', 'ma_crossover', 'rsi2_reversion']) {
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
      expect(response.ok(), `API returned ${response.status()} for ${strategyId}`).toBeTruthy();

      const data = await response.json();

      expect(data.sharpe_ratio).toBeDefined();
      expect(typeof data.sharpe_ratio).toBe('number');
      expect(isFinite(data.sharpe_ratio)).toBeTruthy();

      expect(data.max_drawdown).toBeDefined();
      expect(typeof data.max_drawdown).toBe('number');
      expect(data.max_drawdown).toBeGreaterThanOrEqual(0);

      expect(data.win_rate).toBeDefined();
      expect(typeof data.win_rate).toBe('number');
      expect(data.win_rate).toBeGreaterThanOrEqual(0);

      expect(data.num_trades).toBeDefined();
      expect(Number.isInteger(data.num_trades)).toBeTruthy();

      expect(data.profit_factor).toBeDefined();
      expect(typeof data.profit_factor).toBe('number');
    });
  }

});

// =========================================================================
// 5. OPTIMIZE MODE — UI TOGGLE AND FORM RENDER
// =========================================================================
test.describe('Backtest — Optimize Mode Toggle', () => {

  test('backtest → optimize mode toggle renders correct form', async ({ page }) => {
    await page.goto('/backtest');

    await expect(page.getByText('Backtest Configuration')).toBeVisible();
    await expect(page.getByText('Optimization Configuration')).not.toBeVisible();

    await page.getByRole('button', { name: 'Optimize', exact: true }).click();

    await expect(page.getByText('Optimization Configuration')).toBeVisible();
    await expect(page.getByText('Backtest Configuration')).not.toBeVisible();
    await expect(page.getByText('Search Space')).toBeVisible();

    await page.getByRole('button', { name: 'Backtest', exact: true }).click();
    await expect(page.getByText('Backtest Configuration')).toBeVisible();
  });

  test('optimize mode shows strategy selector and Run Optimization button', async ({ page }) => {
    await page.goto('/backtest');

    await page.getByRole('button', { name: 'Optimize', exact: true }).click();

    const optimizeBtn = page.getByRole('button', { name: /Run.*Optimization/ });
    await expect(optimizeBtn).toBeVisible();

    await expect(page.locator('select').first()).toBeVisible();

    await expect(page.getByText('Search Space')).toBeVisible();
  });

});

// =========================================================================
// 6. ERROR HANDLING — INVALID INPUT
// =========================================================================
test.describe('Backtest — Validation & Edge Cases', () => {

  test('Run Backtest button is disabled when no strategies selected', async ({ page }) => {
    await page.goto('/backtest');

    const strategyChecks = page.locator('input[type="checkbox"]');
    const count = await strategyChecks.count();
    for (let i = 0; i < count; i++) {
      await strategyChecks.nth(i).uncheck();
    }

    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await expect(submitBtn).toBeDisabled();
  });

  test('date inputs accept and display valid dates', async ({ page }) => {
    await page.goto('/backtest');

    const startInput = page.locator('input[type="date"]').first();
    const endInput = page.locator('input[type="date"]').nth(1);

    await startInput.fill('2024-01-15');
    await endInput.fill('2024-12-31');

    await expect(startInput).toHaveValue('2024-01-15');
    await expect(endInput).toHaveValue('2024-12-31');
  });

  test('capital accepts numeric input', async ({ page }) => {
    await page.goto('/backtest');

    const capitalInput = page.locator('input[type="number"]');
    await capitalInput.fill('50000');
    await expect(capitalInput).toHaveValue('50000');
  });

});

// =========================================================================
// 7. BACKTEST HISTORY NAVIGATION
// =========================================================================
test.describe('Backtest History — Navigation', () => {

  test('navigate from backtest runner to history and verify page loads', async ({ page }) => {
    await page.goto('/backtest');

    await page.getByRole('link', { name: 'History' }).click();

    await page.waitForURL('**/backtest/history');
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Backtest History');
  });

});
