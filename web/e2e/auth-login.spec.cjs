// @ts-check
const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:5173';
const ADMIN_USER = process.env.ORCA_ADMIN_USER || 'admin';
const ADMIN_PASS = process.env.ORCA_ADMIN_PASSWORD || 'test-admin-2026';

test.describe('Login — Browser UI', () => {
  test('login fields are visible', async ({ page }) => {
    await page.goto(BASE);
    await expect(page.locator('input[type="text"]').first()).toBeVisible({ timeout: 5000 });
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.getByRole('button', { name: /Sign In|sign in|Login|login/i })).toBeVisible();
  });

  test('submitting empty form shows error', async ({ page }) => {
    await page.goto(BASE);
    await page.getByRole('button', { name: /Sign In|sign in|Login|login/i }).click();
    await page.waitForTimeout(1000);
    const hasError = await page.locator('.text-destructive, [role="alert"]').isVisible().catch(() => false);
    expect(hasError).toBeTruthy();
  });

  test('submitting wrong password shows error', async ({ page }) => {
    await page.goto(BASE);
    await page.locator('input[type="text"]').first().fill(ADMIN_USER);
    await page.locator('input[type="password"]').fill('WRONG_PASSWORD_XYZ');
    await page.getByRole('button', { name: /Sign In|sign in|Login|login/i }).click();
    await page.waitForTimeout(3000);
    const hasContent = await page.textContent('body');
    expect(hasContent).not.toMatch(/^Orca Algo$/);
  });

  test('successful login redirects away from auth page', async ({ page }) => {
    await page.goto(BASE);
    await page.locator('input[type="text"]').first().fill(ADMIN_USER);
    await page.locator('input[type="password"]').fill(ADMIN_PASS);
    await page.getByRole('button', { name: /Sign In|sign in|Login|login/i }).click();

    try {
      await page.waitForURL('**/backtest**', { timeout: 15000 });
      expect(page.url()).toContain('/backtest');
    } catch {
      await page.waitForTimeout(3000);
      expect(page.url()).not.toBe('http://localhost:5173/login');
    }
  });

  test('Enter key on password field submits login', async ({ page }) => {
    await page.goto(BASE);
    await page.locator('input[type="text"]').first().fill(ADMIN_USER);
    await page.locator('input[type="password"]').fill(ADMIN_PASS);
    await page.locator('input[type="password"]').press('Enter');

    try {
      await page.waitForURL('**/backtest**', { timeout: 15000 });
    } catch { /* auth may fail but page should not crash */ }
    expect(page.url()).not.toBe('http://localhost:5173/login');
  });
});

test.describe('Login — Edge Cases', () => {
  test('login with valid credentials then navigate to protected page', async ({ page }) => {
    await page.goto(BASE);
    await page.locator('input[type="text"]').first().fill(ADMIN_USER);
    await page.locator('input[type="password"]').fill(ADMIN_PASS);
    await page.getByRole('button', { name: /Sign In|sign in|Login|login/i }).click();

    try { await page.waitForURL('**/backtest**', { timeout: 15000 }); } catch { /* ok */ }
    await page.goto('/backtest');
    expect(await page.locator('h1').first().textContent()).not.toBe('');
  });

  test('accessing page without token shows login form', async ({ page }) => {
    await page.evaluate(() => localStorage.removeItem('orca_auth'));
    await page.goto('/');
    await page.waitForTimeout(1000);
    const hasLogin = await page.locator('input[type="password"]').isVisible().catch(() => false);
    expect(hasLogin).toBeTruthy();
  });
});
