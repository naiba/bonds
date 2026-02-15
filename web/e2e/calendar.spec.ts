import { test, expect } from '@playwright/test';

async function setupContactPage(page: import('@playwright/test').Page) {
  const email = `calendar-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Calendar');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

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

    const addButton = page.locator('button').filter({ hasText: /add/i }).filter({
      has: page.locator('[aria-label="plus"]')
    }).first();

    const importantDatesCard = page.locator('.ant-card').filter({ hasText: 'Important Dates' });
    await importantDatesCard.getByRole('button', { name: /add/i }).click();

    await expect(page.getByText('Gregorian')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Chinese Lunar')).toBeVisible({ timeout: 5000 });
  });

  test('should show calendar type switcher in reminders modal', async ({ page }) => {
    await setupContactPage(page);

    const remindersCard = page.locator('.ant-card').filter({ hasText: 'Reminders' });
    await remindersCard.getByRole('button', { name: /add/i }).click();

    await expect(page.getByText('Gregorian')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Chinese Lunar')).toBeVisible({ timeout: 5000 });
  });

  test('should create an important date with lunar calendar', async ({ page }) => {
    await setupContactPage(page);

    // Open important dates modal
    const importantDatesCard = page.locator('.ant-card').filter({ hasText: 'Important Dates' });
    await importantDatesCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal-content');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Fill label
    await modal.getByRole('textbox').first().fill('Lunar Birthday');

    // Switch to Chinese Lunar
    await page.getByText('Chinese Lunar').click();

    // Wait for lunar selects to appear (3 selects in Space.Compact)
    const compactSelects = modal.locator('.ant-space-compact .ant-select');
    await expect(compactSelects).toHaveCount(3, { timeout: 5000 });

    // Year select (showSearch enabled): type to filter, then pick 2025
    await compactSelects.nth(0).locator('.ant-select-selection-item').click();
    await page.keyboard.type('2025');
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: /^2025$/ }).click();

    // Month select: pick first month (正月)
    await compactSelects.nth(1).click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    // Day select: pick day 15
    await compactSelects.nth(2).click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: /^15$/ }).click();

    // Type select (outside Space.Compact)
    const typeFormItem = modal.locator('.ant-form-item').filter({ hasText: /type/i });
    await typeFormItem.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    // Submit
    await page.locator('.ant-modal-footer').getByRole('button', { name: /ok/i }).click();

    // Verify modal closed and date appears in list
    await expect(modal).not.toBeVisible({ timeout: 10000 });
    await expect(importantDatesCard.getByText('Lunar Birthday')).toBeVisible({ timeout: 5000 });
    await expect(importantDatesCard.locator('.ant-tag').filter({ hasText: 'lunar' })).toBeVisible({ timeout: 5000 });
  });

  test('should create a reminder with lunar calendar', async ({ page }) => {
    await setupContactPage(page);

    // Open reminders modal
    const remindersCard = page.locator('.ant-card').filter({ hasText: 'Reminders' });
    await remindersCard.getByRole('button', { name: /add/i }).click();

    const modal = page.locator('.ant-modal-content');
    await expect(modal).toBeVisible({ timeout: 5000 });

    // Fill label
    await modal.getByRole('textbox').first().fill('Lunar Reminder');

    // Switch to Chinese Lunar
    await page.getByText('Chinese Lunar').click();

    // Wait for lunar selects to appear (3 selects in Space.Compact)
    const compactSelects = modal.locator('.ant-space-compact .ant-select');
    await expect(compactSelects).toHaveCount(3, { timeout: 5000 });

    // Year select (showSearch enabled): type to filter, then pick 2025
    await compactSelects.nth(0).locator('.ant-select-selection-item').click();
    await page.keyboard.type('2025');
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: /^2025$/ }).click();

    // Month select: pick first month (正月)
    await compactSelects.nth(1).click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').first().click();

    // Day select: pick day 15
    await compactSelects.nth(2).click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: /^15$/ }).click();

    // Frequency select (outside Space.Compact)
    const freqFormItem = modal.locator('.ant-form-item').filter({ hasText: /frequency/i });
    await freqFormItem.locator('.ant-select').click();
    await page.locator('.ant-select-dropdown:visible .ant-select-item-option').filter({ hasText: /yearly/i }).click();

    // Submit
    await page.locator('.ant-modal-footer').getByRole('button', { name: /ok/i }).click();

    // Verify modal closed and reminder appears in list
    await expect(modal).not.toBeVisible({ timeout: 10000 });
    await expect(remindersCard.getByText('Lunar Reminder')).toBeVisible({ timeout: 5000 });
    await expect(remindersCard.locator('.ant-tag').filter({ hasText: 'lunar' })).toBeVisible({ timeout: 5000 });
  });
});
