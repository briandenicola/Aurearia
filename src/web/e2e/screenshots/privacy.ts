import type { Locator, Page } from '@playwright/test'

/**
 * Prepares a page for a deterministic, privacy-safe screenshot:
 * - disables CSS animations/transitions and the text-input caret so frames
 *   are stable regardless of timing
 * - hides the notification unread-count badge, which is account-specific
 * - waits for web fonts and network activity to settle
 */
export async function preparePageForScreenshot(page: Page): Promise<void> {
  await page.addStyleTag({
    content: `
      *, *::before, *::after {
        animation-duration: 0s !important;
        animation-delay: 0s !important;
        transition-duration: 0s !important;
        transition-delay: 0s !important;
        scroll-behavior: auto !important;
        caret-color: transparent !important;
      }
      /* Account-specific unread notification count badge (App.vue) */
      nav a[aria-label="Notifications"] span {
        visibility: hidden !important;
      }
    `,
  })
  await page.waitForLoadState('networkidle')
  await page.evaluate(() => document.fonts?.ready).catch(() => undefined)
}

/**
 * Waits for the page/locator to settle (network + fonts) immediately before
 * capturing, then writes a deterministic PNG to `filePath`.
 */
export async function captureScreenshot(page: Page, filePath: string, locator?: Locator): Promise<void> {
  await page.waitForLoadState('networkidle')
  await page.evaluate(() => document.fonts?.ready).catch(() => undefined)
  const target = locator ?? page
  await target.screenshot({ path: filePath, animations: 'disabled' })
}
