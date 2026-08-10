const { test, expect } = require('@playwright/test');

async function setupAuth(page) {
  await page.addInitScript(() => {
    localStorage.setItem('orca_auth', JSON.stringify({
      token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test',
      username: 'admin',
      expires_at: Date.now() + 86400000,
    }));
  });
}

async function mockApi(page) {
  await page.route('**/api/v1/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });
  await page.route('**/ws**', async (route) => { await route.abort(); });
}

test.describe('BacktestRunner — Matrix, Single, Optimize Modes', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
  });

  test('mode radio buttons render', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await expect(page.getByLabel('Matrix')).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel('Single')).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel('Optimize')).toBeVisible({ timeout: 5000 });
  });

  test('switching to Optimize mode shows objective selector', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await page.getByLabel('Optimize').click();
    await page.waitForTimeout(500);

    await expect(page.getByText('Objective')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Max Combinations')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Train Years')).toBeVisible({ timeout: 5000 });

    const btn = page.getByRole('button', { name: 'Run Optimization' });
    await expect(btn).toBeVisible();
  });

  test('switching to Matrix mode shows timeframes', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await page.getByLabel('Matrix').click();
    await page.waitForTimeout(500);

    const main = page.locator('[id="main-content"]');
    await expect(main.getByText('Timeframes')).toBeVisible({ timeout: 5000 });
  });

  test('switching to Single mode hides timeframes', async ({ page }) => {
    test.fixme(true, 'Timeframes section shared across modes since Phase 2 integration — test needs rewrite');
  });

  test('strategy checkboxes are present', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await expect(page.locator('[id="main-content"]').getByText('grid trading', { exact: false })).toBeVisible({ timeout: 5000 });
  });
});
