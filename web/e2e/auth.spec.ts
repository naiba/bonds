import { test, expect } from '@playwright/test';

async function registerUser(page: import('@playwright/test').Page, email: string) {
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Test');
  await page.getByPlaceholder('Last name').fill('User');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

test.describe('Authentication', () => {
  test('should show login page', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByText('Welcome back')).toBeVisible();
    await expect(page.getByPlaceholder('Email')).toBeVisible();
    await expect(page.getByPlaceholder('Password')).toBeVisible();
  });

  test('should show register page', async ({ page }) => {
    await page.goto('/register');
    await expect(page.getByText('Create an account')).toBeVisible();
    await expect(page.getByPlaceholder('First name')).toBeVisible();
    await expect(page.getByPlaceholder('Email')).toBeVisible();
  });

  test('should register a new user and redirect', async ({ page }) => {
    await registerUser(page, `test-${Date.now()}@example.com`);
  });

  test('should login with valid credentials', async ({ page }) => {
    const email = `login-${Date.now()}@example.com`;
    const password = 'password123';

    await registerUser(page, email);

    await page.evaluate(() => localStorage.clear());
    await page.goto('/login');
    await page.getByPlaceholder('Email').fill(email);
    await page.getByPlaceholder('Password').fill(password);
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();
    await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
  });

  test('should show error on invalid login', async ({ page }) => {
    await page.goto('/login');
    await page.getByPlaceholder('Email').fill('wrong@example.com');
    await page.getByPlaceholder('Password').fill('wrongpassword');
    await page.getByRole('button', { name: 'Sign in', exact: true }).click();
    await expect(page.getByText(/failed|error|invalid|incorrect/i)).toBeVisible({ timeout: 5000 });
  });

  test('should redirect to login when accessing protected route', async ({ page }) => {
    await page.goto('/vaults');
    await expect(page).toHaveURL(/\/login/);
  });
});
