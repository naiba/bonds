import { defineConfig, devices } from '@playwright/test';

function readPortEnv(name: string, fallback: number): number {
  const value = process.env[name];
  if (value === undefined) {
    return fallback;
  }

  if (!/^\d+$/.test(value)) {
    throw new Error(`${name} must be a decimal port number`);
  }

  const port = Number(value);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    throw new Error(`${name} must be between 1 and 65535`);
  }

  return port;
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

const serverPort = readPortEnv('PLAYWRIGHT_SERVER_PORT', 8080);
const vitePort = readPortEnv('PLAYWRIGHT_VITE_PORT', 5173);
const webAuthnRpID = process.env.WEBAUTHN_RP_ID ?? 'localhost';
const webAuthnRpOrigins = process.env.WEBAUTHN_RP_ORIGINS ?? `http://localhost:${vitePort}`;
const webAuthnRpDisplayName = process.env.WEBAUTHN_RP_DISPLAY_NAME ?? 'Bonds E2E';
const webAuthnEnv = [
  `WEBAUTHN_RP_ID=${shellQuote(webAuthnRpID)}`,
  `WEBAUTHN_RP_ORIGINS=${shellQuote(webAuthnRpOrigins)}`,
  `WEBAUTHN_RP_DISPLAY_NAME=${shellQuote(webAuthnRpDisplayName)}`,
].join(' ');

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // E2E 共享同一个 Go 后端 + SQLite DB，多 worker 并行会导致数据污染（admin 首用户竞争、联系人表串扰等）
  workers: 1,
  reporter: 'html',
  use: {
    baseURL: `http://localhost:${vitePort}`,
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: [
    {
      command: `rm -f ../server/bonds.db ../server/bonds.db-shm ../server/bonds.db-wal && cd ../server && SERVER_PORT=${serverPort} ${webAuthnEnv} go run -ldflags="-X main.Version=e2e-test" cmd/server/main.go`,
      port: serverPort,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: `bun dev --host 0.0.0.0 --port ${vitePort}`,
      port: vitePort,
      reuseExistingServer: !process.env.CI,
    },
  ],
});
