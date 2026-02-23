import { test, expect } from '@playwright/test';

let counter = 0;

function uniqueEmail(prefix: string): string {
  return `${prefix}-${Date.now()}-${++counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function registerAndCreateVault(page: import('@playwright/test').Page, prefix: string) {
  const email = uniqueEmail(prefix);
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('VaultExt');
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

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

async function createJournalAndNavigate(page: import('@playwright/test').Page, vaultUrl: string, journalName: string) {
  await page.goto(vaultUrl + '/journals');
  await page.getByRole('button', { name: 'New Journal' }).click();
  const modal = page.locator('.ant-modal').filter({ hasText: /new journal/i });
  await expect(modal).toBeVisible({ timeout: 5000 });
  await modal.locator('#name').fill(journalName);
  await modal.getByRole('button', { name: 'OK' }).click();
  await expect(modal).not.toBeVisible({ timeout: 10000 });
  await page.waitForLoadState('networkidle');

  await page.getByText(journalName).click();
  await expect(page).toHaveURL(/\/journals\/\d+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByRole('heading', { name: journalName })).toBeVisible({ timeout: 10000 });
}

async function createPostInJournal(page: import('@playwright/test').Page, postTitle: string) {
  await page.getByRole('button', { name: 'New Post' }).click();
  const modal = page.locator('.ant-modal').filter({ hasText: /new post/i });
  await expect(modal).toBeVisible({ timeout: 5000 });
  await modal.getByRole('textbox', { name: /title/i }).fill(postTitle);
  const postResp = page.waitForResponse(
    (resp) => resp.url().includes('/posts') && resp.request().method() === 'POST' && resp.status() < 400
  );
  await modal.getByRole('button', { name: 'OK' }).click();
  await postResp;
  await expect(modal).not.toBeVisible({ timeout: 15000 });
  await page.waitForLoadState('networkidle');
  await expect(page.getByText(postTitle)).toBeVisible({ timeout: 10000 });
}

async function navigateToPostDetail(page: import('@playwright/test').Page, postTitle: string) {
  await page.getByText(postTitle).click();
  await expect(page).toHaveURL(/\/posts\/\d+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

test.describe('Vault Extended Features', () => {

  test('Vault Feed - renders feed page', async ({ page }) => {
    await registerAndCreateVault(page, 'vfeed');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/feed');
    await expect(page).toHaveURL(/\/feed$/, { timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 4 }).filter({ hasText: 'Activity Feed' })
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-list').first()).toBeVisible({ timeout: 10000 });
  });

  test('Vault Tasks - renders tasks page', async ({ page }) => {
    await registerAndCreateVault(page, 'vtasks');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/tasks');
    await expect(page).toHaveURL(/\/tasks$/, { timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 4 }).filter({ hasText: 'All Tasks' })
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-card').first()).toBeVisible({ timeout: 10000 });
  });

  test('Vault Calendar - renders calendar with month view', async ({ page }) => {
    await registerAndCreateVault(page, 'vcal');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/calendar');
    await expect(page).toHaveURL(/\/calendar$/, { timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 4 }).filter({ hasText: 'Calendar' })
    ).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-picker-calendar')).toBeVisible({ timeout: 10000 });
  });

  test('Journal Post - create post, verify and navigate to detail', async ({ page }) => {
    await registerAndCreateVault(page, 'jpost');
    const vaultUrl = getVaultUrl(page);

    await createJournalAndNavigate(page, vaultUrl, 'Post Test Journal');
    await createPostInJournal(page, 'My First Post');

    await navigateToPostDetail(page, 'My First Post');
    await expect(page.getByRole('heading', { name: 'My First Post' })).toBeVisible({ timeout: 10000 });
  });

  test('Journal Post Tags - add a tag to a post', async ({ page }) => {
    await registerAndCreateVault(page, 'jtag');
    const vaultUrl = getVaultUrl(page);

    await createJournalAndNavigate(page, vaultUrl, 'Tag Test Journal');
    await createPostInJournal(page, 'Tagged Post');
    await navigateToPostDetail(page, 'Tagged Post');

    await page.getByText('Add tag').click();
    const tagInput = page.getByPlaceholder('Tag name');
    await expect(tagInput).toBeVisible({ timeout: 5000 });
    await tagInput.fill('e2e-tag');

    const tagResponse = page.waitForResponse(
      (resp) => resp.url().includes('/tags') && resp.request().method() === 'POST'
    );
    await tagInput.press('Enter');
    await tagResponse;

    await expect(page.locator('.ant-tag').filter({ hasText: 'e2e-tag' })).toBeVisible({ timeout: 10000 });
  });

  test('Journal Slices of Life - create a slice', async ({ page }) => {
    await registerAndCreateVault(page, 'jslice');
    const vaultUrl = getVaultUrl(page);

    await createJournalAndNavigate(page, vaultUrl, 'Slice Test Journal');
    await expect(page.getByText('Slices of Life', { exact: true }).first()).toBeVisible({ timeout: 5000 });

    await page.getByRole('button', { name: 'New Slice' }).click();
    const sliceModal = page.locator('.ant-modal').filter({ hasText: /new slice/i });
    await expect(sliceModal).toBeVisible({ timeout: 5000 });
    await sliceModal.locator('#name').fill('Summer 2025');
    await sliceModal.locator('#description').fill('A great summer');

    const sliceResponse = page.waitForResponse(
      (resp) => resp.url().includes('/slices') && resp.request().method() === 'POST'
    );
    await sliceModal.getByRole('button', { name: 'OK' }).click();
    await sliceResponse;
    await expect(sliceModal).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText('Summer 2025')).toBeVisible({ timeout: 10000 });
  });

  test('Vault Settings Labels - create a label', async ({ page }) => {
    await registerAndCreateVault(page, 'vlabel');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Labels' }).click();
    await page.waitForLoadState('networkidle');

    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill('e2e-label');

    const labelResponse = page.waitForResponse(
      (resp) => resp.url().includes('/labels') && resp.request().method() === 'POST'
    );
    await page.getByRole('button', { name: 'Add' }).click();
    await labelResponse;
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('e2e-label')).toBeVisible({ timeout: 10000 });
  });

  test('Vault Settings Tags - create a tag', async ({ page }) => {
    await registerAndCreateVault(page, 'vtag');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Tags' }).click();
    await page.waitForLoadState('networkidle');

    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill('e2e-vault-tag');

    const tagResponse = page.waitForResponse(
      (resp) => resp.url().includes('/tags') && resp.request().method() === 'POST'
    );
    await page.getByRole('button', { name: 'Add' }).click();
    await tagResponse;
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('e2e-vault-tag')).toBeVisible({ timeout: 10000 });
  });

  test('Vault Settings Date Types - seed types exist', async ({ page }) => {
    await registerAndCreateVault(page, 'vdate');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Important Date Types' }).click();
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Birthdate', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Deceased date', { exact: true })).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Vault Reports', () => {
  test('should render reports with data sections', async ({ page }) => {
    await registerAndCreateVault(page, 'reports');

    await page.getByText('Reports').click();
    await expect(page).toHaveURL(/\/reports/, { timeout: 10000 });

    await expect(page.getByRole('heading', { name: 'Reports' })).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-statistic').first()).toBeVisible({ timeout: 10000 });

    await expect(page.getByText('Address Distribution')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Important Dates Overview')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Mood Trends')).toBeVisible({ timeout: 5000 });
  });
});
