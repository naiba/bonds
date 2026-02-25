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

    // Archive button is now inside the More dropdown (ellipsis menu)
    await page.locator('button').filter({ has: page.locator('.anticon-more') }).click();
    const archiveMenuItem = page.locator('.ant-dropdown-menu-item').filter({ hasText: /Archive/i });
    await expect(archiveMenuItem).toBeVisible({ timeout: 10000 });

    const archiveResp = page.waitForResponse(
      (resp) => resp.url().includes('/archive') && resp.request().method() === 'PUT'
    );
    await archiveMenuItem.click();
    const resp = await archiveResp;
    expect(resp.status()).toBeLessThan(400);

    // After archiving, the More menu should now show Unarchive
    await page.locator('button').filter({ has: page.locator('.anticon-more') }).click();
    await expect(page.locator('.ant-dropdown-menu-item').filter({ hasText: /Unarchive/i })).toBeVisible({ timeout: 10000 });
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

test.describe('Contact Modules - Religion', () => {
  test('should update contact religion via select', async ({ page }) => {
    await setupVault(page, 'religion');
    await goToContacts(page);
    await createContact(page, 'Rel', 'Tester');

    // Dynamic tab name from seed: "Contact information"
    await navigateToTab(page, 'Contact information');

    // ExtraInfoModule renders multiple cards: Religion, Job Info, Companies.
    // Use card title text precisely to avoid matching contact name.
    const religionCard = page.locator('.ant-card').filter({
      has: page.locator('h5', { hasText: 'Religion' }),
    });
    await expect(religionCard).toBeVisible({ timeout: 10000 });

    await religionCard.getByRole('button', { name: /edit/i }).click();

    // The modal title is "Religion"
    const modal = page.locator('.ant-modal').filter({
      has: page.locator('.ant-modal-title', { hasText: 'Religion' }),
    });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Select a religion from the dropdown
    await modal.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/religion') && resp.request().method() === 'PUT'
    );
    await modal.getByRole('button', { name: /save/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    // After save, "No religion set" should be gone
    await expect(religionCard.getByText('No religion set')).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Job Information', () => {
  test('should update job position', async ({ page }) => {
    await setupVault(page, 'jobinfo');
    await goToContacts(page);
    await createContact(page, 'Job', 'Tester');

    await navigateToTab(page, 'Contact information');

    // Job Info card has title "Job Information" (i18n: contact.detail.job_info)
    const jobCard = page.locator('.ant-card').filter({
      has: page.locator('h5', { hasText: 'Job Information' }),
    });
    await expect(jobCard).toBeVisible({ timeout: 10000 });

    await jobCard.getByRole('button', { name: /edit/i }).click();

    // Modal title is "Edit Job Info" (i18n: contact.detail.edit_job)
    const modal = page.locator('.ant-modal').filter({
      has: page.locator('.ant-modal-title', { hasText: 'Edit Job Info' }),
    });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // The form field label is "Position" (i18n: contact.detail.job_position)
    await modal.getByLabel('Position').fill('Software Engineer');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/jobInformation') && resp.request().method() === 'PUT'
    );
    await modal.getByRole('button', { name: /save/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(jobCard.getByText('Software Engineer')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Documents', () => {
  test('should show documents module with upload area', async ({ page }) => {
    await setupVault(page, 'docs');
    await goToContacts(page);
    await createContact(page, 'Docs', 'Tester');

    // Dynamic tab: "Information" (exact match to avoid "Contact information")
    await navigateToTab(page, 'Information', true);

    // Documents card title is "Documents"
    const docsCard = page.locator('.ant-card').filter({
      has: page.locator('span', { hasText: 'Documents' }),
    }).filter({
      has: page.locator('.ant-upload-drag'),
    });
    await expect(docsCard).toBeVisible({ timeout: 10000 });

    await expect(docsCard.locator('.ant-upload-drag')).toBeVisible({ timeout: 5000 });
    await expect(docsCard.getByText('No documents')).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Contact Modules - Photos', () => {
  test('should show photos module with upload area', async ({ page }) => {
    await setupVault(page, 'photos');
    await goToContacts(page);
    // Avoid "Photos" in contact name to prevent selector ambiguity
    await createContact(page, 'Pic', 'Tester');

    // Dynamic tab: "Information" (exact match)
    await navigateToTab(page, 'Information', true);

    // Photos card — match by title span text AND presence of upload area
    const photosCard = page.locator('.ant-card').filter({
      has: page.locator('span', { hasText: 'Photos' }),
    }).filter({
      has: page.locator('.ant-upload-drag'),
    });
    await expect(photosCard).toBeVisible({ timeout: 10000 });

    await expect(photosCard.locator('.ant-upload-drag')).toBeVisible({ timeout: 5000 });
    await expect(photosCard.getByText('No photos uploaded')).toBeVisible({ timeout: 5000 });
  });
});
