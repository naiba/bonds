import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:5173',
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
      command: 'rm -f ../server/bonds.db ../server/bonds.db-shm ../server/bonds.db-wal && cd ../server && go run cmd/server/main.go',
      port: 8080,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: 'bun dev',
      port: 5173,
      reuseExistingServer: !process.env.CI,
    },
  ],
});
