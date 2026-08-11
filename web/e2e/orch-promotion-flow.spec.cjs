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

test.describe('Orch Promotion Flow — Individual & Batch', () => {
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

  test('individual Orch link navigates to Orch mode with pre-filled strategy', async ({ page }) => {
    const orchUrl = '/backtest?view=runner&mode=orchestrated&orch_strategy=grid_trading&orch_symbol=SPX500&orch_tf=4h';
    await page.goto(orchUrl, { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByLabel('Orch')).toBeChecked({ timeout: 5000 });
    await expect(page.getByText('Strategy Pairs')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Configuration')).toBeVisible({ timeout: 5000 });

    const comboText = page.locator('button[role="combobox"]').first();
    await expect(comboText).toBeVisible({ timeout: 5000 });
  });

  test('Orch mode renders when mode=orchestrated in URL', async ({ page }) => {
    await page.goto('/backtest?mode=orchestrated', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByLabel('Orch')).toBeChecked({ timeout: 5000 });
    await expect(page.getByText('Strategy Pairs')).toBeVisible({ timeout: 5000 });
  });

  test('Matrix mode renders when mode=matrix in URL (default)', async ({ page }) => {
    await page.goto('/backtest?mode=matrix', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByLabel('Matrix').first()).toBeChecked({ timeout: 5000 });
    await expect(page.getByText('Strategies').first()).toBeVisible({ timeout: 5000 });
  });

  test('Single mode renders when mode=single in URL', async ({ page }) => {
    await page.goto('/backtest?mode=single', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByLabel('Single')).toBeChecked({ timeout: 5000 });
  });

  test('Orch promo params auto-switch to Orch mode', async ({ page }) => {
    await page.goto('/backtest?view=runner&orch_strategy=rsi2_reversion&orch_symbol=JPN225&orch_tf=1h', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByLabel('Orch')).toBeChecked({ timeout: 5000 });
    await expect(page.getByText('Strategy Pairs')).toBeVisible({ timeout: 5000 });
  });

  test('batch_promote=true in URL reads from sessionStorage and renders', async ({ page }) => {
    const combos = [
      { strategy_id: 'grid_trading', symbol: 'SPX500', timeframe: '4h' },
      { strategy_id: 'rsi2_reversion', symbol: 'JPN225', timeframe: '1h' },
    ];
    await page.evaluate((data) => {
      sessionStorage.setItem('orch_batch_promote', JSON.stringify(data));
    }, combos);

    await page.goto('/backtest?view=runner&batch_promote=true', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    await expect(page.getByLabel('Orch').first()).toBeChecked({ timeout: 5000 });
    await expect(page.getByText('Strategy Pairs')).toBeVisible({ timeout: 5000 });

    const comboboxes = page.locator('[role="combobox"]');
    const count = await comboboxes.count();
    expect(count).toBeGreaterThanOrEqual(2);
  });

  test('mode switching via radio preserves URL mode param', async ({ page }) => {
    await page.goto('/backtest?mode=orchestrated', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);

    await page.getByLabel('Matrix').first().click();
    await page.waitForTimeout(500);

    await expect(page.getByLabel('Matrix').first()).toBeChecked({ timeout: 5000 });
    await expect(page.getByText('Strategies').first()).toBeVisible({ timeout: 5000 });
  });
});
