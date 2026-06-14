import { test, expect } from '@playwright/test';
import type { BrowserContext, Page } from '@playwright/test';

async function registerUser(page: import('@playwright/test').Page, email: string) {
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Test');
  await page.getByPlaceholder('Last name').fill('User');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

async function addVirtualAuthenticator(context: BrowserContext, page: Page) {
  const cdp = await context.newCDPSession(page);
  await cdp.send('WebAuthn.enable');
  await cdp.send('WebAuthn.addVirtualAuthenticator', {
    options: {
      protocol: 'ctap2',
      transport: 'internal',
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
}

test.describe('Authentication', () => {
  test.describe.configure({ timeout: 60000 });

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

  test('should finish passkey login without returning internal error', async ({ page, context }) => {
    const authEvents: string[] = [];
    page.on('console', msg => authEvents.push(`console.${msg.type()}: ${msg.text()}`));
    page.on('pageerror', err => authEvents.push(`pageerror: ${err.message}`));
    page.on('requestfailed', request => {
      if (request.url().includes('/api/auth/webauthn')) {
        authEvents.push(`requestfailed: ${request.url()} ${request.failure()?.errorText ?? ''}`);
      }
    });
    page.on('response', response => {
      if (response.url().includes('/api/auth/webauthn')) {
        authEvents.push(`response: ${response.status()} ${response.url()}`);
      }
    });

    const instanceInfoResponse = await page.request.get('/api/instance/info');
    expect(instanceInfoResponse.status()).toBe(200);
    const instanceInfo = await instanceInfoResponse.json() as { data?: { webauthn_enabled?: boolean } };
    expect(
      instanceInfo.data?.webauthn_enabled,
      'Playwright config must enable WebAuthn so this regression test cannot silently skip',
    ).toBe(true);

    await addVirtualAuthenticator(context, page);

    const email = `passkey-${Date.now()}@example.com`;
    await registerUser(page, email);

    await page.goto('/settings/webauthn');
    await expect(page.getByRole('button', { name: /register new key/i })).toBeVisible({ timeout: 10000 });

    const registrationFinish = page.waitForResponse(resp =>
      resp.url().includes('/api/settings/webauthn/register/finish'),
    );
    await page.getByRole('button', { name: /register new key/i }).click();
    const registrationResponse = await registrationFinish;
    expect(
      registrationResponse.status(),
      `passkey registration finish returned ${registrationResponse.status()} from ${registrationResponse.url()}`,
    ).toBe(201);

    await page.evaluate(() => localStorage.clear());
    await page.goto('/login');
    await page.getByPlaceholder('Email').fill(email);
    await expect(page.getByRole('button', { name: /sign in with passkey/i })).toBeVisible({ timeout: 10000 });

    const loginBegin = page.waitForResponse(
      resp => resp.url().includes('/api/auth/webauthn/login/begin'),
      { timeout: 10000 },
    ).catch(error => {
      throw new Error(`passkey login begin response was not observed\n${authEvents.join('\n')}\n${String(error)}`);
    });
    const loginFinish = page.waitForResponse(
      resp => resp.url().includes('/api/auth/webauthn/login/finish'),
      { timeout: 10000 },
    ).catch(error => {
      throw new Error(`passkey login finish response was not observed\n${authEvents.join('\n')}\n${String(error)}`);
    });
    await page.getByRole('button', { name: /sign in with passkey/i }).click();
    const beginResponse = await loginBegin;
    expect(
      beginResponse.status(),
      `passkey login begin returned ${beginResponse.status()} from ${beginResponse.url()}\n${authEvents.join('\n')}`,
    ).toBe(200);

    const loginResponse = await loginFinish;

    expect(
      loginResponse.status(),
      `passkey login finish returned ${loginResponse.status()} from ${loginResponse.url()}\n${authEvents.join('\n')}`,
    ).toBe(200);
    await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
  });
});
