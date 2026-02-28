import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('QFact');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('QFact Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For quick facts testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: 'QFact Vault' })).toBeVisible({ timeout: 10000 });
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

test.describe('Contact Modules - Quick Facts', () => {
  test('should show Quick Facts on contact page and support CRUD', async ({ page }) => {
    await setupVault(page, 'qfact');
    await goToContacts(page);
    await createContact(page, 'QFactTest', 'User');

    // Navigate to "Contact information" tab (name from seed data)
    await navigateToTab(page, 'Contact information');

    // Quick Facts card should be visible (this verifies the seed data fix)
    const qfCard = page.locator('.ant-card').filter({ hasText: 'Quick Facts' });
    await expect(qfCard).toBeVisible({ timeout: 10000 });

    // --- CREATE: Add a quick fact ---
    await qfCard.getByRole('button', { name: /add/i }).click();

    const input = qfCard.getByPlaceholder(/content/i);
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill('Loves hiking');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/quickFacts') && resp.request().method() === 'POST'
    );
    await qfCard.getByRole('button', { name: /save/i }).click();
    const resp = await createResp;
    expect(resp.status()).toBeLessThan(400);

    await expect(qfCard.getByText('Loves hiking')).toBeVisible({ timeout: 10000 });

    // --- UPDATE: Edit the quick fact ---
    await qfCard.getByRole('button', { name: /edit/i }).first().click();

    const editInput = qfCard.getByPlaceholder(/content/i);
    await expect(editInput).toBeVisible({ timeout: 5000 });
    await editInput.clear();
    await editInput.fill('Loves mountain hiking');

    const updateResp = page.waitForResponse(
      (resp) => resp.url().includes('/quickFacts') && resp.request().method() === 'PUT'
    );
    await qfCard.getByRole('button', { name: /update/i }).click();
    const uResp = await updateResp;
    expect(uResp.status()).toBeLessThan(400);

    await expect(qfCard.getByText('Loves mountain hiking')).toBeVisible({ timeout: 10000 });

    // --- DELETE: Remove the quick fact ---
    await qfCard.getByRole('button', { name: /delete/i }).first().click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/quickFacts') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(qfCard.getByText('Loves mountain hiking')).not.toBeVisible({ timeout: 10000 });
  });
});
