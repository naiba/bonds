import { test, expect } from '@playwright/test';

async function enableAlternativeCalendar(page: import('@playwright/test').Page) {
  await page.goto('/settings/preferences');
  await page.waitForLoadState('networkidle');
  const toggle = page.locator('.ant-form-item').filter({ hasText: /alternative calendar/i }).locator('.ant-switch');
  const isChecked = await toggle.getAttribute('aria-checked');
  if (isChecked !== 'true') {
    await toggle.click();
  }
  await page.getByRole('button', { name: /save/i }).click();
  await page.waitForLoadState('networkidle');
}

async function setupContactPage(page: import('@playwright/test').Page) {
  const email = `calendar-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Calendar');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await enableAlternativeCalendar(page);

  await page.goto('/vaults');
  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Cal Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('Calendar testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');

  await page.getByText('View all contacts').click();
  await expect(page).toHaveURL(/\/contacts/, { timeout: 5000 });

  await page.getByRole('button', { name: /add contact/i }).click();
  await page.getByPlaceholder('First name').fill('Lunar');
  await page.getByPlaceholder('Last name').fill('Test');
  await page.getByRole('button', { name: /create contact/i }).click();
  await expect(page.getByText('Lunar Test')).toBeVisible({ timeout: 5000 });
}

test.describe('Calendar System', () => {
  test('should show calendar type switcher in important dates modal', async ({ page }) => {
    await setupContactPage(page);

    await page.getByRole('tab', { name: 'Contact information' }).click();
    const importantDatesCard = page.locator('.ant-card').filter({ hasText: 'Important Dates' });
    await importantDatesCard.getByRole('button', { name: /add/i }).click();

    await expect(page.getByText('Gregorian', { exact: true })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Chinese Lunar', { exact: true })).toBeVisible({ timeout: 5000 });
  });

  test('should show calendar type switcher in reminders modal', async ({ page }) => {
    await setupContactPage(page);

    await page.getByRole('tab', { name: 'Information', exact: true }).click();
    const remindersCard = page.locator('.ant-card').filter({ hasText: 'Reminders' });
    await remindersCard.getByRole('button', { name: /add/i }).click();

    await expect(page.getByText('Gregorian', { exact: true })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Chinese Lunar', { exact: true })).toBeVisible({ timeout: 5000 });
  });

  test('should create an important date with lunar calendar', async ({ page }) => {
    await setupContactPage(page);

    await page.getByRole('tab', { name: 'Contact information' }).click();
    const importantDatesCard = page.locator('.ant-card').filter({ hasText: 'Important Dates' });
    await importantDatesCard.getByRole('button', { name: /add/i }).click();

    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible({ timeout: 5000 });

    const typeFormItem = modal.locator('.ant-form-item').filter({ hasText: /type/i });
    await typeFormItem.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown').last().locator('.ant-select-item-option').first().click();
    await modal.locator('.ant-modal-header').click();

    await modal.getByRole('textbox').first().fill('Lunar Birthday');

    await page.getByText('Chinese Lunar', { exact: true }).click();

    const compactSelects = modal.locator('.ant-space-compact .ant-select');
    await expect(compactSelects).toHaveCount(3, { timeout: 5000 });

    await compactSelects.nth(1).click();
    await page.locator('.ant-select-dropdown').last().locator('.ant-select-item-option').first().click();
    await modal.locator('.ant-modal-header').click();

    await modal.getByRole('button', { name: /ok/i }).click();

    await expect(modal).not.toBeVisible({ timeout: 15000 });
    await expect(importantDatesCard.getByText('Lunar Birthday')).toBeVisible({ timeout: 10000 });
    await expect(importantDatesCard.locator('.ant-tag').filter({ hasText: 'lunar' })).toBeVisible({ timeout: 5000 });
  });

  test('should create a reminder with lunar calendar', async ({ page }) => {
    await setupContactPage(page);

    await page.getByRole('tab', { name: 'Information', exact: true }).click();
    const remindersCard = page.locator('.ant-card').filter({ hasText: 'Reminders' });
    await remindersCard.getByRole('button', { name: /add/i }).click();

    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible({ timeout: 5000 });

    await modal.getByRole('textbox').first().fill('Lunar Reminder');

    await page.getByText('Chinese Lunar', { exact: true }).click();

    const compactSelects = modal.locator('.ant-space-compact .ant-select');
    await expect(compactSelects).toHaveCount(3, { timeout: 5000 });

    await compactSelects.nth(1).click();
    await page.locator('.ant-select-dropdown').last().locator('.ant-select-item-option').first().click();
    await modal.locator('.ant-modal-header').click();

    const freqFormItem = modal.locator('.ant-form-item').filter({ hasText: /frequency/i });
    await freqFormItem.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown').last().locator('.ant-select-item-option').filter({ hasText: /yearly/i }).click();
    await modal.locator('.ant-modal-header').click();

    await modal.getByRole('button', { name: /ok/i }).click();

    await expect(modal).not.toBeVisible({ timeout: 10000 });
    await expect(remindersCard.getByText('Lunar Reminder')).toBeVisible({ timeout: 5000 });
    await expect(remindersCard.locator('.ant-tag').filter({ hasText: 'lunar' })).toBeVisible({ timeout: 5000 });
  });
});
