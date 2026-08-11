// @ts-check
const { test, expect } = require('@playwright/test');

test.describe('Strategy Pages Full Verification', () => {
  test('Backtest page shows strategy selector from API', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'networkidle', timeout: 15000 });
    await expect(page.getByText('Backtest Runner')).toBeVisible({ timeout: 5000 });

    const strategyLabels = page.locator('text=Strategies');
    await expect(strategyLabels.first()).toBeVisible({ timeout: 5000 });
  });
});
