import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function setupVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Test');
  await page.getByPlaceholder('Last name').fill('User');
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

test.describe('Vault - Companies CRUD', () => {
  test('should create a company', async ({ page }) => {
    await setupVault(page, 'company-create');

    const vaultUrl = page.url();
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Companies/i }).click();
    await page.waitForLoadState('networkidle');


    // Button text is "Add Company"
    await page.getByRole('button', { name: /add company/i }).first().click();

    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Form field label: "Company Name" (vault.companies.name)
    await modal.getByLabel(/company name/i).fill('Acme Corp');
    // Form field label: "Type" (vault.companies.type)
    await modal.getByLabel(/type/i).fill('Technology');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(page.getByText('Acme Corp')).toBeVisible({ timeout: 10000 });
  });

  test('should edit a company', async ({ page }) => {
    await setupVault(page, 'company-edit');

    const vaultUrl = page.url();
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Companies/i }).click();
    await page.waitForLoadState('networkidle');


    // Create a company first
    await page.getByRole('button', { name: /add company/i }).first().click();
    const modal = page.locator('.ant-modal');
    await modal.getByLabel(/company name/i).fill('Old Name');
    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await expect(page.getByText('Old Name')).toBeVisible({ timeout: 10000 });

    // Click edit button in the row
    await page.getByRole('row').filter({ hasText: 'Old Name' }).locator('button').filter({ has: page.locator('.anticon-edit') }).click();

    const editModal = page.locator('.ant-modal');
    await expect(editModal).toBeVisible({ timeout: 5000 });
    const nameInput = editModal.getByLabel(/company name/i);
    await nameInput.clear();
    await nameInput.fill('New Name');

    const updateResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'PUT'
    );
    await editModal.getByRole('button', { name: /ok/i }).click();
    const resp = await updateResp;
    expect(resp.status()).toBeLessThan(400);

    await expect(page.getByText('New Name')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Old Name')).not.toBeVisible({ timeout: 5000 });
  });

  test('should delete a company', async ({ page }) => {
    await setupVault(page, 'company-delete');

    const vaultUrl = page.url();
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Companies/i }).click();

    // Create first
    await page.getByRole('button', { name: /add company/i }).first().click();
    const modal = page.locator('.ant-modal');
    await modal.getByLabel(/company name/i).fill('Delete Me Corp');
    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await expect(page.getByText('Delete Me Corp')).toBeVisible({ timeout: 10000 });

    // Click delete button in the row
    await page.getByRole('row').filter({ hasText: 'Delete Me Corp' }).locator('button').filter({ has: page.locator('.anticon-delete') }).click();

    // Modal.confirm uses ant-modal-confirm
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-modal-confirm').getByRole('button', { name: /delete/i }).click();
    const resp = await deleteResp;
    expect(resp.status()).toBeLessThan(400);

    await expect(page.getByText('Delete Me Corp')).not.toBeVisible({ timeout: 10000 });
  });
});
