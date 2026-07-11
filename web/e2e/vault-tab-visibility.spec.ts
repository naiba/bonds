import { expect, test } from '@playwright/test';

let counter = 0;

function uniqueEmail(): string {
  counter += 1;
  return `tab-visibility-${Date.now()}-${counter}-${Math.random().toString(36).slice(2, 6)}@example.com`;
}

async function registerAndCreateVault(page: import('@playwright/test').Page): Promise<string> {
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Tab');
  await page.getByPlaceholder('Last name').fill('Tester');
  await page.getByPlaceholder('Email').fill(uniqueEmail());
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 15000 });

  await page.getByRole('button', { name: /new vault/i }).click();
  await page.getByPlaceholder(/e\.g\. family/i).fill('Visibility Vault');
  await page.getByPlaceholder(/what is this vault/i).fill('Navigation visibility testing');
  await page.getByRole('button', { name: /create vault/i }).click();
  await expect(page).toHaveURL(/\/vaults\/[a-f0-9-]{36}$/, { timeout: 20000 });
  await page.waitForLoadState('networkidle');

  return page.url();
}

test.describe('Vault tab visibility', () => {
  test('hides a configured navigation tab immediately and after reload', async ({ page }) => {
    // Given: Groups is visible in a newly-created vault.
    const vaultUrl = await registerAndCreateVault(page);
    const navigation = page.getByRole('navigation');
    await expect(navigation.getByText('Groups')).toBeVisible();

    // When: the user disables the Groups tab in Vault Settings.
    await page.goto(`${vaultUrl}/settings`);
    await page.getByRole('tab', { name: 'Tab Visibility' }).click();
    const groupSetting = page.getByRole('listitem').filter({
      has: page.getByRole('heading', { name: 'Show Groups tab' }),
    });
    const visibilityResponse = page.waitForResponse(
      (response) => response.url().includes('/settings/visibility') && response.request().method() === 'PUT',
    );
    await groupSetting.getByRole('switch').click();
    const response = await visibilityResponse;
    expect(response.ok()).toBeTruthy();

    // Then: the top navigation updates now and remains hidden after a full reload.
    await expect(navigation.getByText('Groups')).toHaveCount(0);
    await page.reload();
    await expect(page.getByRole('navigation').getByText('Groups')).toHaveCount(0);
    await expect(page.getByRole('navigation').getByText('Dashboard')).toBeVisible();
    await expect(page.getByRole('navigation').getByText('Contacts')).toBeVisible();
  });
});
