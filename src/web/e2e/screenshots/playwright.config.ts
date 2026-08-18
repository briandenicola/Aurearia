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
  // Force a single worker: `fullyParallel: false` only serializes tests
  // within a file, but the desktop/mobile projects below still ran as
  // separate parallel workers by default. Both projects call
  // ensureScreenshotFixtures() independently, and each worker's
  // GET-then-POST fixture lookup is not atomic — with 2 workers, both can
  // GET (find nothing) before either POST, creating duplicate
  // `[Screenshot]`-prefixed coins with the exact same name. Capping workers
  // at 1 makes fixture seeding fully sequential across projects.
  workers: 1,
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
      // Mobile / PWA-like viewport (iPhone 13 dimensions), rendered with
      // Chromium instead of the default WebKit engine so the screenshot
      // tour only needs `npx playwright install chromium` (no WebKit
      // download required). We keep the iPhone 13 device's viewport,
      // touch/mobile emulation flags, and UA string, but override
      // `defaultBrowserType` back to chromium.
      name: 'mobile',
      use: {
        ...devices['iPhone 13'],
        defaultBrowserType: 'chromium',
        viewport: { width: 390, height: 844 },
      },
    },
  ],
})
