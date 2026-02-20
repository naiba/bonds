import { test, expect } from '@playwright/test';

const PASSWORD = 'password123';

async function registerUser(page: import('@playwright/test').Page, email: string, firstName = 'Test', lastName = 'User') {
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill(firstName);
  await page.getByPlaceholder('Last name').fill(lastName);
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill(PASSWORD);
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

async function loginUser(page: import('@playwright/test').Page, email: string) {
  await page.goto('/login');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder('Password').fill(PASSWORD);
  await page.getByRole('button', { name: 'Sign in', exact: true }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

test.describe('Login Page Improvements', () => {
  test('login page shows version in footer', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('a[href*="github.com/naiba/bonds"]')).toBeVisible({ timeout: 10000 });
  });

  test('login page shows register link by default', async ({ page }) => {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Create one')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Admin Features', () => {
  // Serial mode: tests share DB state, first registered user is admin
  test.describe.configure({ mode: 'serial' });

  let adminEmail: string;

  test('first registered user is admin and can access User Management', async ({ page }) => {
    adminEmail = `admin-${Date.now()}@example.com`;
    await registerUser(page, adminEmail, 'Admin', 'Boss');

    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('User Management')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
  });

  test('admin menu is visible in user dropdown', async ({ page }) => {
    await loginUser(page, adminEmail);

    // Open user dropdown by clicking the avatar
    await page.locator('.ant-avatar').click();
    await expect(page.getByText('Administration')).toBeVisible({ timeout: 5000 });
  });

  test('admin users page shows current user email', async ({ page }) => {
    await loginUser(page, adminEmail);

    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('table').getByText(adminEmail)).toBeVisible({ timeout: 10000 });
  });

  test('non-admin should not see admin menu', async ({ page }) => {
    // Register a second user (not admin)
    const userEmail = `nonadmin-${Date.now()}@example.com`;
    await registerUser(page, userEmail);

    // Open user dropdown
    await page.locator('.ant-avatar').click();
    // "Administration" should NOT appear for non-admin
    await expect(page.getByText('Administration')).not.toBeVisible({ timeout: 3000 });
  });

  test('admin settings page loads with form fields', async ({ page }) => {
    await loginUser(page, adminEmail);

    await page.goto('/admin/settings');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('System Settings')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('GitHub OAuth Key')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('OIDC Client ID')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Password Authentication')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('User Registration')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: 'Save Settings' })).toBeVisible({ timeout: 5000 });
  });
});

test.describe('2FA QR Code', () => {
  test('2FA setup shows QR code after clicking enable', async ({ page }) => {
    const email = `twofa-${Date.now()}@example.com`;
    await registerUser(page, email);

    await page.goto('/settings/2fa');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: /two-factor/i })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /enable/i }).click();

    // antd QRCode renders as canvas inside .ant-qrcode
    await expect(page.locator('.ant-qrcode').first()).toBeVisible({ timeout: 10000 });
  });
});
