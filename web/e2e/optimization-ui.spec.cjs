// @ts-check
const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:5173';
const API_BASE = process.env.API_BASE || BASE;
const ADMIN_USER = process.env.ORCA_ADMIN_USER || 'admin';
const ADMIN_PASS = process.env.ORCA_ADMIN_PASSWORD || 'test-admin-2026';

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

test.describe('BacktestRunner — Matrix, Single, Orch Modes', () => {
  let token;

  test.beforeAll(async () => { token = await getToken(); });

  test.beforeEach(async ({ page }) => {
    if (!token) { test.skip(true, 'No auth token'); return; }
    await page.goto(BASE, { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.evaluate((t) => {
      localStorage.setItem('orca_auth', JSON.stringify({ token: t, username: 'admin', expires_at: Date.now() + 86400000 }));
    }, token);
    await page.reload({ waitUntil: 'networkidle', timeout: 10000 });
  });

  test('mode radio buttons render (Matrix default)', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await expect(page.getByLabel('Matrix')).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel('Single')).toBeVisible({ timeout: 5000 });
    await expect(page.getByLabel('Orch')).toBeVisible({ timeout: 5000 });
  });

  test('Matrix mode shows Auto-Optimize checkbox and combo count', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await expect(page.getByLabel('Auto-Optimize')).toBeVisible({ timeout: 5000 });

    const runBtn = page.getByRole('button', { name: /Run Matrix/i });
    await expect(runBtn).toBeVisible({ timeout: 5000 });
  });

  test('Matrix mode shows timeframes multi-select', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await expect(page.getByText('Timeframes')).toBeVisible({ timeout: 5000 });
  });

  test('switching to Single mode preserves core inputs', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await page.getByLabel('Single').click();
    await page.waitForTimeout(500);

    await expect(page.getByText('Start')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Capital')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: /Run/i })).toBeVisible({ timeout: 5000 });
  });

  test('strategy section visible', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await expect(page.getByText('Strategies')).toBeVisible({ timeout: 5000 });
  });

  test('Orch mode renders orchestration panel', async ({ page }) => {
    await page.goto('/backtest', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);

    await page.getByLabel('Orch').click();
    await page.waitForTimeout(500);

    await expect(page.getByText('Strategy Pairs')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Configuration')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Top Picks')).toBeVisible({ timeout: 5000 });
  });
});
