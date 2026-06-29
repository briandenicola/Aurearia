import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../QuickCaptureDraftPage.vue'), 'utf8')

describe('QuickCaptureDraftPage', () => {
  it('loads a draft into an editable resume form and persists changed fields/images', () => {
    expect(source).toContain('getQuickCaptureDraft')
    expect(source).toContain('populateForm(res.data)')
    expect(source).toContain('updateQuickCaptureDraft')
    expect(source).toContain('workingTitle: workingTitle.value')
    expect(source).toContain('removeImageIds.value.size > 0')
    expect(source).toContain('QuickCaptureImageSlots')
    expect(source).toContain('Draft saved.')
  })

  it('surfaces validation/load errors and uses an explicit discard confirmation flow', () => {
    expect(source).toContain('Unable to load quick capture draft.')
    expect(source).toContain('Failed to save draft. Please try again.')
    expect(source).toContain('discardQuickCaptureDraft')
    expect(source).toContain('Discard this draft?')
    expect(source).toContain('Yes, discard')
  })

  it('preserves promotion integration and links terminal states without broad page coupling', () => {
    expect(source).toContain('PromotionReadinessPanel')
    expect(source).toContain('This draft was promoted to a coin.')
    expect(source).toContain('View Coin')
    expect(source).toContain('This draft has been discarded.')
    expect(source).toContain('router.push(`/coin/${coinId}`)')
  })
})
