import { request as pwRequest, type APIRequestContext, type Page } from '@playwright/test'

export interface AuthenticatedApiSession {
  api: APIRequestContext
  dispose: () => Promise<void>
}

/**
 * Logs in through the real `/api/auth/login` contract to obtain a bearer
 * token for fixture setup, then returns an APIRequestContext that sends it on
 * every request. Never logs the credentials or the token.
 */
export async function createAuthenticatedApiContext(baseURL: string, username: string, password: string): Promise<AuthenticatedApiSession> {
  const anonymous = await pwRequest.newContext({ baseURL })
  let token: string
  try {
    const res = await anonymous.post('/api/auth/login', { data: { username, password } })
    if (!res.ok()) {
      throw new Error(
        `Beta login failed (HTTP ${res.status()}). Verify AUREARIA_SCREENSHOT_USERNAME/AUREARIA_SCREENSHOT_PASSWORD ` +
          `are correct for ${baseURL}.`,
      )
    }
    const body = (await res.json()) as { token: string }
    token = body.token
  } finally {
    await anonymous.dispose()
  }

  const api = await pwRequest.newContext({ baseURL, extraHTTPHeaders: { Authorization: ['Bearer', token].join(' ') } })
  return { api, dispose: () => api.dispose() }
}

/**
 * Authenticates the browser session through the real login UI (not a mocked
 * or injected session) so the captured tour reflects the actual production
 * sign-in flow.
 */
export async function loginThroughUi(page: Page, username: string, password: string): Promise<void> {
  await page.goto('/login')
  await page.locator('input[autocomplete="username"]').fill(username)
  await page.locator('input[type="password"]').fill(password)
  await page.getByRole('button', { name: 'Sign In' }).click()
  await page.waitForURL((url) => url.pathname === '/', { timeout: 20_000 })
}
