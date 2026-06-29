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
})
