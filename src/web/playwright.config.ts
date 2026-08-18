import { defineConfig, devices } from '@playwright/test'

// Deterministic local smoke suite (F013) targets a local Vite dev server by
// default. Setting PLAYWRIGHT_BASE_URL points the suite at any other
// reachable environment (e.g. the beta site) instead, and skips starting the
// local webServer since it would be pointless and would fight for port 4173.
const LOCAL_BASE_URL = 'http://127.0.0.1:4173'
const externalBaseURL = process.env.PLAYWRIGHT_BASE_URL
const baseURL = externalBaseURL || LOCAL_BASE_URL

export default defineConfig({
  testDir: './e2e',
  // The screenshot tour is a separate, deliberately-real-network tool (see
  // e2e/screenshots/) with its own config and `npm run screenshots:beta`
  // script. It must never run as part of the deterministic `test:browser`
  // suite, which mocks the API and requires no credentials.
  testIgnore: ['**/screenshots/**'],
  fullyParallel: false,
  reporter: [['list']],
  use: {
    baseURL,
    serviceWorkers: 'block',
    trace: 'retain-on-failure',
  },
  webServer: externalBaseURL
    ? undefined
    : {
        command: 'npm.cmd run dev -- --host 127.0.0.1 --port 4173',
        url: LOCAL_BASE_URL,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
      },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
})
