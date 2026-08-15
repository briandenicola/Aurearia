import { expect, test } from '@playwright/test'
import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

test.beforeEach(async ({ page }) => {
  await installAuthenticatedSession(page)
})

test('starting Deep Analysis from new intake requires both faces and navigates to the job page', async ({ page }) => {
  await installWorkflowApiMocks(page)

  await page.goto('/lookup')
  await expect(page.getByRole('heading', { name: 'Identify Coin' })).toBeVisible()

  await page.getByRole('button', { name: 'Deep Analysis' }).click()
  await expect(page.getByRole('heading', { name: 'Deep Analysis' })).toBeVisible()

  // Missing obverse/reverse blocks submit with a specific validation message.
  await page.getByRole('button', { name: 'Start Deep Analysis' }).click()
  await expect(page.getByText('Obverse and reverse photos are both required to start Deep Analysis.')).toBeVisible()

  const fileInputs = page.locator('input[type="file"]')
  await fileInputs.nth(1).setInputFiles({
    name: 'obverse.jpg',
    mimeType: 'image/jpeg',
    buffer: Buffer.from([0xff, 0xd8, 0xff, 0xd9]),
  })
  await fileInputs.nth(2).setInputFiles({
    name: 'reverse.jpg',
    mimeType: 'image/jpeg',
    buffer: Buffer.from([0xff, 0xd8, 0xff, 0xd9]),
  })

  await page.getByRole('button', { name: 'Start Deep Analysis' }).click()

  await expect(page).toHaveURL(/\/deep-analysis\/\d+$/)
  await expect(page.getByRole('heading', { name: /Job #\d+/ })).toBeVisible()
})
