const { test, expect } = require('@playwright/test');

async function setupAuth(page) {
  await page.addInitScript(() => {
    const auth = { token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test', username: 'admin', expires_at: Date.now() + 86400000 };
    localStorage.setItem('orca_auth', JSON.stringify(auth));
  });
}

test.describe('BacktestRunner — Backtest + Optimization Modes', () => {
  test.beforeEach(async ({ page }) => { await setupAuth(page); });

  test('Mode tabs render and toggle', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'networkidle' });

    const backtestTab = page.locator('.app-main button').filter({ hasText: 'Backtest' }).first();
    const optimizeTab = page.locator('.app-main button').filter({ hasText: 'Optimize' }).first();
    await expect(backtestTab).toBeVisible({ timeout: 5000 });
    await expect(optimizeTab).toBeVisible({ timeout: 5000 });

    await optimizeTab.click();
    await page.waitForTimeout(500);
    await expect(page.locator('h2').filter({ hasText: 'Optimization Configuration' })).toBeVisible({ timeout: 5000 });

    await backtestTab.click();
    await page.waitForTimeout(500);
    await expect(page.locator('h2').filter({ hasText: 'Backtest Configuration' })).toBeVisible({ timeout: 5000 });
  });

  test('Optimization form shows search space for trend', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'networkidle' });
    await page.locator('.app-main').getByRole('button', { name: 'Optimize' }).click();

    await expect(page.getByText('Optimization Configuration')).toBeVisible();
    await expect(page.getByText('Search Space')).toBeVisible();
    await expect(page.getByText('fast_period')).toBeVisible();
    await expect(page.getByText('slow_period')).toBeVisible();
    await expect(page.getByText('atr_period')).toBeVisible();
    await expect(page.getByText('atr_multiplier')).toBeVisible();

    const btn = page.locator('.app-main').getByRole('button', { name: /Run/ });
    await expect(btn).toBeVisible();
  });

  test('Strategy switch changes search space', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'networkidle' });
    await page.locator('.app-main').getByRole('button', { name: 'Optimize' }).click();
    await page.waitForTimeout(500);

    const selects = page.locator('.app-main select');
    const strategySelect = selects.first();
    await strategySelect.selectOption('opening_range_breakout');
    await page.waitForTimeout(500);

    await expect(page.locator('table').getByText('range_minutes')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('table').getByText('entry_buffer_pct')).toBeVisible({ timeout: 5000 });
  });

  test('Objective dropdown shows all 6 options', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'networkidle' });
    await page.locator('.app-main').getByRole('button', { name: 'Optimize' }).click();

    const objectiveSelect = page.locator('.app-main select').filter({ hasText: 'Sharpe / Drawdown' });
    await expect(objectiveSelect).toBeVisible();
    const options = await objectiveSelect.locator('option').allTextContents();
    expect(options.length).toBe(6);
  });

  test('Backtest mode form still works', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'networkidle' });

    await expect(page.getByText('Backtest Configuration')).toBeVisible();
    const submitBtn = page.locator('.app-main').getByRole('button', { name: /Run/ });
    await expect(submitBtn).toBeVisible();
  });
});
