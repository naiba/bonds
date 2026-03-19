import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  // E2E 共享同一个 Go 后端 + SQLite DB，多 worker 并行会导致数据污染（admin 首用户竞争、联系人表串扰等）
  workers: 1,
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
      command: 'rm -f ../server/bonds.db ../server/bonds.db-shm ../server/bonds.db-wal && cd ../server && go run -ldflags="-X main.Version=e2e-test" cmd/server/main.go',
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
