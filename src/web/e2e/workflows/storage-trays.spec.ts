import { expect, test, type Page } from '@playwright/test'
import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'
import {
  emptyThreeByThreeTray,
  missingImageTray,
  occupiedThreeByThreeTray,
  twentyByTwentyTray,
} from '../../src/test/fixtures/storageTrays'

test.beforeEach(async ({ page }) => {
  await installAuthenticatedSession(page)
  await page.addInitScript(() => {
    window.localStorage.setItem('tray:feltColor', 'navy')
  })
  await installWorkflowApiMocks(page)
})

async function mockTrayResponse(page: Page, trays: unknown[]) {
  await page.route('**/api/storage-trays', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ trays }),
  }))
}

test('desktop renders sparse and empty trays and supports pointer activation', async ({ page }) => {
  await mockTrayResponse(page, [occupiedThreeByThreeTray, { ...emptyThreeByThreeTray, id: 2, name: 'Empty Tray' }])
  await page.goto('/storage-trays')

  await expect(page.getByRole('heading', { name: 'Storage Trays' })).toBeVisible()
  const pageBounds = await page.locator('main.storage-trays-page').boundingBox()
  expect(pageBounds?.x ?? 0).toBeGreaterThan(0)
  const occupied = page.getByRole('region', { name: 'Cabinet A' })
  await expect(occupied.locator('.tray-surface')).toHaveClass(/felt-navy/)
  await expect(occupied.getByRole('grid', { name: 'Cabinet A, 3 rows by 3 columns, 1 of 9 occupied' })).toBeVisible()
  await expect(occupied.getByRole('gridcell')).toHaveCount(9)
  await expect(occupied.getByRole('button', { name: 'Denarius, row 2, column 3' })).toBeVisible()
  await expect(occupied.getByText('Denarius', { exact: true })).toBeVisible()
  const surfaceBounds = await occupied.locator('.physical-surface').boundingBox()
  const gridBounds = await occupied.getByRole('grid').boundingBox()
  const wellBounds = await occupied.getByRole('button', { name: 'Denarius, row 2, column 3' }).boundingBox()
  expect(Math.abs(
    ((surfaceBounds?.x ?? 0) + (surfaceBounds?.width ?? 0) / 2) -
    ((gridBounds?.x ?? 0) + (gridBounds?.width ?? 0) / 2)
  )).toBeLessThan(2)
  expect(wellBounds?.width).toBe(76)
  const titleBounds = await occupied.getByText('Denarius', { exact: true }).boundingBox()
  const emptyTitleBounds = await occupied.getByRole('gridcell').nth(3).locator('.tray-name').boundingBox()
  expect(Math.abs((titleBounds?.y ?? 0) - (emptyTitleBounds?.y ?? 0))).toBeLessThan(1)

  const faceToggle = occupied.locator('.tray-face-toggle')
  await expect(faceToggle).toHaveAccessibleName('Show reverse side for all coins')
  const initialImageSrc = await occupied.getByRole('button', { name: 'Denarius, row 2, column 3' }).locator('img').getAttribute('src')
  await faceToggle.click()
  await expect(faceToggle).toHaveAttribute('aria-pressed', 'true')
  await expect.poll(async () =>
    occupied.getByRole('button', { name: 'Denarius, row 2, column 3' }).locator('img').getAttribute('src')
  ).not.toBe(initialImageSrc)
  const empty = page.getByRole('region', { name: 'Empty Tray' })
  await expect(empty.getByRole('gridcell')).toHaveCount(9)
  await expect(empty.getByRole('button')).toHaveCount(0)

  await occupied.getByRole('button', { name: 'Denarius, row 2, column 3' }).click()
  await expect(page).toHaveURL('/coin/42')
})

test('mobile preserves 20-column horizontal scrolling and Enter activation', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await mockTrayResponse(page, [occupiedThreeByThreeTray, twentyByTwentyTray])
  await page.goto('/storage-trays')

  const largeTray = page.getByRole('region', { name: 'Large Tray' })
  await expect(largeTray.getByRole('gridcell')).toHaveCount(400)
  const scroller = largeTray.getByLabel('Large Tray, 20 columns')
  await expect.poll(() => scroller.evaluate(element => element.scrollWidth > element.clientWidth)).toBe(true)

  const coin = page.getByRole('button', { name: 'Denarius, row 2, column 3' })
  await coin.focus()
  await coin.press('Enter')
  await expect(page).toHaveURL('/coin/42')
})

test('Space activates an occupied well and missing images keep an occupied fallback', async ({ page }) => {
  await mockTrayResponse(page, [missingImageTray])
  await page.goto('/storage-trays')

  const missingImageCoin = page.getByRole('button', { name: 'Image Missing, row 1, column 1' })
  await expect(missingImageCoin.locator('img')).toHaveCount(0)
  await expect(missingImageCoin.locator('svg')).toBeVisible()
  await missingImageCoin.focus()
  await missingImageCoin.press('Space')
  await expect(page).toHaveURL('/coin/43')
})

test('aggregate failure shows a recoverable error and retry reloads trays', async ({ page }) => {
  let attempts = 0
  await page.route('**/api/storage-trays', async route => {
    attempts++
    if (attempts === 1) {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'temporary failure' }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ trays: [emptyThreeByThreeTray] }),
    })
  })
  await page.goto('/storage-trays')

  await expect(page.getByRole('alert')).toContainText('Storage trays could not be loaded.')
  await page.getByRole('button', { name: 'Retry' }).click()
  await expect(page.getByRole('region', { name: 'Cabinet A' })).toBeVisible()
  expect(attempts).toBe(2)
})
