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

  test('Vault Settings Date Types - seed types and CRUD', async ({ page }) => {
    await registerAndCreateVault(page, 'vdate');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
    await page.getByRole('tab', { name: 'Important Date Types' }).click();
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('Birthdate', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Deceased date', { exact: true })).toBeVisible({ timeout: 10000 });
    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill('Graduation Day');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/dateTypes') && resp.request().method() === 'POST' && resp.status() < 400
    );
    await page.getByRole('button', { name: 'Add' }).click();
    await createResp;

    await expect(page.getByText('Graduation Day')).toBeVisible({ timeout: 10000 });

    const createdRow = page.locator('.ant-list-item').filter({ hasText: 'Graduation Day' });
    await createdRow.getByRole('button', { name: 'delete' }).click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/dateTypes') && resp.request().method() === 'DELETE' && resp.status() < 400
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(page.getByText('Graduation Day')).not.toBeVisible({ timeout: 10000 });
  });

  test('Vault Settings Mood Parameters - seed and CRUD', async ({ page }) => {
    await registerAndCreateVault(page, 'vmood');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 15000 });

    await page.getByRole('tab', { name: 'Mood Parameters' }).click();
    await page.waitForLoadState('networkidle');

    await expect(page.locator('.ant-list-item').first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-list-item')).toHaveCount(5, { timeout: 10000 });

    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill('Super Happy');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/moodParams') && resp.request().method() === 'POST' && resp.status() < 400
    );
    await page.getByRole('button', { name: 'Add' }).click();
    await createResp;

    await expect(page.getByText('Super Happy')).toBeVisible({ timeout: 10000 });

    const createdRow = page.locator('.ant-list-item').filter({ hasText: 'Super Happy' });
    await createdRow.getByRole('button', { name: 'delete' }).click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/moodParams') && resp.request().method() === 'DELETE' && resp.status() < 400
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(page.getByText('Super Happy')).not.toBeVisible({ timeout: 10000 });
    await expect(page.locator('.ant-list-item')).toHaveCount(5, { timeout: 10000 });
  });

  test('Vault Settings Quick Fact Templates - seed and CRUD', async ({ page }) => {
    await registerAndCreateVault(page, 'vqft');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Quick Fact Templates' }).click();
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('Hobbies')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Food preferences')).toBeVisible({ timeout: 10000 });

    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill('Favorite Movies');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/quickFactTemplates') && resp.request().method() === 'POST' && resp.status() < 400
    );
    await page.getByRole('button', { name: 'Add' }).click();
    await createResp;

    await expect(page.getByText('Favorite Movies')).toBeVisible({ timeout: 10000 });

    const createdRow = page.locator('.ant-list-item').filter({ hasText: 'Favorite Movies' });
    await createdRow.getByRole('button', { name: 'delete' }).click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/quickFactTemplates') && resp.request().method() === 'DELETE' && resp.status() < 400
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(page.getByText('Favorite Movies')).not.toBeVisible({ timeout: 10000 });
  });

  test('Vault Settings Mood Parameters - reorder position', async ({ page }) => {
    await registerAndCreateVault(page, 'vmoodpos');
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 15000 });
    await page.getByRole('tab', { name: 'Mood Parameters' }).click();
    await page.waitForLoadState('networkidle');

    // Wait for list items and get the second item text
    await expect(page.locator('.ant-list-item').first()).toBeVisible({ timeout: 10000 });
    const secondItem = page.locator('.ant-list-item').nth(1);
    await expect(secondItem).toBeVisible({ timeout: 5000 });
    const secondItemText = await secondItem.locator('.ant-list-item-meta-title').textContent();

    // Click the UP arrow on the second item to move it to position 0
    const upArrow = secondItem.locator('.anticon-arrow-up');
    await expect(upArrow).toBeVisible({ timeout: 5000 });

    // Set up response listeners before click to avoid race condition
    const posRespPromise = page.waitForResponse(
      (resp) => resp.url().includes('/moodParams') && resp.url().includes('/position') && resp.status() < 400
    );
    const refetchPromise = page.waitForResponse(
      (resp) => resp.url().includes('/moodParams') && resp.request().method() === 'GET' && resp.status() < 400
    );
    await upArrow.click();
    const posResp = await posRespPromise;
    expect(posResp).toBeTruthy();
    const posBody = await posResp.json();
    expect(posBody.success).toBe(true);
    await refetchPromise;
    await page.waitForLoadState('networkidle');
    const firstItemTextAfter = await page.locator('.ant-list-item').first().locator('.ant-list-item-meta-title').textContent();
    expect(firstItemTextAfter).toBe(secondItemText);
  });

  test('Vault Settings Quick Fact Templates - reorder position', async ({ page }) => {
    await registerAndCreateVault(page, 'vqftpos');
    const vaultUrl = getVaultUrl(page);
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: 'Quick Fact Templates' }).click();
    await page.waitForLoadState('networkidle');

    // Wait for list items and get the second item text
    await expect(page.locator('.ant-list-item').first()).toBeVisible({ timeout: 10000 });
    const secondItem = page.locator('.ant-list-item').nth(1);
    await expect(secondItem).toBeVisible({ timeout: 5000 });
    const secondItemText = await secondItem.locator('.ant-list-item-meta-title').textContent();

    // Click the UP arrow on the second item to move it to position 0
    const upArrow = secondItem.locator('.anticon-arrow-up');
    await expect(upArrow).toBeVisible({ timeout: 5000 });

    // Set up response listeners before click to avoid race condition
    const posRespPromise = page.waitForResponse(
      (resp) => resp.url().includes('/quickFactTemplates') && resp.url().includes('/position') && resp.status() < 400
    );
    const refetchPromise = page.waitForResponse(
      (resp) => resp.url().includes('/quickFactTemplates') && resp.request().method() === 'GET' && resp.status() < 400
    );
    await upArrow.click();
    const posResp = await posRespPromise;
    expect(posResp).toBeTruthy();
    const posBody = await posResp.json();
    expect(posBody.success).toBe(true);
    await refetchPromise;
    await page.waitForLoadState('networkidle');
    const firstItemTextAfter = await page.locator('.ant-list-item').first().locator('.ant-list-item-meta-title').textContent();
    expect(firstItemTextAfter).toBe(secondItemText);
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

test.describe('Vault Reminders Page', () => {
  test('should show reminders page with a created reminder', async ({ page }) => {
    await registerAndCreateVault(page, 'vrem');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/contacts');
    await page.waitForLoadState('networkidle');
    await page.getByRole('button', { name: /add contact/i }).click();
    await page.getByPlaceholder('First name').fill('ReminderTest');
    await page.getByPlaceholder('Last name').fill('User');
    await page.getByRole('button', { name: /create contact/i }).click();
    await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });
    await expect(page.getByText('ReminderTest User').first()).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Information', exact: true }).click();
    await page.waitForLoadState('networkidle');

    const remindersCard = page.locator('.ant-card').filter({ hasText: /Reminders/ });
    await expect(remindersCard).toBeVisible({ timeout: 10000 });
    await remindersCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /reminder/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.locator('input#label').fill('Test Vault Reminder');

    await modal.locator('.ant-picker').click();
    const dateCell = page.locator('.ant-picker-dropdown:visible .ant-picker-cell:not(.ant-picker-cell-disabled):not(.ant-picker-cell-today)').first();
    await dateCell.click();

    await modal.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    const reminderResp = page.waitForResponse(
      (resp) => resp.url().includes('/reminders') && resp.request().method() === 'POST'
    );
    await page.locator('.ant-modal-footer .ant-btn-primary').click();
    await reminderResp;

    await expect(remindersCard.getByText('Test Vault Reminder')).toBeVisible({ timeout: 10000 });

    await page.goto(vaultUrl + '/reminders');
    await expect(page).toHaveURL(/\/reminders$/, { timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 4 }).filter({ hasText: 'All Reminders' })
    ).toBeVisible({ timeout: 10000 });

    await expect(page.getByText('Test Vault Reminder')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('ReminderTest')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Vault Life Metrics', () => {
  test('should create and delete a life metric', async ({ page }) => {
    await registerAndCreateVault(page, 'vlm');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/life-metrics');
    await expect(page).toHaveURL(/\/life-metrics$/, { timeout: 10000 });
    await expect(
      page.getByRole('heading', { level: 4 }).filter({ hasText: 'Life Metrics' })
    ).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /add metric/i }).click();
    const modal = page.locator('.ant-modal').filter({ hasText: /add metric|life metric/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.locator('input#label').fill('Health Score');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/lifeMetrics') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;

    await expect(page.getByText('Health Score')).toBeVisible({ timeout: 10000 });

    await page.locator('[aria-label="delete"]').first().click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/lifeMetrics') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-modal-confirm').getByRole('button', { name: /delete|ok/i }).click();
    await deleteResp;

    await expect(page.getByText('Health Score')).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Journal CRUD', () => {
  test('should create, edit, and delete a journal', async ({ page }) => {
    await registerAndCreateVault(page, 'jcrud');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/journals');
    await expect(
      page.getByRole('heading', { level: 4 }).filter({ hasText: 'Journals' })
    ).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: /new journal/i }).click();
    const modal = page.locator('.ant-modal').filter({ hasText: /new journal/i });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('#name').fill('My CRUD Journal');
    await modal.locator('#description').fill('A test journal');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/journals') && resp.request().method() === 'POST' && resp.status() < 400
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;
    await expect(modal).not.toBeVisible({ timeout: 10000 });

    await expect(page.getByText('My CRUD Journal')).toBeVisible({ timeout: 10000 });

    await page.locator('.ant-list-item').filter({ hasText: 'My CRUD Journal' }).locator('.anticon-edit').click();
    const editModal = page.locator('.ant-modal:visible');
    await expect(editModal).toBeVisible({ timeout: 5000 });
    await editModal.locator('#name').clear();
    await editModal.locator('#name').fill('Renamed Journal');

    const updateResp = page.waitForResponse(
      (resp) => resp.url().includes('/journals') && resp.request().method() === 'PUT' && resp.status() < 400
    );
    await editModal.getByRole('button', { name: /ok/i }).click();
    await updateResp;

    await expect(page.getByText('Renamed Journal')).toBeVisible({ timeout: 10000 });

    await page.locator('.ant-list-item').filter({ hasText: 'Renamed Journal' }).locator('.anticon-delete').click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/journals') && resp.request().method() === 'DELETE' && resp.status() < 400
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(page.getByText('Renamed Journal')).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Journal Metrics', () => {
  test('should add and remove a journal metric', async ({ page }) => {
    await registerAndCreateVault(page, 'jmetric');
    const vaultUrl = getVaultUrl(page);

    await page.goto(vaultUrl + '/journals');
    await page.getByRole('button', { name: /new journal/i }).click();
    const modal = page.locator('.ant-modal').filter({ hasText: /new journal/i });
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.locator('#name').fill('Metric Test Journal');
    await modal.getByRole('button', { name: /ok/i }).click();
    await expect(modal).not.toBeVisible({ timeout: 10000 });
    await page.waitForLoadState('networkidle');

    await page.getByText('Metric Test Journal').click();
    await expect(page).toHaveURL(/\/journals\/\d+$/, { timeout: 10000 });
    await page.waitForLoadState('networkidle');

    const addMetricTag = page.locator('.ant-tag').filter({ hasText: /add metric/i });
    await expect(addMetricTag).toBeVisible({ timeout: 10000 });
    await addMetricTag.click();

    const metricInput = page.locator('input[type="text"]').last();
    await expect(metricInput).toBeVisible({ timeout: 5000 });
    await metricInput.fill('Productivity');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/metrics') && resp.request().method() === 'POST'
    );
    await metricInput.press('Enter');
    await createResp;

    const metricTag = page.locator('.ant-tag').filter({ hasText: 'Productivity' });
    await expect(metricTag).toBeVisible({ timeout: 10000 });

    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/metrics') && resp.request().method() === 'DELETE'
    );
    await metricTag.locator('.anticon-close').click();
    await deleteResp;

    await expect(page.locator('.ant-tag').filter({ hasText: 'Productivity' })).not.toBeVisible({ timeout: 10000 });
  });
});
