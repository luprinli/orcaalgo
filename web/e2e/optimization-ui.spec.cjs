// @ts-check
const { test, expect } = require('@playwright/test');

test.describe('BacktestRunner — Matrix, Single, Orch Modes', () => {
  test('mode radio buttons render', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await expect(page.getByLabel('Matrix')).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel('Single')).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel('Orch')).toBeVisible({ timeout: 5000 });
  });

  test('switching to Optimize mode shows objective selector', async ({ page }) => {
    test.fixme(true, 'Optimize mode replaced by light_optimize checkbox in Matrix mode');
  });

  test('switching to Matrix mode shows timeframes', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await expect(page.getByText('Timeframes')).toBeVisible({ timeout: 5000 });
  });

  test('switching to Single mode hides timeframes', async ({ page }) => {
    test.fixme(true, 'Timeframes section shared across modes since Phase 2 integration');
  });

  test('strategy dropdown present', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Strategies')).toBeVisible({ timeout: 5000 });
  });
});
