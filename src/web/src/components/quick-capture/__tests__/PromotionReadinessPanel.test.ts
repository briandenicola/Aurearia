import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../PromotionReadinessPanel.vue'), 'utf8')

describe('PromotionReadinessPanel', () => {
  it('provides explicit, retry-safe promotion controls and field guidance', () => {
    expect(source).toContain('promoteQuickCaptureDraft')
    expect(source).toContain('confirm')
    expect(source).toContain('fieldErrors.name')
    expect(source).toContain('alreadyPromoted')
    expect(source).toContain(':disabled="!confirmed || promoting"')
    expect(source).toContain('emit(\'promoted\'')
  })

  it('lets the collector promote to either collection or wishlist using the backend target contract', () => {
    expect(source).toContain("type PromotionTarget = 'collection' | 'wishlist'")
    expect(source).toContain('v-model="target"')
    expect(source).toContain('value="collection"')
    expect(source).toContain('value="wishlist"')
    expect(source).toContain('target: target.value')
    expect(source).toContain('fieldErrors.target')
    expect(source).toContain('Promote to ${destinationLabel}')
  })
})
