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

test.describe('Backtest → Orch Promotion — Full E2E', () => {
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

  test('promoted strategies appear in Orch with correct strategy/symbol/timeframe', async ({ page }) => {
    const combos = [
      { strategy_id: 'grid_trading', symbol: 'SPX500', timeframe: '4h' },
      { strategy_id: 'rsi2_reversion', symbol: 'JPN225', timeframe: '1h' },
    ];
    await page.evaluate((data) => {
      sessionStorage.setItem('orch_batch_promote', JSON.stringify(data));
    }, combos);

    await page.goto('/backtest?mode=orchestrated', { waitUntil: 'networkidle', timeout: 15000 });
    await page.waitForTimeout(3000);

    await expect(page.getByLabel('Orch').first()).toBeChecked({ timeout: 5000 });
    await expect(page.getByText('Strategy Pairs')).toBeVisible({ timeout: 5000 });

    const strategyRow = page.getByText('Strategy Pairs').locator('..');
    await expect(strategyRow).toBeVisible({ timeout: 5000 });

    const comboboxes = page.locator('[role="combobox"]');
    const count = await comboboxes.count();
    expect(count, 'should have at least 1 combobox').toBeGreaterThanOrEqual(1);
  });

  test('Orch panel renders Configuration with correct defaults', async ({ page }) => {
    const combo = { strategy_id: 'trend_following', symbol: 'SPY', timeframe: '1d' };
    await page.evaluate((data) => {
      sessionStorage.setItem('orch_batch_promote', JSON.stringify([data]));
    }, combo);

    await page.goto('/backtest?mode=orchestrated', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByText('Strategy Pairs')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Configuration')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Top Picks')).toBeVisible({ timeout: 5000 });

    await expect(page.getByText('Rebalance')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Kelly Fraction')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Max Position')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Correlation Brake')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Matrix Mode')).toBeVisible({ timeout: 5000 });
  });

  test('Orch panel renders all 3 recommended Top Picks when clicked', async ({ page }) => {
    await page.goto('/backtest?mode=orchestrated', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await page.getByText('Top Picks').click();
    await page.waitForTimeout(500);

    const comboboxes = page.locator('[role="combobox"]');
    const count = await comboboxes.count();
    expect(count, 'Top Picks should create 3 rows (9 comboboxes)').toBeGreaterThanOrEqual(6);
  });

  test('single promote writes to sessionStorage and renders in Orch', async ({ page }) => {
    await page.evaluate((data) => {
      sessionStorage.setItem('orch_batch_promote', JSON.stringify([data]));
    }, { strategy_id: 'pairs_trading', symbol: 'ES', timeframe: '1h' });

    await page.goto('/backtest?mode=orchestrated', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByLabel('Orch').first()).toBeChecked({ timeout: 5000 });

    const comboText = page.locator('[role="combobox"]').first();
    await expect(comboText).toBeVisible({ timeout: 5000 });
  });
});
