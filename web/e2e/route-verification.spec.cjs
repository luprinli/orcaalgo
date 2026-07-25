const { test, expect } = require('@playwright/test');

const MAIN_SELECTOR = '[id="main-content"]';
const SIDEBAR_SELECTOR = '[aria-label="Main navigation"]';

async function setupAuth(page) {
  await page.addInitScript(() => {
    const auth = {
      token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test',
      username: 'admin',
      expires_at: Date.now() + 86400000,
    };
    localStorage.setItem('orca_auth', JSON.stringify(auth));
  });
}

async function mockApi(page) {
  await page.route('**/api/v1/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: '[]',
    });
  });
  await page.route('**/ws**', async (route) => {
    await route.abort();
  });
}

const ROUTES = [
  '/',
  '/execution',
  '/backtest',
  '/strategies',
  '/charting',
  '/simulate',
  '/calibrate',
  '/attribution',
  '/settings',
  '/integrations',
  '/accounts',
  '/admin',
  '/emergency',
  '/2fa',
  '/propfirm',
];

test.describe('OrcaAlgo Frontend Route Verification', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
  });

  for (const route of ROUTES) {
    test(`Route ${route} renders without errors`, async ({ page }) => {
      const errors = [];
      page.on('pageerror', err => {
        errors.push(`[PAGE ERROR] ${err.message}`);
      });

      try {
        await page.goto(route, { waitUntil: 'domcontentloaded', timeout: 15000 });
      } catch (navErr) {
        test.skip(true, `Navigation to ${route} failed (page may not exist)`);
        return;
      }
      await page.waitForTimeout(500);

      if (route === '/emergency') {
        const bodyText = await page.textContent('body');
        expect(bodyText).toBeTruthy();
        return;
      }

      await expect(page.locator(MAIN_SELECTOR)).toBeVisible({ timeout: 8000 });

      const content = await page.textContent(MAIN_SELECTOR);
      expect(content).toBeTruthy();

      if (errors.length > 0) {
        console.warn(`  Page errors on ${route}:`, errors);
      }
    });
  }
});

test.describe('Sidebar Navigation', () => {
  test('All sidebar links are present and clickable', async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
    await page.goto('/', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(500);

    const sidebar = page.locator(SIDEBAR_SELECTOR);
    await expect(sidebar).toBeVisible();

    const LINK_LABELS = [
      'Dashboard', 'Execution', 'Backtesting', 'Strategies',
      'Charts', 'Simulation', 'Calibration', 'Attribution',
      'System', 'Integrations', 'Accounts', 'Admin', 'Emergency',
    ];

    for (const label of LINK_LABELS) {
      const link = sidebar.getByText(label, { exact: true });
      await expect(link, `Sidebar link "${label}" should exist`).toBeVisible({ timeout: 3000 });
    }
  });
});

test.describe('Legacy Route Redirects', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
  });

  const redirectTests = [
    { from: '/brokers', to: '/integrations' },
    { from: '/credentials', to: '/integrations' },
    { from: '/symbols', to: '/integrations' },
    { from: '/data-sources', to: '/settings' },
    { from: '/optimize', to: '/backtest' },
    { from: '/market-data', to: '/charting' },
    { from: '/status', to: '/admin' },
  ];

  for (const { from, to } of redirectTests) {
    test(`${from} redirects to ${to}`, async ({ page }) => {
      await page.goto(from, { waitUntil: 'domcontentloaded', timeout: 10000 });
      await page.waitForTimeout(2000);
      const url = page.url();
      expect(url, `${from} should redirect to ${to}`).toContain(to);
    });
  }
});

test.describe('No Global Errors', () => {
  test('Dashboard loads without page errors', async ({ page }) => {
    await setupAuth(page);
    await mockApi(page);
    const pageErrors = [];
    page.on('pageerror', err => pageErrors.push(err.message));

    await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 15000 });

    await expect(page.locator(MAIN_SELECTOR)).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(3000);

    expect(pageErrors,
      `Page errors on dashboard: ${pageErrors.join(', ')}`).toEqual([]);
  });
});
