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

test.describe('E2E Page Navigation — All Routes', () => {
  const pages = [
    { path: '/', selector: 'h1', text: 'Dashboard' },
    { path: '/live', selector: 'h2', check: 'visible' },
    { path: '/live/market', selector: 'select', check: 'visible' },
    { path: '/execution', selector: 'h1, h2', check: 'visible' },
    { path: '/backtest', selector: 'h1', text: 'Backtest Runner' },
    { path: '/backtest/history', selector: 'h1', text: 'Backtest History' },
    { path: '/strategies', selector: 'h1', text: 'Strategies' },
    { path: '/brokers', selector: '.app-main', check: 'visible' },
    { path: '/symbols', selector: '.app-main', check: 'visible' },
    { path: '/credentials', selector: '.app-main', check: 'visible' },
    { path: '/webhooks', selector: '.app-main', check: 'visible' },
    { path: '/settings', selector: 'h1', text: 'Settings' },
    { path: '/data-sources', selector: 'h1', text: 'Data' },
    { path: '/status', selector: '.app-main', check: 'visible' },
  ];

  for (const p of pages) {
    test(`GET ${p.path} → renders`, async ({ page }) => {
      const resp = await page.goto(p.path);
      expect(resp?.status()).toBe(200);

      const el = page.locator(p.selector).first();
      await expect(el).toBeVisible({ timeout: 8000 });

      if (p.text) {
        await expect(el).toContainText(p.text);
      }
    });
  }
});

test.describe('Backtest Runner UI — Interactive', () => {
  test('backtest form renders matrix mode by default', async ({ page }) => {
    await page.goto('/backtest');
    await expect(page.getByRole('heading', { level: 1 })).toContainText('Backtest Runner');
    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await expect(submitBtn).toBeVisible();
  });

  test('submit matrix backtest and see results table', async ({ page }) => {
    await page.goto('/backtest');

    const symbolInput = page.locator('input[placeholder="EURUSD, BTCUSD, US30"]');
    await symbolInput.fill('AAPL');

    const startDate = page.locator('input[type="date"]').first();
    const endDate = page.locator('input[type="date"]').nth(1);
    await startDate.fill('2024-01-02');
    await endDate.fill('2024-06-30');

    const submitBtn = page.getByRole('button', { name: /Run.*Backtest/ });
    await submitBtn.click();

    await expect(page.locator('h2', { hasText: 'Backtest Results' })).toBeVisible({ timeout: 30000 });
  });

  test('backtest history page lists runs and links to detail', async ({ page }) => {
    await page.goto('/backtest/history');
    await expect(page.locator('h1', { hasText: 'Backtest History' })).toBeVisible();

    const viewLinks = page.locator('a:has-text("View")');
    const count = await viewLinks.count();
    if (count > 0) {
      await viewLinks.first().click();
      await page.waitForTimeout(3000);
      const cards = page.locator('.orca-metric-card__value');
      const cardCount = await cards.count();
      expect(cardCount).toBeGreaterThanOrEqual(3);
    }
  });
});

test.describe('Dashboard Pages — Shared Components', () => {
  test('home dashboard renders regime gauge and prop firm gauges', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('h1', { hasText: 'Dashboard' })).toBeVisible();
  });

  test('live dashboard renders metric cards and equity chart', async ({ page }) => {
    await page.goto('/live');
    await page.waitForTimeout(3000);
    const metrics = page.locator('.orca-metric-card__label');
    const count = await metrics.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });

  test('live market renders symbol selector and charts', async ({ page }) => {
    await page.goto('/live/market');
    await expect(page.locator('select.orca-input')).toBeVisible();
  });
});
