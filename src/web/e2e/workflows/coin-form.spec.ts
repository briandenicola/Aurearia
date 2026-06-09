import { expect, test } from '@playwright/test'
import {
  coinFormControl,
  installAuthenticatedSession,
  installWorkflowApiMocks,
} from '../fixtures/workflow'
import {
  buildGoldenCoinFixtures,
  buildImageHeavyDrachm,
  buildRomanDenariusCore,
  buildTaggedFollisStorage,
  buildTestTags,
} from '../../src/test/fixtures'

test.beforeEach(async ({ page }) => {
  await installAuthenticatedSession(page)
})

test('manual add coin saves deterministic fixture-shaped data', async ({ page }) => {
  const api = await installWorkflowApiMocks(page)

  await page.goto('/add')
  await expect(page.getByRole('heading', { name: 'Add Coin' })).toBeVisible()

  await coinFormControl(page, 'Name').fill('Browser Workflow Denarius')
  await coinFormControl(page, 'Denomination').fill('Denarius')
  await coinFormControl(page, 'Ruler').fill('Trajan')
  await coinFormControl(page, 'Mint').fill('Rome')
  await coinFormControl(page, 'Weight').fill('3.41')
  await coinFormControl(page, 'Purchase Price').fill('180')
  await page.getByRole('button', { name: 'Add to Collection' }).click()

  await expect(page).toHaveURL(/\/coin\/7001$/)
  await expect(page.getByRole('heading', { name: 'Browser Workflow Denarius' })).toBeVisible()
  expect(api.createPayloads).toHaveLength(1)
  expect(api.createPayloads[0]).toMatchObject({
    name: 'Browser Workflow Denarius',
    denomination: 'Denarius',
    ruler: 'Trajan',
    mint: 'Rome',
    weightGrams: 3.41,
    purchasePrice: 180,
  })
})

test('edit one field preserves the loaded golden fixture workflow', async ({ page }) => {
  const coin = buildRomanDenariusCore()
  const api = await installWorkflowApiMocks(page, [coin])

  await page.goto(`/coin/${coin.id}`)
  await page.getByRole('link', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Edit Coin' })).toBeVisible()

  await coinFormControl(page, 'Name').fill('Trajan Denarius Retitled')
  await page.getByRole('button', { name: 'Save Changes' }).click()

  await expect(page).toHaveURL(`/coin/${coin.id}`)
  await expect(page.getByRole('heading', { name: 'Trajan Denarius Retitled' })).toBeVisible()
  expect(api.updatePayloads).toHaveLength(1)
  expect(api.updatePayloads[0]?.payload).toMatchObject({
    id: coin.id,
    name: 'Trajan Denarius Retitled',
    tags: coin.tags,
    sets: coin.sets,
  })
})

test('edit storage location changes and clears the golden fixture location', async ({ page }) => {
  const coin = buildTaggedFollisStorage()
  const api = await installWorkflowApiMocks(page, [coin])
  const vaultBox = api.storageLocations.find((location) => location.name === 'Vault Box 2')
  if (!vaultBox) throw new Error('Vault Box 2 fixture is required for the storage workflow')

  await page.goto(`/coin/${coin.id}`)
  await page.getByRole('link', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Edit Coin' })).toBeVisible()

  await coinFormControl(page, 'Storage Location').selectOption(String(vaultBox.id))
  await page.getByRole('button', { name: 'Save Changes' }).click()

  await expect(page).toHaveURL(`/coin/${coin.id}`)
  await expect(page.locator('.metadata-row').filter({ hasText: 'Storage Location' })).toContainText(vaultBox.name)

  await page.getByRole('link', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Edit Coin' })).toBeVisible()

  await coinFormControl(page, 'Storage Location').selectOption('')
  await page.getByRole('button', { name: 'Save Changes' }).click()

  await expect(page).toHaveURL(`/coin/${coin.id}`)
  await expect(page.locator('.metadata-row').filter({ hasText: 'Storage Location' })).toContainText('—')
  expect(api.updatePayloads.map(({ payload }) => payload.storageLocationId)).toEqual([vaultBox.id, null])
})

test('edit tags and sets updates detail-page associations deterministically', async ({ page }) => {
  const photographed = buildTestTags().find((tag) => tag.name === 'Photographed')
  if (!photographed) throw new Error('Photographed fixture tag is required for the tags workflow')
  const coin = buildTaggedFollisStorage({ tags: [photographed], sets: [] })
  const api = await installWorkflowApiMocks(page, [coin])
  const needsResearch = api.tags.find((tag) => tag.name === 'Needs Research')
  const twelveCaesars = api.sets.find((set) => set.name === 'Twelve Caesars')
  if (!needsResearch || !twelveCaesars) throw new Error('Tag and set fixtures are required for the association workflow')

  await page.goto(`/coin/${coin.id}`)
  const tagsSection = page.locator('section').filter({ has: page.getByRole('heading', { name: 'Tags & Sets' }) })
  await expect(tagsSection.getByText('Photographed')).toBeVisible()

  await tagsSection.getByRole('button', { name: 'Remove Photographed tag' }).click()
  await expect(tagsSection.getByText('Photographed')).toBeHidden()

  await tagsSection.getByRole('button', { name: '+ Tag or Set' }).click()
  await tagsSection.locator('select').selectOption(`tag:${needsResearch.id}`)
  await expect(tagsSection.getByText('Needs Research')).toBeVisible()

  await tagsSection.getByRole('button', { name: '+ Tag or Set' }).click()
  await tagsSection.locator('select').selectOption(`set:${twelveCaesars.id}`)
  await expect(tagsSection.getByText('Twelve Caesars')).toBeVisible()

  await tagsSection.getByRole('button', { name: 'Remove Twelve Caesars set' }).click()
  await expect(tagsSection.getByText('Twelve Caesars')).toBeHidden()
  expect(api.tagPayloads).toEqual([
    { action: 'remove', coinId: coin.id, tagId: photographed.id },
    { action: 'add', coinId: coin.id, tagId: needsResearch.id },
  ])
  expect(api.setPayloads).toEqual([
    { action: 'add', coinId: coin.id, setId: twelveCaesars.id },
    { action: 'remove', coinId: coin.id, setId: twelveCaesars.id },
  ])
})

test('upload and delete image workflow updates deterministic image routes', async ({ page }) => {
  const coin = buildImageHeavyDrachm()
  const api = await installWorkflowApiMocks(page, [coin])
  const obverse = coin.images.find((image) => image.imageType === 'obverse')
  const reverse = coin.images.find((image) => image.imageType === 'reverse')
  if (!obverse || !reverse) throw new Error('Image-heavy fixture must include obverse and reverse images')

  await page.goto(`/coin/${coin.id}`)
  await page.getByRole('link', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Edit Coin' })).toBeVisible()

  await page.getByLabel('Remove obverse image').click()
  await page.getByLabel('Upload reverse image').setInputFiles({
    name: 'workflow-reverse.png',
    mimeType: 'image/png',
    buffer: Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
  })
  await page.getByRole('button', { name: 'Save Changes' }).click()

  await expect(page).toHaveURL(`/coin/${coin.id}`)
  await expect.poll(() => api.imageDeletes).toEqual([
    { coinId: coin.id, imageId: obverse.id },
    { coinId: coin.id, imageId: reverse.id },
  ])
  expect(api.imageUploads).toEqual([
    expect.objectContaining({
      coinId: coin.id,
      imageType: 'reverse',
      isPrimary: false,
      fileName: 'workflow-reverse.png',
    }),
  ])
  await expect(page.locator('.hero-slot').first().getByText('No image')).toBeVisible()
  await expect(page.getByAltText('Reverse')).toBeVisible()
})

test('collection search and filters query deterministic fixture data', async ({ page }) => {
  const api = await installWorkflowApiMocks(page, buildGoldenCoinFixtures())

  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Trajan Denarius Core' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Syracuse Drachm Image Heavy' })).toBeVisible()

  await page.getByPlaceholder('Search coins by name, ruler, inscription...').fill('Syracuse')
  await expect(page.getByRole('heading', { name: 'Syracuse Drachm Image Heavy' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Trajan Denarius Core' })).toBeHidden()
  await expect.poll(() => api.coinQueries.some((query) => query.search === 'Syracuse')).toBe(true)

  await page.getByPlaceholder('Search coins by name, ruler, inscription...').fill('')
  await page.getByRole('button', { name: 'Greek' }).click()
  await expect(page.getByRole('heading', { name: 'Athens Owl Tetradrachm Valued' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Trajan Denarius Core' })).toBeHidden()
  await expect.poll(() => api.coinQueries.some((query) => query.category === 'Greek')).toBe(true)

  await page.getByRole('button', { name: 'All', exact: true }).click()
  await page.locator('.tag-filter-select').selectOption({ label: 'Needs Research' })
  await expect(page.getByRole('heading', { name: 'Diocletian Follis Tagged Storage' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Athens Owl Tetradrachm Valued' })).toBeHidden()
  await expect.poll(() => api.coinQueries.some((query) => query.tag === '302')).toBe(true)
})

test('mobile viewport edit workflow saves without desktop-only controls', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  const coin = buildRomanDenariusCore()
  const api = await installWorkflowApiMocks(page, [coin])

  await page.goto(`/coin/${coin.id}`)
  await page.getByRole('link', { name: 'Edit' }).click()
  await expect(page.getByRole('heading', { name: 'Edit Coin' })).toBeVisible()

  await coinFormControl(page, 'Name').fill('Mobile Edited Denarius')
  await page.getByRole('button', { name: 'Save Changes' }).click()

  await expect(page).toHaveURL(`/coin/${coin.id}`)
  await expect(page.getByRole('heading', { name: 'Mobile Edited Denarius' })).toBeVisible()
  expect(api.updatePayloads).toHaveLength(1)
  expect(api.updatePayloads[0]?.payload).toMatchObject({
    id: coin.id,
    name: 'Mobile Edited Denarius',
  })
})
