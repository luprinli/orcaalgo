const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:5174';
const API_BASE = 'http://localhost:8080';
const ADMIN_USER = process.env.ORCA_ADMIN_USER || 'admin';
const ADMIN_PASS = process.env.ORCA_ADMIN_PASSWORD || 'test-admin-2026';

const MAIN_ROUTES = [
  '/', '/execution',
  '/backtest', '/backtest/history', '/strategies',
  '/accounts', '/propfirm', '/charting',
  '/calibrate', '/attribution', '/simulate',
  '/admin', '/admin/universe', '/admin/propfirm', '/admin/symbols',
  '/integrations', '/2fa', '/settings', '/emergency',
];
const REDIRECTS = [
  '/live', '/live/market', '/risk', '/status',
  '/market-data', '/indicators', '/data-sources', '/brokers', '/symbols',
  '/credentials', '/webhooks', '/llm', '/notifications', '/optimize',
  '/admin/health', '/admin/logs',
  '/audit', '/users',
];

async function getToken() {
  try {
    const r = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: ADMIN_USER, password: ADMIN_PASS }),
    });
    const b = await r.json();
    return b.access_token || null;
  } catch { return null; }
}

test.describe('Full Route Coverage', () => {
  let token;

  test.beforeAll(async () => { token = await getToken(); });

  test.beforeEach(async ({ page }) => {
    if (!token) { test.skip(true, 'No auth'); return; }
    await page.goto(BASE, { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.evaluate((t) => {
      localStorage.setItem('orca_auth', JSON.stringify({ token: t, username: 'admin', expires_at: Date.now() + 86400000 }));
    }, token);
    await page.reload({ waitUntil: 'networkidle', timeout: 10000 });
  });

  for (const route of MAIN_ROUTES) {
    test(route, async ({ page }) => {
      const errs = [];
      page.on('console', m => {
        if (m.type() === 'error' && !m.text().includes('WebSocket') && !m.text().includes('ws://')) errs.push(m.text().slice(0, 99));
      });
      await page.goto(route, { waitUntil: 'domcontentloaded', timeout: 12000 });
      await page.waitForTimeout(300);
      const txt = await page.locator('body').textContent().catch(() => '');
      expect(txt?.length || 0, route + ' body empty').toBeGreaterThan(50);
      if (errs.length) console.log(`  ${route} — ${errs.length} non-WS errors: ${errs.join(' | ')}`);
    });
  }

  for (const route of REDIRECTS) {
    test(`${route} (redirect)`, async ({ page }) => {
      await page.goto(route, { waitUntil: 'domcontentloaded', timeout: 12000 });
      await page.waitForTimeout(300);
      const url = page.url();
      expect(url, `${route} should redirect`).not.toMatch(new RegExp(route.replace('/', '\\/') + '$'));
    });
  }
});
