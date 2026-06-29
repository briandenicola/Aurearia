import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../QuickCapturePage.vue'), 'utf8')

describe('QuickCapturePage', () => {
  it('composes the MVP save workflow and count-exclusion message', () => {
    expect(source).toContain('QuickCaptureForm')
    expect(source).toContain('Draft saved')
    expect(source).toContain('excluded from collection, wishlist, sold, stats, and health counts')
  })

  it('keeps Quick Capture v1 manual and navigates to drafts without AI intake expansion', () => {
    expect(source).toContain('to="/quick-capture/drafts"')
    expect(source).toContain('aria-label="All captures"')
    expect(source).toContain('List')
    expect(source).toContain('Capture sparse coin details quickly')
    expect(source).not.toContain('createIntakeDraft')
    expect(source).not.toContain('lookupCoin')
    expect(source).not.toContain('agent')
  })
})
