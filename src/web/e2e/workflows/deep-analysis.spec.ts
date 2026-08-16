import { expect, test } from '@playwright/test'
import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

const tinyPng = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
  'base64',
)

test.beforeEach(async ({ page }) => {
  await installAuthenticatedSession(page)
})

test('starting Deep Analysis from new intake requires both faces and navigates to the job page', async ({ page }) => {
  await installWorkflowApiMocks(page)

  await page.goto('/lookup')
  await expect(page.getByRole('heading', { name: 'Identify Coin' })).toBeVisible()

  await page.getByRole('button', { name: 'Upload from library' }).click()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'obverse.png',
    mimeType: 'image/png',
    buffer: tinyPng,
  })
  await page.getByRole('button', { name: 'Deep Analysis' }).click()
  await expect(page.getByText('Add a reverse image before starting Deep Analysis.')).toBeVisible()
  await page.getByRole('button', { name: 'Upload from library' }).click()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'reverse.png',
    mimeType: 'image/png',
    buffer: tinyPng,
  })
  await page.getByRole('button', { name: 'Deep Analysis' }).click()
  await expect(page.getByRole('heading', { name: 'Deep Analysis' })).toBeVisible()
  await page.getByRole('button', { name: 'Start Deep Analysis' }).click()

  await expect(page).toHaveURL(/\/deep-analysis\/\d+$/)
  await expect(page.getByRole('heading', { name: /Job #\d+/ })).toBeVisible()
})

test('starting Deep Analysis from a saved coin reuses existing images and never writes to the coin', async ({ page }) => {
  const state = await installWorkflowApiMocks(page)
  const coinId = state.coins[0]!.id

  await page.goto(`/coin/${coinId}/actions`)
  await expect(page.getByRole('heading', { name: 'Actions' })).toBeVisible()

  await page.getByRole('button', { name: 'Deep Analysis' }).click()
  await expect(page.getByRole('heading', { name: 'Deep Analysis' })).toBeVisible()

  // Both faces already exist on the saved coin, so no upload inputs for
  // obverse/reverse are shown - submit is immediately available.
  await expect(page.getByText("Using this coin's existing obverse photo")).toBeVisible()
  await expect(page.getByText("Using this coin's existing reverse photo")).toBeVisible()

  await page.getByRole('button', { name: 'Start Deep Analysis' }).click()

  await expect(page).toHaveURL(/\/deep-analysis\/\d+$/)
  await expect(page.getByRole('heading', { name: /Job #\d+/ })).toBeVisible()

  expect(state.deepIdentificationJobs).toHaveLength(1)
  // Principle IV: Deep Analysis start must never directly write to the coin
  // via the existing update path, regardless of how many roles it reuses.
  expect(state.updatePayloads).toHaveLength(0)
})

test('T108: observes streamed progress and can cancel a running Deep Analysis job', async ({ page }) => {
  await installWorkflowApiMocks(page)

  await page.goto('/lookup')
  await page.getByRole('button', { name: 'Upload from library' }).click()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'obverse.png',
    mimeType: 'image/png',
    buffer: tinyPng,
  })
  await page.getByRole('button', { name: 'Add reverse image' }).click()
  await page.getByRole('button', { name: 'Upload from library' }).click()
  await page.locator('input[type="file"]').setInputFiles({
    name: 'reverse.png',
    mimeType: 'image/png',
    buffer: tinyPng,
  })
  await page.getByRole('button', { name: 'Deep Analysis' }).click()
  await page.getByRole('button', { name: 'Start Deep Analysis' }).click()
  await expect(page).toHaveURL(/\/deep-analysis\/\d+$/)

  // Streamed progress events (the mocked SSE fixture replays job_accepted,
  // a progress frame, then a terminal frame) are rendered in the timeline.
  await expect(page.getByText('Job accepted')).toBeVisible()
  await expect(page.getByText('Running providers')).toBeVisible()

  // The job is still shown non-terminal from the plain GET snapshot (the
  // fixture only flips status to cancelled once /cancel is called), so the
  // Cancel button is available; clicking it calls the cancel endpoint.
  const cancelButton = page.getByRole('button', { name: 'Cancel Deep Analysis' })
  await expect(cancelButton).toBeVisible()
  await cancelButton.click()

  await expect(page.getByLabel('Deep Analysis progress').getByText('cancelled')).toBeVisible()
})

test('T124: a partial-success terminal job shows an editable proposal that only applies to a new draft on explicit confirm', async ({ page }) => {
  const state = await installWorkflowApiMocks(page)
  const jobId = 6001
  state.deepIdentificationJobs.push({
    id: jobId,
    notes: '',
    providers: '',
    status: 'partial',
    source: 'intake',
    report: {
      schemaVersion: 1,
      narrative: 'Nomisma and Numista agree this is a Trajan denarius; NGC could not be automated.',
      coverage: [
        { provider: 'nomisma', status: 'contributed' },
        { provider: 'numista', status: 'contributed' },
        { provider: 'ngc', status: 'not_automated', note: 'Link out only' },
        { provider: 'ocre', status: 'not_automated' },
        { provider: 'rpc', status: 'unavailable' },
      ],
      partialSuccess: true,
      generatedAt: '2030-01-01T00:00:00Z',
    },
    proposal: {
      ruler: { proposed: 'Trajan', ownerEdited: false, ownerValue: null, accepted: null },
      denomination: { proposed: 'Denarius', ownerEdited: false, ownerValue: null, accepted: null },
    },
  })

  await page.goto(`/deep-analysis/${jobId}`)
  await expect(page.getByRole('heading', { name: `Job #${jobId}` })).toBeVisible()

  // Partial-success banner and coverage must remain visible alongside the
  // editable proposal - a partial result never hides provider status.
  await expect(page.getByText('Partial results')).toBeVisible()
  await expect(page.getByText('Not automated').first()).toBeVisible()
  await expect(page.getByText('Unavailable', { exact: true })).toBeVisible()

  const confirmButton = page.getByRole('button', { name: 'Save as Draft' })
  await expect(confirmButton).toBeDisabled()

  // Accept only the ruler field; the button stays disabled until a
  // decision is made, then becomes available.
  await page.locator('#deep-proposal-field-ruler').fill('Trajan (edited)')
  await page.getByRole('group', { name: /Ruler decision/ }).getByRole('button', { name: 'Accept' }).click()
  await expect(confirmButton).toBeEnabled()

  await confirmButton.click()

  expect(state.deepIdentificationApplies).toHaveLength(1)
  expect(state.deepIdentificationApplies[0]).toMatchObject({ id: jobId, target: 'draft' })
  await expect(page).toHaveURL('/quick-capture/drafts/8002')
})

test('T124: a completed terminal job for a saved coin applies proposal edits through the existing coin-update path only', async ({ page }) => {
  const state = await installWorkflowApiMocks(page)
  const coinId = state.coins[0]!.id
  const jobId = 6002
  state.deepIdentificationJobs.push({
    id: jobId,
    notes: '',
    providers: '',
    status: 'completed',
    source: 'saved_coin',
    report: {
      schemaVersion: 1,
      narrative: 'All automated providers agree on the attribution for this saved coin.',
      coverage: [
        { provider: 'nomisma', status: 'contributed' },
        { provider: 'numista', status: 'contributed' },
      ],
      partialSuccess: false,
      generatedAt: '2030-01-01T00:00:00Z',
    },
    proposal: {
      mint: { proposed: 'Rome', ownerEdited: false, ownerValue: null, accepted: null },
    },
  })

  await page.goto(`/deep-analysis/${jobId}`)
  await expect(page.getByRole('heading', { name: `Job #${jobId}` })).toBeVisible()

  await page.getByRole('group', { name: /Mint decision/ }).getByRole('button', { name: 'Accept' }).click()
  await page.getByRole('button', { name: 'Apply to Coin' }).click()

  expect(state.deepIdentificationApplies).toHaveLength(1)
  expect(state.deepIdentificationApplies[0]).toMatchObject({ id: jobId, target: 'coin' })
  // Principle IV: applying a Deep Analysis proposal must never bypass the
  // existing coin-update path with an ad-hoc write.
  expect(state.updatePayloads.filter((entry) => entry.id === coinId)).toHaveLength(0)
  await expect(page.getByText(/Applied to coin on/)).toBeVisible()
})
