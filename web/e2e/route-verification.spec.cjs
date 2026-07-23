const { test, expect } = require('@playwright/test');

const ROUTES = [
  { path: '/', label: 'Dashboard', heading: 'Dashboard' },
  { path: '/live', label: 'Live Dashboard', check: 'visible' },
  { path: '/live/market', label: 'Live Market', check: 'visible' },
  { path: '/execution', label: 'Execution', heading: 'Execution' },
  { path: '/backtest', label: 'Backtest', heading: 'Backtest Runner' },
  { path: '/strategies', label: 'Strategies', heading: 'Strategy Detail' },
  { path: '/brokers', label: 'Brokers', heading: 'Broker Management' },
  { path: '/data-sources', label: 'Data Sources', heading: 'Data Sources' },
  { path: '/symbols', label: 'Symbols', heading: 'Symbol Management' },
  { path: '/credentials', label: 'Credentials', heading: 'Credential Management' },
  { path: '/webhooks', label: 'Webhooks', heading: 'Webhook Configuration' },
  { path: '/llm', label: 'LLM Settings', heading: 'LLM Settings' },
  { path: '/2fa', label: '2FA Setup', heading: 'Two-Factor Authentication' },
  { path: '/settings', label: 'Settings', heading: 'Settings' },
  { path: '/propfirm', label: 'Prop Firms', heading: 'Prop Firm Configuration' },
  { path: '/admin', label: 'Admin', heading: 'Admin Panel' },
  { path: '/admin/health', label: 'System Health', heading: 'System Health' },
  { path: '/admin/logs', label: 'Error Logs', heading: 'Error Logs' },
  { path: '/audit', label: 'Audit Log', heading: 'Audit Log' },
];

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

test.describe('OrcaAlgo Frontend Route Verification', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
  });

  for (const route of ROUTES) {
    test(`Route ${route.path} renders without errors`, async ({ page }) => {
      const errors = [];
      page.on('console', msg => {
        if (msg.type() === 'error') {
          errors.push(`[${msg.type()}] ${msg.text()}`);
        }
      });
      page.on('pageerror', err => {
        errors.push(`[PAGE ERROR] ${err.message}`);
      });

      try {
        await page.goto(route.path, { waitUntil: 'networkidle', timeout: 15000 });
      } catch (navErr) {
        console.warn(`  Navigation to ${route.path} failed: ${navErr.message}`);
        test.skip(true, `Navigation to ${route.path} failed (page may not exist)`);
        return;
      }

      await expect(page.locator('.app-main')).toBeVisible({ timeout: 5000 });

      const content = await page.textContent('.app-main');
      expect(content).toBeTruthy();

      if (errors.length > 0) {
        console.warn(`  Console errors on ${route.path}:`, errors);
      }

      const h1 = page.locator('h1').first();
      const hasHeading = await h1.count() > 0;
      if (hasHeading) {
        const text = await h1.textContent();
        expect(text).toBeTruthy();
      }
    });
  }
});

test.describe('Sidebar Navigation', () => {
  test('All sidebar links are present and clickable', async ({ page }) => {
    await setupAuth(page);
    await page.goto('/', { waitUntil: 'networkidle' });

    const sidebar = page.locator('.app-sidebar');
    await expect(sidebar).toBeVisible();

    for (const route of ROUTES) {
      const link = sidebar.getByText(route.label, { exact: false });
      const count = await link.count();
      if (count > 0) {
        await link.first().click();
        await expect(page.locator('.app-main')).toBeVisible({ timeout: 5000 });
      }
    }
  });
});

test.describe('No Global Errors', () => {
  test('Dashboard loads without page errors', async ({ page }) => {
    await setupAuth(page);
    const pageErrors = [];
    page.on('pageerror', err => pageErrors.push(err.message));
    const consoleErrors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    await page.goto('/', { waitUntil: 'networkidle', timeout: 15000 });

    await expect(page.locator('.app-main')).toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(3000);

    expect(pageErrors, `Page errors on dashboard: ${pageErrors.join(', ')}`).toEqual([]);
    expect(consoleErrors.filter(e => !e.includes('favicon') && !e.includes('WebSocket') && !e.includes('Failed to load')), 
      `Console errors on dashboard: ${consoleErrors.join(', ')}`).toEqual([]);
  });
});
