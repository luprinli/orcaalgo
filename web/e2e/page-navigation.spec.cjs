const { test, expect } = require('@playwright/test');

async function setupAuth(page) {
  await page.addInitScript(() => {
    localStorage.setItem('orca_auth', JSON.stringify({
      token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.DuZ_rWj_5v4IQp9JSOCbVB7BqBQpKBQfBJyq6rzYhxI',
      user: 'e2e-test',
      expires: Date.now() + 86400000,
    }));
  });
}

async function mockApi(page) {
  await page.route('**/api/v1/**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });
  await page.route('**/ws**', async (route) => { await route.abort(); });
}

test.describe('E2E Page Navigation — All Routes', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
  });

  const pages = [
    '/', '/execution', '/backtest', '/backtest/history', '/strategies',
    '/accounts', '/charting', '/settings', '/integrations',
    '/calibrate', '/attribution', '/simulate',
    '/admin',
  ];

  for (const path of pages) {
    test(`GET ${path} → renders`, async ({ page }) => {
      const resp = await page.goto(path, { waitUntil: 'domcontentloaded', timeout: 10000 });
      await page.waitForTimeout(1000);
      const status = resp?.status();
      expect(status === 200 || status === 304).toBeTruthy();

      const main = page.locator('[id="main-content"]');
      await expect(main.first()).toBeVisible({ timeout: 8000 });
    });
  }
});

test.describe('Backtest Runner UI — Interactive', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
  });

  test('backtest form renders with run modes', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await expect(page.getByLabel('Matrix')).toBeVisible();
    await expect(page.getByLabel('Single')).toBeVisible();
    await expect(page.getByLabel('Orch')).toBeVisible();
  });

  test('switch to optimize mode shows optimize fields', async ({ page }) => {
    test.fixme(true, 'Optimize mode replaced by light_optimize checkbox in matrix mode');
  });

  test('strategy selector is present', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await expect(page.locator('text=Strategies')).toBeVisible();
  });

  test('backtest history page loads', async ({ page }) => {
    await page.goto('/backtest/history?view=history', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(2000);
    await expect(page.locator('h1')).toContainText('Backtest History');
  });
});

test.describe('Dashboard Pages', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
  });

  test('home dashboard renders', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(500);
    await expect(page.locator('h1').first()).toBeVisible();
  });

  test('settings page renders', async ({ page }) => {
    await page.goto('/settings', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);
    await expect(page.locator('h1').first()).toContainText('Settings');
  });

  test('integrations page renders', async ({ page }) => {
    await page.goto('/integrations', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);
    await expect(page.locator('h1').first()).toBeVisible();
  });
});
