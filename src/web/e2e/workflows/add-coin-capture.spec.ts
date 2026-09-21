import { expect, test } from '@playwright/test'
import type { IntakeCommitRequest } from '../../src/types'
import { buildRomanDenariusCore } from '../../src/test/fixtures'
import { coinFormControl, installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

const tinyPng = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
  'base64',
)

for (const mode of ['pwa', 'desktop'] as const) {
  test(`${mode} Add Coin shares Identify capture layout and saves reviewed evidence`, async ({ page }, testInfo) => {
    await page.setViewportSize(mode === 'pwa' ? { width: 390, height: 844 } : { width: 1440, height: 900 })
    await installAuthenticatedSession(page)
    const api = await installWorkflowApiMocks(page)
    await page.addInitScript((standalone) => {
      Object.defineProperty(navigator, 'standalone', { value: standalone })
      Object.defineProperty(navigator, 'mediaDevices', {
        value: {
          getUserMedia: async () => {
            document.documentElement.dataset.cameraRequested = 'true'
            throw new DOMException('Permission denied by fixture', 'NotAllowedError')
          },
        },
      })
    }, mode === 'pwa')

    const drafts: string[] = []
    const commits: IntakeCommitRequest[] = []
    await page.route('**/api/coins/intake/draft', async route => {
      drafts.push(route.request().postDataBuffer()?.toString('latin1') ?? '')
      await route.fulfill({ json: {
        draftId: 701,
        coin: { name: 'Wizard Denarius', category: 'Roman', material: 'Silver', era: 'ancient' },
        confidenceSummary: { overall: 'medium' },
        unresolvedFields: [],
      } })
    })
    await page.route('**/api/coins/intake/commit', async route => {
      commits.push(route.request().postDataJSON() as IntakeCommitRequest)
      api.coins.push(buildRomanDenariusCore({ id: 7002, name: 'Wizard Denarius', images: [] }))
      await route.fulfill({ json: { coinId: 7002 } })
    })

    await page.goto('/lookup')
    await expect(page.getByRole('heading', { name: 'Add the obverse' })).toBeVisible()
    const identifyBounds = await page.locator('.capture-workspace').boundingBox()
    await page.screenshot({ path: testInfo.outputPath(`identify-${mode}.png`), fullPage: true })
    await page.goto('/add')
    if (mode === 'desktop') {
      await expect(page.getByRole('button', { name: 'Add to Collection' })).toBeVisible()
      await page.getByRole('button', { name: 'AI Assist Mode' }).click()
    }
    await expect(page.getByRole('heading', { name: 'Add the obverse' })).toBeVisible()
    await expect(page.getByRole('list', { name: 'Coin intake progress' }).locator('li')).toHaveText([
      /Obverse/, /Reverse/, /Card/,
    ])
    await expect(page.getByRole('button', { name: 'Add reverse image' })).toBeDisabled()
    await expect(page.locator('html')).not.toHaveAttribute('data-camera-requested', 'true')
    const addBounds = await page.locator('.capture-workspace').boundingBox()
    expect(addBounds?.x).toBe(identifyBounds?.x)
    expect(addBounds?.width).toBe(identifyBounds?.width)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    await page.screenshot({ path: testInfo.outputPath(`add-coin-${mode}.png`), fullPage: true })

    if (mode === 'pwa') {
      await page.getByRole('button', { name: 'Start Camera', exact: true }).click()
      await expect(page.getByText('Camera permission was denied. You can still upload images.')).toBeVisible()
    }
    const upload = page.getByRole('button', { name: mode === 'pwa' ? 'Upload from library' : 'Upload Image', exact: true })
    const obverseChooser = page.waitForEvent('filechooser')
    await upload.click()
    await (await obverseChooser).setFiles({ name: 'obverse.png', mimeType: 'image/png', buffer: tinyPng })
    await expect(page.getByAltText('Obverse coin image', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Generate Intake Draft' })).toBeEnabled()
    await page.getByRole('button', { name: 'Add reverse image' }).click()
    await page.getByRole('button', { name: 'Add coin card' }).click()
    await expect(page.getByRole('heading', { name: 'Add a coin card' })).toBeVisible()
    const cardChooser = page.waitForEvent('filechooser')
    await upload.click()
    await (await cardChooser).setFiles({ name: 'card.png', mimeType: 'image/png', buffer: tinyPng })
    await page.getByRole('button', { name: 'Generate Intake Draft' }).click()
    await expect(page.getByRole('heading', { name: 'Review Draft' })).toBeVisible()
    expect(drafts).toHaveLength(1)
    expect(drafts[0]?.match(/name="images";/g)).toHaveLength(1)
    expect(drafts[0]).toContain('name="images"; filename="obverse.jpg"')
    expect(drafts[0]).toContain('name="coinCardImage"; filename="card.jpg"')
    expect(commits).toHaveLength(0)

    await page.getByRole('button', { name: 'Confirm and Save Coin' }).click()
    await expect(page).toHaveURL('/coin/7002')
    await expect(page.getByRole('heading', { name: 'Wizard Denarius', exact: true })).toBeVisible()
    expect(commits).toEqual([expect.objectContaining({ draftId: 701, confirm: true })])
    expect(api.imageUploads).toEqual([expect.objectContaining({
      coinId: 7002, imageType: 'obverse', isPrimary: true, fileName: 'obverse.jpg',
    })])
    expect(api.createPayloads).toHaveLength(0)
  })
}

test('PWA manual bypass saves without calling AI intake', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await installAuthenticatedSession(page)
  await page.addInitScript(() => Object.defineProperty(navigator, 'standalone', { value: true }))
  const api = await installWorkflowApiMocks(page)
  const intakeRequests: string[] = []
  await page.route('**/api/coins/intake/**', async route => {
    intakeRequests.push(route.request().url())
    await route.abort()
  })

  await page.goto('/add')
  await page.getByRole('button', { name: 'Use manual mode instead' }).click()
  await coinFormControl(page, 'Name').fill('Manual PWA Coin')
  await page.getByRole('button', { name: 'Add to Collection' }).click()
  await expect(page).toHaveURL('/coin/7001')
  expect(api.createPayloads).toHaveLength(1)
  expect(intakeRequests).toHaveLength(0)
})
