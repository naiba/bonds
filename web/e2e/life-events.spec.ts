import { test, expect } from '@playwright/test';

async function setupVault(page: import('@playwright/test').Page) {
  const email = `levt-${Date.now()}@example.com`;
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('LifeEvent');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('LE Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('For life event testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[^/]+$/, { timeout: 10000 });
  await page.waitForLoadState('networkidle');
}

function getVaultUrl(page: import('@playwright/test').Page): string {
  return page.url();
}

async function navigateToLifeEventsTab(page: import('@playwright/test').Page) {
  const vaultUrl = getVaultUrl(page);
  await page.goto(vaultUrl + '/settings');
  await page.waitForLoadState('networkidle');
  await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10000 });
  await page.getByRole('tab', { name: /life events/i }).click();
  await page.waitForLoadState('networkidle');
}

test.describe('Vault Settings - Life Events', () => {
  test('should navigate to vault settings life events tab', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);
    await expect(page.getByText('Add Category')).toBeVisible({ timeout: 10000 });
  });

  test('should show life events card with categories', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);

    const lifeEventsCard = page.locator('.ant-card').filter({ hasText: 'Life Events' });
    await expect(lifeEventsCard).toBeVisible({ timeout: 10000 });
  });

  test('should show add category input and button', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);

    const addCard = page.locator('.ant-card').filter({ hasText: 'Add Category' });
    await expect(addCard).toBeVisible({ timeout: 10000 });
    await expect(addCard.getByPlaceholder(/name/i)).toBeVisible({ timeout: 5000 });
    await expect(addCard.getByRole('button', { name: /add/i })).toBeVisible({ timeout: 5000 });
  });

  test('should show seed life event categories in collapse', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);

    const lifeEventsCard = page.locator('.ant-card').filter({ hasText: 'Life Events' });
    await expect(lifeEventsCard).toBeVisible({ timeout: 10000 });
    await expect(lifeEventsCard.locator('[class*="collapse"]').first()).toBeVisible({ timeout: 10000 });
  });

  test('should show collapse component inside life events card', async ({ page }) => {
    await setupVault(page);
    await navigateToLifeEventsTab(page);

    const lifeEventsCard = page.locator('.ant-card').filter({ hasText: 'Life Events' });
    await expect(lifeEventsCard).toBeVisible({ timeout: 10000 });
    await expect(lifeEventsCard.locator('[role="tablist"]')).toBeVisible({ timeout: 5000 });
  });
});
