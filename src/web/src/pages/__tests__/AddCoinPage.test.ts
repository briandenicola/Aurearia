import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const addCoinPagePath = path.resolve(__dirname, '../AddCoinPage.vue')

describe('AddCoinPage', () => {
  it('includes a Roman imperial figure selection in create and intake payloads only for Roman coins', () => {
    const source = fs.readFileSync(addCoinPagePath, 'utf8')

    expect(source).toContain("romanImperialFigureId: source.category === 'Roman' ? (source.romanImperialFigureId ?? null) : null")
    expect(source).toContain('overrides: buildCoinPayload(reviewForm)')
    expect(source).toContain('store.addCoin(buildCoinPayload(form))')
  })
})
