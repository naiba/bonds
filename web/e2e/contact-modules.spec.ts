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

test.describe('Contact Modules - Notes', () => {
  test('should create a note', async ({ page }) => {
    await setupVault(page, 'note-create');
    await goToContacts(page);
    await createContact(page, 'NoteTest', 'User');

    await navigateToTab(page, 'Information', true);

    const notesCard = page.locator('.ant-card').filter({ hasText: /^Notes/ });
    await expect(notesCard).toBeVisible({ timeout: 10000 });
    await notesCard.getByRole('button', { name: /add/i }).click();

    await notesCard.getByPlaceholder(/title/i).fill('Test Note Title');
    await notesCard.locator('textarea').fill('This is a test note body');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/notes') && resp.request().method() === 'POST'
    );
    await notesCard.getByRole('button', { name: /save/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(notesCard.getByText('Test Note Title')).toBeVisible({ timeout: 10000 });
  });

  test('should delete a note', async ({ page }) => {
    await setupVault(page, 'note-delete');
    await goToContacts(page);
    await createContact(page, 'NoteDelTest', 'User');

    await navigateToTab(page, 'Information', true);

    const notesCard = page.locator('.ant-card').filter({ hasText: /^Notes/ });
    await expect(notesCard).toBeVisible({ timeout: 10000 });

    await notesCard.getByRole('button', { name: /add/i }).click();
    await notesCard.getByPlaceholder(/title/i).fill('Delete Me');
    await notesCard.locator('textarea').fill('Note to delete');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/notes') && resp.request().method() === 'POST'
    );
    await notesCard.getByRole('button', { name: /save/i }).click();
    await createResp;

    await expect(notesCard.getByText('Delete Me')).toBeVisible({ timeout: 10000 });

    await notesCard.getByRole('button', { name: /delete/i }).first().click();
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();

    await expect(notesCard.getByText('Delete Me')).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Tasks', () => {
  test('should create and toggle a task', async ({ page }) => {
    await setupVault(page, 'task-create');
    await goToContacts(page);
    await createContact(page, 'TaskTest', 'User');

    await navigateToTab(page, 'Information', true);

    const tasksCard = page.locator('.ant-card').filter({ hasText: /^Tasks/ });
    await expect(tasksCard).toBeVisible({ timeout: 10000 });

    await tasksCard.getByRole('button', { name: /add/i }).click();

    const input = tasksCard.getByPlaceholder(/task/i);
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill('Buy groceries');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/tasks') && resp.request().method() === 'POST'
    );
    await tasksCard.getByRole('button', { name: /save|add|ok/i }).first().click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(tasksCard.getByText('Buy groceries')).toBeVisible({ timeout: 10000 });

    const toggleResp = page.waitForResponse(
      (resp) => resp.url().includes('/toggle') && resp.request().method() === 'PUT'
    );
    await tasksCard.locator('.ant-checkbox').first().click();
    await toggleResp;
  });
});

test.describe('Contact Modules - Reminders', () => {
  test('should create a reminder', async ({ page }) => {
    await setupVault(page, 'reminder');
    await goToContacts(page);
    await createContact(page, 'ReminderTest', 'User');

    await navigateToTab(page, 'Information', true);

    const remindersCard = page.locator('.ant-card').filter({ hasText: /Reminders/ });
    await expect(remindersCard).toBeVisible({ timeout: 10000 });
    await remindersCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /reminder/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.getByLabel(/label/i).fill('Birthday reminder');

    await modal.locator('.ant-picker').click();
    const dateCell = page.locator('.ant-picker-dropdown:visible .ant-picker-cell:not(.ant-picker-cell-disabled):not(.ant-picker-cell-today)').first();
    await dateCell.click();

    await page.locator('.ant-modal .ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/reminders') && resp.request().method() === 'POST'
    );
    await page.locator('.ant-modal-footer .ant-btn-primary').click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(remindersCard.getByText('Birthday reminder')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Addresses', () => {
  test('should create an address', async ({ page }) => {
    await setupVault(page, 'address');
    await goToContacts(page);
    await createContact(page, 'AddressTest', 'User');
    await navigateToTab(page, 'Contact information');

    const addressCard = page.locator('.ant-card').filter({ hasText: /Addresses/ });
    await expect(addressCard).toBeVisible({ timeout: 10000 });
    await addressCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });
    await modal.getByLabel(/address line 1/i).fill('123 Main St');
    await modal.getByLabel(/city/i).fill('San Francisco');
    await modal.getByLabel(/country/i).fill('USA');
    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/addresses') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);
  });

  test('should add and delete an address', async ({ page }) => {
    await setupVault(page, 'addr-del');
    await goToContacts(page);
    await createContact(page, 'AddrDelTest', 'User');

    await navigateToTab(page, 'Contact information');

    const addressCard = page.locator('.ant-card').filter({ hasText: /Addresses/ });
    await expect(addressCard).toBeVisible({ timeout: 10000 });
    await addressCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal');
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.getByLabel(/address line 1/i).fill('742 Evergreen Terrace');
    await modal.getByLabel(/city/i).fill('Springfield');
    await modal.getByLabel(/country/i).fill('USA');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/addresses') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;

    await expect(addressCard.getByText('742 Evergreen Terrace')).toBeVisible({ timeout: 10000 });

    const addressItem = addressCard.locator('.ant-list-item').filter({ hasText: '742 Evergreen Terrace' });
    await addressItem.locator('[aria-label="delete"]').click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/addresses') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(addressCard.getByText('742 Evergreen Terrace')).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Pets', () => {
  test('should create a pet', async ({ page }) => {
    await setupVault(page, 'pet');
    await goToContacts(page);
    await createContact(page, 'PetTest', 'User');

    await navigateToTab(page, 'Social');

    const petsCard = page.locator('.ant-card').filter({ hasText: /Pets/ });
    await expect(petsCard).toBeVisible({ timeout: 10000 });
    await petsCard.getByRole('button', { name: /add/i }).click();

    await petsCard.getByPlaceholder(/name/i).fill('Buddy');
    await petsCard.getByPlaceholder(/category/i).fill('1');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/pets') && resp.request().method() === 'POST'
    );
    await petsCard.getByRole('button', { name: /save/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(petsCard.getByText('Buddy')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Calls', () => {
  test('should log a call', async ({ page }) => {
    await setupVault(page, 'call-create');
    await goToContacts(page);
    await createContact(page, 'CallTest', 'User');

    await navigateToTab(page, 'Information', true);

    const callsCard = page.locator('.ant-card').filter({ hasText: /^Calls/ });
    await expect(callsCard).toBeVisible({ timeout: 10000 });
    await callsCard.getByRole('button', { name: /log call/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /log a call/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.locator('.ant-picker').click();
    const dateCell = page.locator('.ant-picker-dropdown:visible .ant-picker-cell-today .ant-picker-cell-inner');
    await dateCell.click();
    const timeOk = page.locator('.ant-picker-dropdown:visible .ant-picker-ok button');
    if (await timeOk.isVisible().catch(() => false)) {
      await timeOk.click();
    }

    await modal.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: /outgoing/i }).click();

    await modal.locator('.ant-input-number-input').fill('15');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/calls') && resp.request().method() === 'POST'
    );
    await page.locator('.ant-modal-footer .ant-btn-primary').click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(callsCard.getByText('outgoing')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Loans', () => {
  test('should create a loan and toggle settled', async ({ page }) => {
    await setupVault(page, 'loan-create');
    await goToContacts(page);
    await createContact(page, 'LoanTest', 'User');

    await navigateToTab(page, 'Information', true);

    const loansCard = page.locator('.ant-card').filter({ hasText: /^Loans/ });
    await expect(loansCard).toBeVisible({ timeout: 10000 });
    await loansCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /add loan/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.locator('.ant-form-item').filter({ hasText: /name/i }).first().locator('input').fill('Dinner money');

    await modal.locator('.ant-input-number-input').fill('50');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/loans') && resp.request().method() === 'POST'
    );
    await page.locator('.ant-modal-footer .ant-btn-primary').click();
    const resp = await createResp;
    expect(resp.status()).toBeLessThan(400);

    await expect(loansCard.getByText('Dinner money')).toBeVisible({ timeout: 10000 });

    const toggleResp = page.waitForResponse(
      (resp) => resp.url().includes('/toggle') && resp.request().method() === 'PUT'
    );
    await loansCard.getByRole('button', { name: /settle/i }).click();
    const toggleResult = await toggleResp;
    expect(toggleResult.status()).toBeLessThan(400);

    await expect(loansCard.getByText('Settled', { exact: true })).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Goals', () => {
  test('should create a goal', async ({ page }) => {
    await setupVault(page, 'goal-create');
    await goToContacts(page);
    await createContact(page, 'GoalTest', 'User');

    await navigateToTab(page, 'Life & goals');

    const goalsCard = page.locator('.ant-card').filter({ hasText: /^Goals/ });
    await expect(goalsCard).toBeVisible({ timeout: 10000 });
    await goalsCard.getByRole('button', { name: /add/i }).click();

    const input = goalsCard.getByPlaceholder(/goal name/i);
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill('Learn Spanish');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/goals') && resp.request().method() === 'POST'
    );
    await goalsCard.getByRole('button', { name: /^add$/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(goalsCard.getByText('Learn Spanish')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Contact Information', () => {
  test('should create a contact info entry', async ({ page }) => {
    await setupVault(page, 'cinfo-create');
    await goToContacts(page);
    await createContact(page, 'CInfoTest', 'User');

    await navigateToTab(page, 'Contact information');

    const infoCard = page.locator('.ant-card').filter({ hasText: 'Contact Information' });
    await expect(infoCard).toBeVisible({ timeout: 10000 });
    await infoCard.getByRole('button', { name: /add/i }).click();

    const valueInput = infoCard.getByPlaceholder(/value/i);
    await expect(valueInput).toBeVisible({ timeout: 5000 });
    await valueInput.fill('test@example.com');

    const responsePromise = page.waitForResponse(
      (resp) => resp.url().includes('/contactInformation') && resp.request().method() === 'POST'
    );
    await infoCard.getByRole('button', { name: /save/i }).click();
    const resp = await responsePromise;
    expect(resp.status()).toBeLessThan(400);

    await expect(infoCard.getByText('test@example.com')).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Contact Information Delete', () => {
  test('should add and delete a contact info entry', async ({ page }) => {
    await setupVault(page, 'cinfo-del');
    await goToContacts(page);
    await createContact(page, 'CInfoDelTest', 'User');

    await navigateToTab(page, 'Contact information');

    const infoCard = page.locator('.ant-card').filter({ hasText: 'Contact Information' });
    await expect(infoCard).toBeVisible({ timeout: 10000 });
    await infoCard.getByRole('button', { name: /add/i }).click();

    const valueInput = infoCard.getByPlaceholder(/value/i);
    await expect(valueInput).toBeVisible({ timeout: 5000 });
    await valueInput.fill('+1-555-0199');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/contactInformation') && resp.request().method() === 'POST'
    );
    await infoCard.getByRole('button', { name: /save/i }).click();
    await createResp;

    await expect(infoCard.getByText('+1-555-0199')).toBeVisible({ timeout: 10000 });

    const infoItem = infoCard.locator('.ant-list-item').filter({ hasText: '+1-555-0199' });
    await infoItem.locator('[aria-label="delete"]').click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/contactInformation') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(infoCard.getByText('+1-555-0199')).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Tasks Toggle', () => {
  test('should show toggle button for completed tasks', async ({ page }) => {
    await setupVault(page, 'task-toggle');
    await goToContacts(page);
    await createContact(page, 'TaskToggle', 'User');

    await navigateToTab(page, 'Information', true);

    const tasksCard = page.locator('.ant-card').filter({ hasText: /^Tasks/ });
    await expect(tasksCard).toBeVisible({ timeout: 10000 });
    await expect(tasksCard.getByText(/show completed/i)).toBeVisible({ timeout: 10000 });
  });

  test('should create task, toggle it, and see it in completed section', async ({ page }) => {
    await setupVault(page, 'task-complete');
    await goToContacts(page);
    await createContact(page, 'TaskComplete', 'User');

    await navigateToTab(page, 'Information', true);

    const tasksCard = page.locator('.ant-card').filter({ hasText: /^Tasks/ });
    await expect(tasksCard).toBeVisible({ timeout: 10000 });

    await tasksCard.getByRole('button', { name: /add/i }).click();
    const input = tasksCard.getByPlaceholder(/task/i);
    await expect(input).toBeVisible({ timeout: 5000 });
    await input.fill('Complete me');

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/tasks') && resp.request().method() === 'POST'
    );
    await tasksCard.getByRole('button', { name: /save|add|ok/i }).first().click();
    await createResp;
    await expect(tasksCard.getByText('Complete me')).toBeVisible({ timeout: 10000 });

    const toggleResp = page.waitForResponse(
      (resp) => resp.url().includes('/toggle') && resp.request().method() === 'PUT'
    );
    await tasksCard.locator('.ant-checkbox').first().click();
    await toggleResp;

    await tasksCard.getByText(/show completed/i).click();
    await expect(tasksCard.getByText(/hide completed/i)).toBeVisible({ timeout: 10000 });
    await expect(tasksCard.getByText('Complete me').first()).toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Labels', () => {
  test('should add and remove a label from a contact', async ({ page }) => {
    await setupVault(page, 'labels');
    const vaultUrl = page.url();

    // First create a label in vault settings
    await page.goto(vaultUrl + '/settings');
    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
    await page.getByRole('tab', { name: 'Labels' }).click();
    await page.waitForLoadState('networkidle');

    const nameInput = page.getByPlaceholder('Name');
    await expect(nameInput).toBeVisible({ timeout: 10000 });
    await nameInput.fill('e2e-test-label');

    const labelResponse = page.waitForResponse(
      (resp) => resp.url().includes('/labels') && resp.request().method() === 'POST'
    );
    await page.getByRole('button', { name: 'Add' }).click();
    await labelResponse;
    await page.waitForLoadState('networkidle');
    await expect(page.getByText('e2e-test-label')).toBeVisible({ timeout: 10000 });

    // Now create a contact and add the label
    await page.goto(vaultUrl);
    await page.waitForLoadState('networkidle');
    await goToContacts(page);
    await createContact(page, 'LabelTest', 'User');

    await navigateToTab(page, 'Contact information');

    const labelsCard = page.locator('.ant-card').filter({ hasText: 'Labels' });
    await expect(labelsCard).toBeVisible({ timeout: 10000 });
    await labelsCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /add label/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Select the label from dropdown
    await modal.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: 'e2e-test-label' }).click();

    const addResp = page.waitForResponse(
      (resp) => resp.url().includes('/labels') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /save/i }).click();
    await addResp;

    // Verify label appears as a tag
    await expect(labelsCard.locator('.ant-tag').filter({ hasText: 'e2e-test-label' })).toBeVisible({ timeout: 10000 });

    // Delete the label from the contact
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/labels') && resp.request().method() === 'DELETE'
    );
    await labelsCard.locator('.ant-tag').filter({ hasText: 'e2e-test-label' }).locator('[aria-label="delete"]').click();
    await deleteResp;

    await expect(labelsCard.locator('.ant-tag').filter({ hasText: 'e2e-test-label' })).not.toBeVisible({ timeout: 10000 });
  });
});

test.describe('Contact Modules - Important Dates', () => {
  test('should add and delete an important date', async ({ page }) => {
    await setupVault(page, 'idate');
    await goToContacts(page);
    await createContact(page, 'DateTest', 'User');

    await navigateToTab(page, 'Contact information');

    const datesCard = page.locator('.ant-card').filter({ hasText: 'Important Dates' });
    await expect(datesCard).toBeVisible({ timeout: 10000 });

    await datesCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal').filter({ hasText: /important date/i });
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.getByLabel(/label/i).fill('Graduation Day');

    // CalendarDatePicker renders an Ant Design DatePicker when alternative calendar is disabled (default)
    await modal.locator('.ant-picker').click();
    const dateCell = page.locator('.ant-picker-dropdown:visible .ant-picker-cell:not(.ant-picker-cell-disabled):not(.ant-picker-cell-today)').first();
    await dateCell.click();

    const createResp = page.waitForResponse(
      (resp) => resp.url().includes('/dates') && resp.request().method() === 'POST'
    );
    await modal.getByRole('button', { name: /ok/i }).click();
    await createResp;

    await expect(datesCard.getByText('Graduation Day')).toBeVisible({ timeout: 10000 });

    // Delete the important date (skip seed dates Birthdate/Deceased date, target our created one)
    const dateItem = datesCard.locator('.ant-list-item').filter({ hasText: 'Graduation Day' });
    await dateItem.locator('[aria-label="delete"]').click();
    const deleteResp = page.waitForResponse(
      (resp) => resp.url().includes('/dates') && resp.request().method() === 'DELETE'
    );
    await page.locator('.ant-popconfirm').getByRole('button', { name: /ok|yes/i }).click();
    await deleteResp;

    await expect(datesCard.getByText('Graduation Day')).not.toBeVisible({ timeout: 10000 });
  });
});
