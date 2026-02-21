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

  test('admin pages have tab navigation between Users and Settings', async ({ page }) => {
    await loginUser(page, adminEmail);

    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('User Management')).toBeVisible({ timeout: 10000 });

    const segmented = page.locator('.ant-segmented');
    await expect(segmented).toBeVisible({ timeout: 5000 });

    await segmented.getByText('Settings').click();
    await expect(page).toHaveURL(/\/admin\/settings/, { timeout: 10000 });
    await expect(page.getByText('System Settings')).toBeVisible({ timeout: 10000 });

    await page.locator('.ant-segmented').getByText('Users').click();
    await expect(page).toHaveURL(/\/admin\/users/, { timeout: 10000 });
    await expect(page.getByText('User Management')).toBeVisible({ timeout: 10000 });
  });

  test('admin settings page loads with form fields', async ({ page }) => {
    await loginUser(page, adminEmail);

    await page.goto('/admin/settings');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('System Settings')).toBeVisible({ timeout: 10000 });

    // "Application" and "Authentication" panels are expanded by default
    await expect(page.getByText('Password Authentication')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('User Registration')).toBeVisible({ timeout: 5000 });

    // Verify collapsed panel headers are visible
    await expect(page.getByText('OAuth / OIDC')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('SMTP Email')).toBeVisible({ timeout: 5000 });

    // Expand OAuth panel and verify fields inside
    await page.getByText('OAuth / OIDC').click();
    await expect(page.getByText('GitHub OAuth Key')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('OIDC Client ID')).toBeVisible({ timeout: 5000 });

    await expect(page.getByRole('button', { name: 'Save Settings' })).toBeVisible({ timeout: 5000 });
  });

  let secondUserEmail: string;

  test('admin can disable a user', async ({ page }) => {
    secondUserEmail = `user2-${Date.now()}@example.com`;
    await registerUser(page, secondUserEmail, 'Second', 'User');

    await loginUser(page, adminEmail);
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: secondUserEmail });
    await expect(row).toBeVisible({ timeout: 5000 });
    await expect(row.locator('.ant-tag').filter({ hasText: 'Active' })).toBeVisible({ timeout: 5000 });
    await row.getByRole('button', { name: 'Disable' }).click();
    await page.waitForLoadState('networkidle');

    await expect(row.locator('.ant-tag').filter({ hasText: 'Disabled' })).toBeVisible({ timeout: 10000 });
  });

  test('admin can re-enable a disabled user', async ({ page }) => {
    await loginUser(page, adminEmail);
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: secondUserEmail });
    await expect(row.locator('.ant-tag').filter({ hasText: 'Disabled' })).toBeVisible({ timeout: 5000 });
    await row.getByRole('button', { name: 'Enable' }).click();
    await page.waitForLoadState('networkidle');

    await expect(row.locator('.ant-tag').filter({ hasText: 'Active' })).toBeVisible({ timeout: 10000 });
  });

  test('admin can promote a user to admin', async ({ page }) => {
    await loginUser(page, adminEmail);
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: secondUserEmail });
    await expect(row.locator('.ant-tag').filter({ hasText: 'User' })).toBeVisible({ timeout: 5000 });
    await row.getByRole('button', { name: 'Set Admin' }).click();
    await page.waitForLoadState('networkidle');

    await expect(row.locator('.ant-tag').filter({ hasText: 'Admin' })).toBeVisible({ timeout: 10000 });
  });

  test('admin can remove admin role from a user', async ({ page }) => {
    await loginUser(page, adminEmail);
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: secondUserEmail });
    await expect(row.locator('.ant-tag').filter({ hasText: 'Admin' })).toBeVisible({ timeout: 5000 });
    await row.getByRole('button', { name: 'Remove Admin' }).click();
    await page.waitForLoadState('networkidle');

    await expect(row.locator('.ant-tag').filter({ hasText: 'User' })).toBeVisible({ timeout: 10000 });
  });

  test('admin cannot see action buttons for own row', async ({ page }) => {
    await loginUser(page, adminEmail);
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });

    const selfRow = page.getByRole('row').filter({ hasText: adminEmail });
    await expect(selfRow).toBeVisible({ timeout: 5000 });
    await expect(selfRow.getByRole('button', { name: 'Disable' })).not.toBeVisible();
    await expect(selfRow.getByRole('button', { name: 'Set Admin' })).not.toBeVisible();
  });

  test('admin can delete a user', async ({ page }) => {
    const throwawayEmail = `throwaway-${Date.now()}@example.com`;
    await registerUser(page, throwawayEmail, 'Throw', 'Away');

    await loginUser(page, adminEmail);
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('table')).toBeVisible({ timeout: 10000 });

    const row = page.getByRole('row').filter({ hasText: throwawayEmail });
    await expect(row).toBeVisible({ timeout: 5000 });

    await row.locator('button.ant-btn-dangerous').click();

    const popconfirm = page.locator('.ant-popconfirm');
    await expect(popconfirm).toBeVisible({ timeout: 5000 });
    await popconfirm.getByRole('button', { name: 'OK' }).click();
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('table').getByText(throwawayEmail)).not.toBeVisible({ timeout: 10000 });
  });

  test('admin can save and persist settings', async ({ page }) => {
    await loginUser(page, adminEmail);
    await page.goto('/admin/settings');
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('System Settings')).toBeVisible({ timeout: 10000 });

    const appNameInput = page.locator('#app\\.name');
    await expect(appNameInput).toBeVisible({ timeout: 5000 });

    await appNameInput.clear();
    await appNameInput.fill('Test Bonds App');

    await page.getByRole('button', { name: 'Save Settings' }).click();
    await page.waitForLoadState('networkidle');

    await page.reload();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('System Settings')).toBeVisible({ timeout: 10000 });

    const reloadedInput = page.locator('#app\\.name');
    await expect(reloadedInput).toHaveValue('Test Bonds App', { timeout: 10000 });
  });

  test('non-admin user cannot access admin users page', async ({ page }) => {
    await loginUser(page, secondUserEmail);

    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');

    await page.waitForTimeout(2000);
    await expect(page.getByRole('table').getByText(adminEmail)).not.toBeVisible({ timeout: 5000 });
  });

  test('non-admin user cannot access admin settings page', async ({ page }) => {
    await loginUser(page, secondUserEmail);

    await page.goto('/admin/settings');
    await page.waitForLoadState('networkidle');

    await page.waitForTimeout(2000);
    const appNameInput = page.locator('#app\\.name');
    const count = await appNameInput.count();
    if (count > 0) {
      await expect(appNameInput).toHaveValue('', { timeout: 5000 });
    }
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
