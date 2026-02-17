import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('NewFeature');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Test Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'Test Vault' })).toBeVisible({ timeout: 10000 });
}

async function goToContacts(page: import('@playwright/test').Page) {
  await page.getByText('View all contacts').click();
  await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });
}

async function createContact(page: import('@playwright/test').Page, firstName: string, lastName: string) {
  await page.getByRole('button', { name: /add contact/i }).click();
  await page.getByPlaceholder('First name').fill(firstName);
  await page.getByPlaceholder('Last name').fill(lastName);
  await page.getByRole('button', { name: /create contact/i }).click();
  await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
  await expect(page.getByText(`${firstName} ${lastName}`).first()).toBeVisible({ timeout: 10000 });
}

test.describe('New Features', () => {
  test('Contact Detail - Edit Contact', async ({ page }) => {
    await setupVault(page, 'edit');
    await goToContacts(page);
    await createContact(page, 'EditTest', 'User');

    await page.getByRole('button', { name: 'Edit' }).first().click();

    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await expect(modal.getByText('Edit Contact')).toBeVisible();

    const firstNameInput = modal.locator('#first_name');
    await expect(firstNameInput).toBeVisible({ timeout: 5000 });

    await firstNameInput.clear();
    await firstNameInput.fill('UpdatedName');

    await modal.getByRole('button', { name: 'Save' }).click();

    await expect(modal).not.toBeVisible({ timeout: 15000 });
    await expect(page.getByText('UpdatedName User').first()).toBeVisible({ timeout: 10000 });
  });

  test('Contact Detail - Move button visible', async ({ page }) => {
    await setupVault(page, 'move');
    await goToContacts(page);
    await createContact(page, 'MoveTest', 'User');

    await expect(page.getByRole('button', { name: 'Move' })).toBeVisible({ timeout: 5000 });
  });

  test('Contact Detail - Feed Tab', async ({ page }) => {
    await setupVault(page, 'feed');
    await goToContacts(page);
    await createContact(page, 'FeedTest', 'User');

    const feedTab = page.getByRole('tab', { name: /feed/i });
    await feedTab.click();

    await expect(
      page.locator('.ant-card').filter({ hasText: /Activity Feed/i })
    ).toBeVisible({ timeout: 10000 });
  });

  test('Contact List - vCard buttons visible', async ({ page }) => {
    await setupVault(page, 'vcard');
    await goToContacts(page);

    await expect(page.getByRole('button', { name: 'Import vCard' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Export All' })).toBeVisible({ timeout: 10000 });
  });

  test('Settings - Delete Account visible', async ({ page }) => {
    await setupVault(page, 'del');

    await page.goto('/settings');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Danger Zone')).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Delete Account' })).toBeVisible({ timeout: 5000 });
  });

  test('Vault Reports - renders with data sections', async ({ page }) => {
    await setupVault(page, 'reports');

    await page.getByText('Reports').click();
    await expect(page).toHaveURL(/\/reports/, { timeout: 10000 });

    await expect(page.getByRole('heading', { name: 'Reports' })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-statistic').first()).toBeVisible({ timeout: 10000 });

    await expect(page.getByText('Address Distribution')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Important Dates Overview')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Mood Trends')).toBeVisible({ timeout: 5000 });
  });

  test('Journal Detail - Metrics and Slices sections visible', async ({ page }) => {
    await setupVault(page, 'journal');

    await page.getByText('Journal').click();
    await expect(page).toHaveURL(/\/journals/, { timeout: 10000 });

    await page.getByRole('button', { name: 'New Journal' }).click();
    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.locator('#name').fill('Test Journal');
    await modal.getByRole('button', { name: 'OK' }).click();

    await expect(modal).not.toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    await page.getByText('Test Journal').click();
    await expect(page).toHaveURL(/\/journals\/\d+$/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    await expect(page.getByRole('heading', { name: 'Test Journal' })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Metrics' })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Slices of Life', { exact: true }).first()).toBeVisible({ timeout: 5000 });
  });
});
