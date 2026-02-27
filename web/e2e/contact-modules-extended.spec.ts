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


async function createGroup(page: import('@playwright/test').Page, vaultUrl: string, groupName: string) {
  await page.goto(vaultUrl + '/groups');
  await page.waitForLoadState('networkidle');

  await page.getByRole('button', { name: /new group/i }).first().click();
  const modal = page.getByRole('dialog');
  await expect(modal).toBeVisible({ timeout: 5000 });

  await modal.locator('input#name').fill(groupName);

  const responsePromise = page.waitForResponse(
    (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
  );
  await modal.getByRole('button', { name: /ok/i }).click();
  const resp = await responsePromise;
  expect(resp.status()).toBeLessThan(400);

  await expect(page.locator('.ant-list').getByText(groupName)).toBeVisible({ timeout: 15000 });
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
  // Companies are inside VaultSettings tab — extra navigation adds time.
  test.setTimeout(60000);
  test('should add and update job position', async ({ page }) => {
    await setupVault(page, 'jobinfo');
    const vaultUrl = page.url();

    // Create a company first so we can assign a job
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Companies/i }).click();
    // Wait for Companies tab content to render (Add Company button)
    await expect(page.getByRole('button', { name: /add company/i }).first()).toBeVisible({ timeout: 15000 });
    await page.getByRole('button', { name: /add company/i }).first().click();
    const companyModal = page.locator('.ant-modal');
    await expect(companyModal).toBeVisible({ timeout: 10000 });
    await companyModal.locator('input#name').fill('JobTestCorp');
    const companyResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'POST'
    );
    await companyModal.getByRole('button', { name: /ok/i }).click();
    await companyResp;
    await expect(page.getByText('JobTestCorp')).toBeVisible({ timeout: 10000 });

    // Create a contact
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');
    await goToContacts(page);
    await createContact(page, 'Job', 'Tester');

    await navigateToTab(page, 'Contact information');

    // Job Info card has title "Job Information"
    const jobCard = page.locator('.ant-card').filter({
      has: page.locator('h5', { hasText: 'Job Information' }),
    });
    await expect(jobCard).toBeVisible({ timeout: 10000 });

    // New UI: click "Add Job" button (PlusOutlined icon + text)
    await jobCard.getByRole('button', { name: /add job/i }).click();

    // Modal title is "Add Job"
    const addModal = page.locator('.ant-modal').filter({
      has: page.locator('.ant-modal-title', { hasText: 'Add Job' }),
    });
    await expect(addModal).toBeVisible({ timeout: 5000 });

    // Select a company from the dropdown
    await addModal.locator('.ant-select').first().click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'JobTestCorp' }).click();

    // Fill position
    await addModal.getByLabel('Position').fill('Software Engineer');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/jobs') && resp.request().method() === 'POST'
    );
    await addModal.getByRole('button', { name: /save/i }).click();
    const resp = await createResp;
    expect(resp.status()).toBeLessThan(400);

    // Verify job appears in the list
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

test.describe('Contact Modules - Groups', () => {
  test('contact groups module shows empty state and allows adding groups', async ({ page }) => {
    await setupVault(page, 'grp-module');
    const vaultUrl = page.url();

    // Create a group first
    await createGroup(page, vaultUrl, 'Work');

    // Create a contact
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');
    await goToContacts(page);
    await createContact(page, 'Grp', 'Contact');

    // Navigate to Social tab
    await navigateToTab(page, 'Social');

    // Assert: Groups card is visible — use "Not in any group" text to uniquely identify
    const groupsCard = page.locator('.ant-card').filter({ hasText: 'Not in any group' });
    await expect(groupsCard).toBeVisible({ timeout: 10000 });

    // Assert: empty state
    await expect(groupsCard.getByText('Not in any group')).toBeVisible({ timeout: 5000 });

    // Click Add button
    await groupsCard.getByRole('button', { name: /add/i }).click();

    // Wait for modal
    const groupModal = page.locator('.ant-modal:visible');
    await expect(groupModal).toBeVisible({ timeout: 5000 });

    // Select "Work" group from dropdown
    await groupModal.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'Work' }).click();

    // Save
    const addResp = page.waitForResponse(
      (resp) => resp.url().includes('/groups') && resp.request().method() === 'POST'
    );
    await groupModal.getByRole('button', { name: /save/i }).click();
    const resp = await addResp;
    expect(resp.status()).toBeLessThan(400);

    // Assert: "Work" tag appears in the card
    const updatedGroupsCard = page.locator('.ant-card').filter({ hasText: 'Work' }).filter({ has: page.locator('.ant-tag') });
    await expect(updatedGroupsCard.locator('.ant-tag').filter({ hasText: 'Work' })).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Job Information Company', () => {
  // Companies are inside VaultSettings tab — extra navigation adds time.
  test.setTimeout(60000);
  test('job information company dropdown shows created companies', async ({ page }) => {
    await setupVault(page, 'job-company');
    const vaultUrl = page.url();

    // Create a company first via the companies page
    await page.goto(vaultUrl + '/settings');
    await page.waitForLoadState('networkidle');
    await page.getByRole('tab', { name: /Companies/i }).click();
    // Wait for Companies tab content to render (Add Company button)
    await expect(page.getByRole('button', { name: /add company/i }).first()).toBeVisible({ timeout: 15000 });
    await page.getByRole('button', { name: /add company/i }).first().click();

    const companyModal = page.locator('.ant-modal');
    await expect(companyModal).toBeVisible({ timeout: 10000 });
    await companyModal.locator('input#name').fill('TestCorp Inc');

    const companyResp = page.waitForResponse(
      (resp) => resp.url().includes('/companies') && resp.request().method() === 'POST'
    );
    await companyModal.getByRole('button', { name: /ok/i }).click();
    const cResp = await companyResp;
    expect(cResp.status()).toBeLessThan(400);
    await expect(page.getByText('TestCorp Inc')).toBeVisible({ timeout: 10000 });

    // Create a contact
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');
    await goToContacts(page);
    await createContact(page, 'JobCo', 'Tester');

    // Navigate to Contact information tab
    await navigateToTab(page, 'Contact information');

    // Job Info card — new UI has "Add Job" button instead of "Edit"
    const jobCard = page.locator('.ant-card').filter({
      has: page.locator('h5', { hasText: 'Job Information' }),
    });
    await expect(jobCard).toBeVisible({ timeout: 10000 });
    await jobCard.getByRole('button', { name: /add job/i }).click();

    // Modal title is "Add Job" (new multi-job UI)
    const modal = page.locator('.ant-modal').filter({
      has: page.locator('.ant-modal-title', { hasText: 'Add Job' }),
    });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Click the company select dropdown and verify TestCorp Inc appears
    await modal.locator('.ant-select').first().click();
    await expect(
      page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'TestCorp Inc' })
    ).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Relationship Manage Types', () => {
  test('relationship modal shows manage types link', async ({ page }) => {
    await setupVault(page, 'rel-manage');
    await goToContacts(page);

    // Need two contacts for relationship creation
    await createContact(page, 'RelMgmt', 'Alice');

    await page.getByRole('button', { name: /back/i }).first().click();
    await expect(page).toHaveURL(/\/contacts$/, { timeout: 5000 });

    await createContact(page, 'RelMgmt', 'Bob');

    await page.getByRole('button', { name: /back/i }).first().click();
    await expect(page).toHaveURL(/\/contacts$/, { timeout: 5000 });

    // Go to Alice's page
    await page.getByText('RelMgmt Alice').click();
    await expect(page).toHaveURL(/\/contacts\/[a-f0-9-]+$/, { timeout: 10000 });

    // Navigate to Social tab
    await navigateToTab(page, 'Social');

    const relCard = page.locator('.ant-card').filter({ hasText: /Relationships/ });
    await expect(relCard).toBeVisible({ timeout: 10000 });

    // Open Add Relationship modal
    await relCard.locator('.ant-card-extra button').click();

    const modal = page.locator('.ant-modal').filter({ hasText: /relationship/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Verify "Manage Types" link is present
    await expect(modal.getByText('Manage Types')).toBeVisible({ timeout: 5000 });
    // Verify the hint text is also present
    await expect(modal.getByText(/Configure relationship types/i)).toBeVisible({ timeout: 5000 });
  });
});
