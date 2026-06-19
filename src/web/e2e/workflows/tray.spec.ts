import { expect, test } from '@playwright/test'
import {
  installAuthenticatedSession,
  installWorkflowApiMocks,
} from '../fixtures/workflow'
import { buildRomanDenariusCore } from '../../src/test/fixtures'

test.beforeEach(async ({ page }) => {
  await installAuthenticatedSession(page)
})

test('desktop tray renders 67 measured coins through authenticated eager media', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  const measuredCoins = Array.from({ length: 67 }, (_, index) =>
    buildRomanDenariusCore({
      id: index + 1,
      name: `Measured Tray Coin ${index + 1}`,
      diameterMm: 16 + (index % 12),
    }),
  )
  const api = await installWorkflowApiMocks(page, measuredCoins)

  await page.goto('/tray')

  await expect(page.locator('.empty-state')).toBeHidden()
  await expect(page.getByText('Tray 1 of 6')).toBeVisible()
  await expect(page.locator('.tray-well')).toHaveCount(12)
  await expect.poll(async () => {
    const columns = await page.locator('.tray-grid').evaluate((grid) =>
      getComputedStyle(grid).gridTemplateColumns.split(' ').filter(Boolean).length,
    )
    return columns
  }).toBe(6)
  await expect(page.getByRole('button', { name: 'Measured Tray Coin 1', exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Measured Tray Coin 12', exact: true })).toBeVisible()
  const firstTrayImage = page.getByAltText('Measured Tray Coin 1', { exact: true })
  await expect(firstTrayImage).toBeVisible()
  await expect(firstTrayImage).toHaveAttribute('loading', 'eager')
  await expect(firstTrayImage).toHaveAttribute('decoding', 'async')
  await expect.poll(() => api.mediaRequests.length).toBeGreaterThan(0)
  expect(api.mediaRequests[0]).toMatchObject({
    authorization: 'Bearer workflow-access-token',
    cacheControl: 'no-store',
  })
  expect(api.coinQueries[0]).toMatchObject({
    wishlist: 'false',
    sold: 'false',
    limit: '100',
    page: '1',
  })

  await page.getByRole('button', { name: /Next/ }).click()
  await expect(page.getByText('Tray 2 of 6')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Measured Tray Coin 13', exact: true })).toBeVisible()
})
