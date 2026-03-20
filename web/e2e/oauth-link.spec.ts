import { test, expect } from '@playwright/test';

const API_BASE = 'http://localhost:8080/api';

async function registerUser(page: import('@playwright/test').Page, email: string) {
  await page.goto('/register');
  await page.getByPlaceholder('First name').fill('Test');
  await page.getByPlaceholder('Last name').fill('User');
  await page.getByPlaceholder('Email').fill(email);
  await page.getByPlaceholder(/password/i).fill('password123');
  await page.getByRole('button', { name: /create account/i }).click();
  await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
}

async function registerAndGetToken(email: string): Promise<string> {
  const res = await fetch(`${API_BASE}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      first_name: 'Test',
      last_name: 'User',
      email,
      password: 'password123',
    }),
  });
  const json = await res.json();
  return json.data.token;
}

function buildFakeLinkToken(provider: string, email: string, name: string): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(JSON.stringify({
    provider,
    provider_user_id: `${provider}-test-${Date.now()}`,
    email,
    name,
    exp: Math.floor(Date.now() / 1000) + 300,
  }));
  return `${header}.${payload}.fake-signature`;
}

test.describe('OAuth Account Linking', () => {
  test('OAuthCallback redirects to oauth-link page when link_token is present', async ({ page }) => {
    await page.goto('/auth/callback?link_token=some-test-token');
    await expect(page).toHaveURL(/\/auth\/oauth-link\?link_token=some-test-token/);
  });

  test('OAuthCallback stores token and navigates to vaults', async ({ page }) => {
    const email = `oauth-cb-${Date.now()}@example.com`;
    const jwtToken = await registerAndGetToken(email);
    await page.goto('/login');
    await page.evaluate((t) => localStorage.setItem('token', t), jwtToken);
    await page.goto(`/auth/callback?token=${jwtToken}`);
    await expect(page).toHaveURL(/\/vaults/, { timeout: 10000 });
  });

  test('OAuth link page shows login and register tabs', async ({ page }) => {
    const fakeLinkToken = buildFakeLinkToken('github', 'oauth-test@github.com', 'OAuth Test User');
    await page.goto(`/auth/oauth-link?link_token=${fakeLinkToken}`);
    await page.waitForLoadState('networkidle');
    await expect(page.getByText(/link your account/i).first()).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('GitHub', { exact: true })).toBeVisible();
    await expect(page.getByText('oauth-test@github.com', { exact: true })).toBeVisible();
    await expect(page.getByText(/login & link/i)).toBeVisible();
    await expect(page.getByText(/register & link/i)).toBeVisible();
  });

  test('OAuth link page redirects to login when no link_token', async ({ page }) => {
    await page.goto('/auth/oauth-link');
    await expect(page).toHaveURL(/\/login/, { timeout: 5000 });
  });

  test('OAuth link-register endpoint rejects invalid token', async () => {
    const res = await fetch(`${API_BASE}/auth/oauth/link-register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        link_token: 'invalid-token',
        first_name: 'Test',
        last_name: 'User',
        email: 'test@example.com',
        password: 'password123',
      }),
    });
    expect(res.status).toBe(400);
  });

  test('OAuth link endpoint requires authentication', async () => {
    const res = await fetch(`${API_BASE}/auth/oauth/link`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ link_token: 'some-token' }),
    });
    expect(res.status).toBe(401);
  });

  test('OAuth link endpoint rejects invalid token for authenticated user', async () => {
    const email = `oauth-link-e2e-${Date.now()}@example.com`;
    const jwtToken = await registerAndGetToken(email);

    const res = await fetch(`${API_BASE}/auth/oauth/link`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${jwtToken}`,
      },
      body: JSON.stringify({ link_token: 'invalid-token' }),
    });
    expect(res.status).toBe(400);
  });

  test('OAuth settings page renders for authenticated user', async ({ page }) => {
    const email = `oauth-settings-${Date.now()}@example.com`;
    await registerUser(page, email);
    await page.goto('/settings/oauth');
    await page.waitForLoadState('networkidle');
    await expect(page.getByRole('heading', { name: 'Connected Accounts' })).toBeVisible({ timeout: 15000 });
  });
});
