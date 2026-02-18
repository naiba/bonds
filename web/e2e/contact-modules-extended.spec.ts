import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Module');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Module Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For module testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'Module Vault' })).toBeVisible({ timeout: 10000 });
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

async function navigateToTab(page: import('@playwright/test').Page, tabName: string, exact = false) {
  const tab = page.getByRole('tab', { name: tabName, exact });
  await tab.click();
  await page.waitForLoadState('networkidle');
}

test.describe('Contact Modules - Relationships', () => {
  test('should create a relationship between two contacts', async ({ page }) => {
    await setupVault(page, 'rel-create');
    await goToContacts(page);

    await createContact(page, 'RelAlice', 'Smith');

    await page.getByRole('button', { name: /back/i }).first().click();
    await expect(page).toHaveURL(/\/contacts$/, { timeout: 5000 });

    await createContact(page, 'RelBob', 'Jones');

    await page.getByRole('button', { name: /back/i }).first().click();
    await expect(page).toHaveURL(/\/contacts$/, { timeout: 5000 });

    await page.getByText('RelAlice Smith').click();
    await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });

    await navigateToTab(page, 'Social');

    const relCard = page.locator('.ant-card').filter({ hasText: /Relationships/ });
    await expect(relCard).toBeVisible({ timeout: 10000 });

    await relCard.locator('.ant-card-extra button').click();

    const modal = page.locator('.ant-modal').filter({ hasText: /relationship/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    const selects = modal.locator('.ant-select');
    await selects.first().click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'RelBob' }).click();

    // Wait for first dropdown to fully close before opening second
    await expect(page.locator('.ant-select-dropdown:visible')).not.toBeVisible({ timeout: 5000 }).catch(() => {});
    await selects.nth(1).click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/relationships') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(relCard.getByText('RelBob Jones')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Operations', () => {
  test('should toggle favorite', async ({ page }) => {
    await setupVault(page, 'fav-toggle');
    await goToContacts(page);
    await createContact(page, 'FavTest', 'User');

    const favButton = page.getByRole('button', { name: /Favorite/i }).first();
    await expect(favButton).toBeVisible({ timeout: 10000 });

    const favResp = page.waitForResponse(
      (resp) => resp.url().includes('/favorite') && resp.request().method() === 'PUT'
    );
    await favButton.click();
    const resp = await favResp;
    expect(resp.status()).toBeLessThan(400);

    await expect(page.getByRole('button', { name: /Unfavorite/i })).toBeVisible({ timeout: 10000 });
  });

  test('should toggle archive', async ({ page }) => {
    await setupVault(page, 'archive-toggle');
    await goToContacts(page);
    await createContact(page, 'ArchiveTest', 'User');

    const archiveButton = page.getByRole('button', { name: /Archive/i }).first();
    await expect(archiveButton).toBeVisible({ timeout: 10000 });

    const archiveResp = page.waitForResponse(
      (resp) => resp.url().includes('/archive') && resp.request().method() === 'PUT'
    );
    await archiveButton.click();
    const resp = await archiveResp;
    expect(resp.status()).toBeLessThan(400);

    await expect(page.getByRole('button', { name: /Unarchive/i })).toBeVisible({ timeout: 10000 });
  });

  test('should delete a contact', async ({ page }) => {
    await setupVault(page, 'contact-delete');
    await goToContacts(page);
    await createContact(page, 'DeleteMe', 'User');

    await page.locator('button').filter({ has: page.locator('.anticon-more') }).click();

    await page.locator('.ant-dropdown-menu-item-danger').filter({ hasText: /delete/i }).click();

    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/contacts/') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-modal-confirm').getByRole('button', { name: /delete/i }).click();
    const resp = await deleteResp;
    expect(resp.status()).toBeLessThan(400);

    await expect(page).toHaveURL(/\/contacts$/, { timeout: 10000 });
    await expect(page.getByText('DeleteMe User')).not.toBeVisible({ timeout: 5000 });
  });
});
