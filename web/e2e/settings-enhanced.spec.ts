import { test, expect } from '@playwright/test';

async function registerUser(page: import('@playwright/test').Page) {
  const email = `se-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Settings');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

test.describe('Enhanced Settings', () => {
  test('should navigate to WebAuthn settings page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/webauthn');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card')).toBeVisible({ timeout: 10000 });
  });

  test('should show register passkey button on WebAuthn page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/webauthn');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button').filter({ has: page.locator('.anticon-plus') })).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to OAuth providers page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/oauth');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card')).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to storage info page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/storage');
    await expect(
      page.getByRole('heading', { level: 4 }).first()
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card').first()).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to users page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/users');
    await expect(
      page.getByRole('heading', { level: 4 }).getByText('Users')
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card')).toBeVisible({ timeout: 10000 });
  });

  test('should show current user in users list', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table').getByText('Settings Tester')).toBeVisible({ timeout: 10000 });
  });

  test('should navigate to notifications page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/notifications');
    await expect(
      page.getByRole('heading', { level: 4 }).getByText('Notifications')
    ).toBeVisible({ timeout: 10000 });
  });

  test('should show add channel button on notifications page', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/notifications');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('button').filter({ has: page.locator('.anticon-plus') })).toBeVisible({ timeout: 10000 });
  });

  test('should show notification channel with verification status', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/notifications');
    await page.waitForLoadState('networkidle');

    const seedChannel = page.locator('.ant-list-item').first();
    await expect(seedChannel).toBeVisible({ timeout: 10000 });
    await expect(
      seedChannel.getByText(/Verified|Unverified/)
    ).toBeVisible({ timeout: 5000 });
  });

  test('should navigate to personalize page and show currencies section', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { level: 4 }).first()).toBeVisible({ timeout: 10000 });

    const currenciesHeader = page.getByText('Currencies').first();
    await expect(currenciesHeader).toBeVisible({ timeout: 10000 });
    await currenciesHeader.click();
    await page.waitForLoadState('networkidle');

    await expect(page.getByPlaceholder(/search currencies/i)).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: /enable all/i })).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: /disable all/i })).toBeVisible({ timeout: 5000 });
  });

  test('should search and filter currencies', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    const currenciesHeader = page.getByText('Currencies').first();
    await currenciesHeader.click();
    await page.waitForLoadState('networkidle');

    const searchInput = page.getByPlaceholder(/search currencies/i);
    await expect(searchInput).toBeVisible({ timeout: 10000 });

    await searchInput.fill('USD');
    await expect(page.getByText('USD')).toBeVisible({ timeout: 5000 });
  });

  test('should show currency toggle switches', async ({ page }) => {
    await registerUser(page);
    await page.goto('/settings/personalize');
    await page.waitForLoadState('networkidle');

    const currenciesHeader = page.getByText('Currencies').first();
    await currenciesHeader.click();
    await page.waitForLoadState('networkidle');

    await expect(page.locator('.ant-switch').first()).toBeVisible({ timeout: 10000 });
  });
});
