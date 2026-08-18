import path from 'node:path'
import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { test, expect } from '@playwright/test'
import { requireScreenshotEnv } from './env'
import { ensureScreenshotFixtures, SCREENSHOT_PREFIX } from './fixtures'
import { createAuthenticatedApiContext, loginThroughUi } from './auth'
import { preparePageForScreenshot, captureScreenshot } from './privacy'

const dirname = path.dirname(fileURLToPath(import.meta.url))

// Deterministic output directory reviewed by Brian before anything is
// published. Ignored by git — see .gitignore (src/web/artifacts/).
const OUTPUT_DIR = path.resolve(dirname, '../../artifacts/screenshots')

test.beforeAll(() => {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true })
})

test('captures a production-like UX tour of the beta site', async ({ page, baseURL }, testInfo) => {
  test.setTimeout(120_000)

  // Fails fast with a clear message if credentials are missing — never
  // hardcoded, logged, or written to disk.
  const { username, password } = requireScreenshotEnv()
  if (!baseURL) {
    throw new Error('No base URL resolved. Set PLAYWRIGHT_BASE_URL before running "npm run screenshots:beta".')
  }

  // 1. Seed/refresh the [Screenshot]-prefixed fixtures via the real API
  //    contract. Idempotent: matches by exact name, never duplicates, never
  //    touches unrelated coins.
  const apiSession = await createAuthenticatedApiContext(baseURL, username, password)
  const fixtures = await ensureScreenshotFixtures(apiSession.api)
  await apiSession.dispose()

  const ancientCoin = fixtures[0]!
  const wishlistCoin = fixtures[2]!

  // 2. Authenticate the capture session through the real login UI.
  await loginThroughUi(page, username, password)
  await preparePageForScreenshot(page)

  const variant = testInfo.project.name // 'desktop' | 'mobile'
  const shotPath = (name: string) => path.join(OUTPUT_DIR, `${variant}-${name}.png`)

  // 3. Collection gallery — filtered to screenshot fixtures via the real
  //    search UI so only fixture coins appear in the captured frame.
  await page.goto('/')
  await page.getByPlaceholder('Search coins by name, ruler, inscription...').fill(SCREENSHOT_PREFIX)
  await expect(page.locator('.coins-grid').getByText(ancientCoin.name)).toBeVisible()
  await expect(page.locator('.coins-grid')).not.toContainText('$')
  await captureScreenshot(page, shotPath('01-collection-gallery'), page.locator('.coins-grid'))

  // 4. Coin detail — one representative (ancient collection) coin.
  await page.goto(`/coin/${ancientCoin.id}`)
  await expect(page.getByRole('heading', { name: ancientCoin.name })).toBeVisible()
  await captureScreenshot(page, shotPath('02-coin-detail'))

  // 5. Wishlist — the wishlist page has no search/filter UI, so the capture
  //    is scoped to just the matching fixture's card to guarantee no other
  //    (possibly real, personal) wishlist items are ever in frame.
  await page.goto('/wishlist')
  const wishlistCard = page.locator('.card').filter({ hasText: wishlistCoin.name })
  await expect(wishlistCard).toBeVisible()
  await captureScreenshot(page, shotPath('03-wishlist'), wishlistCard)

  // 6. Stats — aggregate collection insights (counts/charts only, no raw
  //    per-coin notes/prices), works against whatever data already exists.
  await page.goto('/stats')
  await expect(page.getByRole('heading', { name: 'Stats' })).toBeVisible()
  await captureScreenshot(page, shotPath('04-stats-overview'))

  // 7. Deep Analysis entry surface — the Actions panel exposes the "Deep
  //    Analysis" entry point without ever starting a job or invoking AI.
  await page.goto(`/coin/${ancientCoin.id}/actions`)
  await expect(page.getByRole('heading', { name: 'Actions' })).toBeVisible()
  await captureScreenshot(page, shotPath('05-deep-analysis-entry'))
})
