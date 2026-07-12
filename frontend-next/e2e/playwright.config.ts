import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for LingMirror Next.js frontend E2E.
 *
 * Targets the Next.js dev server (port 3000) by default. Backend (Go)
 * is expected to be running on port 8080. See the main chain spec for
 * the full flow.
 *
 * Run: cd e2e && npx playwright install && npm run e2e
 */
export default defineConfig({
  testDir: './tests',
  // Historical mock suites assert the superseded AgentOS dashboard, Owner mock
  // console, and legacy product UI. Keep the files for traceability, but do not
  // let them define acceptance for the current Owner self-use business loop.
  testIgnore: [
    '**/login.spec.ts',
    '**/main-chain.spec.ts',
    '**/owner-approval.spec.ts',
    '**/products.spec.ts',
  ],
  fullyParallel: false, // E2E shares a DB; run serially to avoid data races
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [
    ['list'],
    ['html', { outputFolder: 'playwright-report', open: 'never' }],
  ],
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10000,
    navigationTimeout: 15000,
    extraHTTPHeaders: {
      'Content-Type': 'application/json',
    },
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  // Start Next dev server automatically when running locally.
  webServer: process.env.E2E_SKIP_WEB_SERVER
    ? undefined
    : {
        command: 'cd .. && npm run dev',
        url: 'http://localhost:3000',
        reuseExistingServer: !process.env.CI,
        timeout: 60_000,
      },
});
