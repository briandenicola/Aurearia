import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e/exploration',
  outputDir: './.artifacts/ai-browser/playwright-output',
  workers: 1,
  fullyParallel: false,
  reporter: [['list']],
  use: {
    ...devices['Desktop Chrome'],
    browserName: 'chromium',
    baseURL: process.env.AI_BROWSER_APP_ORIGIN,
    // Automatic screenshots are disabled because they bypass the masked
    // evidence capture path added by the exploration runner.
    trace: 'off',
    video: 'off',
    screenshot: 'off',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
