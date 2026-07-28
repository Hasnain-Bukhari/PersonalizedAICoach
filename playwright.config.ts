// playwright.config.ts

import { defineConfig, devices } from '@playwright/test'

const demoMode = process.env.E2E_DEMO_MODE === 'true'
const baseURL = process.env.E2E_BASE_URL ?? (demoMode ? 'http://127.0.0.1:5174' : 'http://127.0.0.1:5173')
const parsedBaseURL = new URL(baseURL)
if (!['127.0.0.1', 'localhost', '::1'].includes(parsedBaseURL.hostname)) {
  throw new Error(`E2E_BASE_URL must be loopback; refusing to test ${parsedBaseURL.hostname}`)
}

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  retries: 1,
  workers: 1,

  reporter: [
    ['list'],
    ['html', { outputFolder: process.env.PLAYWRIGHT_HTML_REPORT ?? 'artifacts/reports/playwright', open: 'never' }],
  ],

  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },

  outputDir: process.env.PLAYWRIGHT_OUTPUT_DIR ?? 'artifacts/playwright-results',

  webServer: demoMode
    ? {
        command: 'VITE_DEMO_MODE=true pnpm --filter frontend dev --host 127.0.0.1 --port 5174',
        url: 'http://127.0.0.1:5174',
        reuseExistingServer: false,
        timeout: 120_000,
      }
    : [
        {
          command: './scripts/dev-backend.sh',
          url: 'http://127.0.0.1:8080/readyz',
          reuseExistingServer: true,
          timeout: 120_000,
        },
        {
          command: 'VITE_DEV_TOKEN=dev:playwright pnpm --filter frontend dev --host 127.0.0.1',
          url: 'http://127.0.0.1:5173',
          reuseExistingServer: true,
          timeout: 120_000,
        },
      ],

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
