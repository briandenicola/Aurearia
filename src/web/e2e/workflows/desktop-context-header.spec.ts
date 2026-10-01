import { expect, test } from '@playwright/test'
import { buildRomanDenariusCore } from '../../src/test/fixtures'
import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

test('switches between local and shared headers at the 768/769px boundary', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 768, height: 900 })
  await installAuthenticatedSession(page)
  await installWorkflowApiMocks(page)
  await page.goto('/add')

  const localHeader = page.locator('header.page-header')
  const sharedTitle = page.locator('#desktop-page-title')
  await expect(localHeader.getByRole('heading', { name: 'Add Coin' })).toBeVisible()
  await expect(sharedTitle).toBeHidden()
  await page.screenshot({ path: testInfo.outputPath('add-coin-768-local-header.png'), fullPage: true })

  await page.setViewportSize({ width: 769, height: 900 })
  await expect(localHeader).toBeHidden()
  await expect(sharedTitle).toBeVisible()
  await expect(sharedTitle).toHaveText('Add Coin')
  await expect(page.locator('#desktop-page-actions').getByRole('button', { name: 'AI Assist Mode' })).toBeVisible()
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.screenshot({ path: testInfo.outputPath('add-coin-769-shared-header.png'), fullPage: true })
})

test('installed PWA keeps its local header at desktop width', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1200, height: 900 })
  await installAuthenticatedSession(page)
  await page.addInitScript(() => {
    Object.defineProperty(navigator, 'standalone', { value: true })
  })
  await installWorkflowApiMocks(page)
  await page.goto('/add')

  await expect(page.locator('header.page-header').getByRole('heading', { name: 'Add Coin' })).toBeVisible()
  await expect(page.locator('#desktop-page-title')).toHaveCount(0)
  await expect(page.getByRole('heading', { name: 'Add the obverse' })).toBeVisible()
  await page.screenshot({ path: testInfo.outputPath('add-coin-installed-pwa.png'), fullPage: true })
})

test('crowded Wishlist actions stay ahead of global actions without overflow', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 1024, height: 900 })
  await installAuthenticatedSession(page)
  await installWorkflowApiMocks(page, [
    buildRomanDenariusCore({ id: 81, name: 'Wishlist Denarius', isWishlist: true }),
  ])
  await page.goto('/wishlist')

  const title = page.locator('#desktop-page-title')
  const actions = page.locator('#desktop-page-actions')
  const quickAccess = page.getByRole('link', { name: 'Quick Access' })
  await expect(title).toHaveText('Wishlist')
  await expect(actions.getByRole('button').or(actions.getByRole('link'))).toHaveCount(4)

  const titleBounds = await title.boundingBox()
  const actionBounds = await actions.boundingBox()
  const quickAccessBounds = await quickAccess.boundingBox()
  expect(titleBounds).not.toBeNull()
  expect(actionBounds).not.toBeNull()
  expect(quickAccessBounds).not.toBeNull()
  if (!titleBounds || !actionBounds || !quickAccessBounds) throw new Error('Desktop header controls are not visible')
  expect(titleBounds.x + titleBounds.width).toBeLessThanOrEqual(actionBounds.x)
  expect(actionBounds.x + actionBounds.width).toBeLessThanOrEqual(quickAccessBounds.x)
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.screenshot({ path: testInfo.outputPath('wishlist-crowded-actions.png'), fullPage: true })
})
