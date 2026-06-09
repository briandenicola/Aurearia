import { expect, test } from '@playwright/test'
import {
  expectAuthenticatedApiCall,
  installAuthenticatedSession,
  installWorkflowApiMocks,
  workflowUser,
} from '../fixtures/workflow'

test('login stores the authenticated session and opens the collection', async ({ page }) => {
  await installWorkflowApiMocks(page)

  await page.goto('/login')
  await page.getByRole('textbox').first().fill(workflowUser.username)
  await page.locator('input[type="password"]').fill('correct-password')
  await page.getByRole('button', { name: 'Sign In' }).click()

  await expect(page).toHaveURL('/')
  await expect(page.getByRole('button', { name: 'Coin Collection' })).toBeVisible()
  await expect(page.evaluate(() => window.localStorage.getItem('token'))).resolves.toBe('workflow-access-token')
})

test('authenticated setup helper opens protected workflows without a live backend', async ({ page }) => {
  await installAuthenticatedSession(page)
  const api = await installWorkflowApiMocks(page)

  await page.goto('/add')

  await expect(page.getByRole('heading', { name: 'Add Coin' })).toBeVisible()
  await expectAuthenticatedApiCall(api, 'GET /storage-locations')
})
