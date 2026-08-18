/**
 * Required environment variables for the beta screenshot tour.
 *
 * These reference an existing beta account and MUST be supplied via the
 * environment only — never hardcoded, logged, or persisted to disk (no
 * storage state files under tracked paths).
 */
export interface ScreenshotCredentials {
  username: string
  password: string
}

export function requireScreenshotEnv(): ScreenshotCredentials {
  const username = process.env.AUREARIA_SCREENSHOT_USERNAME
  const password = process.env.AUREARIA_SCREENSHOT_PASSWORD

  const missing: string[] = []
  if (!username) missing.push('AUREARIA_SCREENSHOT_USERNAME')
  if (!password) missing.push('AUREARIA_SCREENSHOT_PASSWORD')

  if (missing.length > 0) {
    throw new Error(
      `Missing required environment variable(s) for the beta screenshot tour: ${missing.join(', ')}. ` +
        'Set them before running "npm run screenshots:beta" — see src/web/README.md for PowerShell examples. ' +
        'Credentials must reference an existing beta account and are never hardcoded or logged.',
    )
  }

  return { username: username as string, password: password as string }
}
