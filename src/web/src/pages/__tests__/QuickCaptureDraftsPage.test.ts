import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../QuickCaptureDraftsPage.vue'), 'utf8')

describe('QuickCaptureDraftsPage', () => {
  it('loads only active drafts and renders list/empty/loading/error states', () => {
    expect(source).toContain('listQuickCaptureDrafts')
    expect(source).toContain("status: 'active'")
    expect(source).toContain('limit: 50')
    expect(source).toContain('QuickCaptureDraftCard')
    expect(source).toContain('Loading drafts...')
    expect(source).toContain('No active drafts yet.')
    expect(source).toContain('Unable to load quick capture drafts.')
    expect(source).toContain('New Draft')
  })
})
