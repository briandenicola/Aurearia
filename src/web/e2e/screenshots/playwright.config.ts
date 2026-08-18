import { defineConfig, devices } from '@playwright/test'

// Dedicated config for the beta screenshot tour (see README in this folder).
// This intentionally does NOT extend the deterministic src/web/playwright.config.ts
// webServer behavior: it always targets a real, already-running site (local or
// beta) via PLAYWRIGHT_BASE_URL and never starts/reuses a local dev server.
const LOCAL_BASE_URL = 'http://127.0.0.1:4173'
const baseURL = process.env.PLAYWRIGHT_BASE_URL || LOCAL_BASE_URL

export default defineConfig({
  testDir: '.',
  fullyParallel: false,
  timeout: 120_000,
  reporter: [['list']],
  outputDir: '../../test-results/screenshots',
  use: {
    baseURL,
    serviceWorkers: 'block',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      // Production-like desktop viewport for prospective-user-facing captures.
      name: 'desktop',
      use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 1000 } },
    },
    {
      // Mobile / PWA-like viewport (iPhone 13 dimensions).
      name: 'mobile',
      use: { ...devices['iPhone 13'], viewport: { width: 390, height: 844 } },
    },
  ],
})
