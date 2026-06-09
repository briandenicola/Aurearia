import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const editCoinPagePath = path.resolve(__dirname, '../EditCoinPage.vue')

describe('EditCoinPage', () => {
  it('does not clear legacy or custom era values when loading an existing coin', () => {
    const source = fs.readFileSync(editCoinPagePath, 'utf8')

    expect(source).not.toContain('COIN_ERAS')
    expect(source).not.toContain('form.era = \'\'')
  })
})
