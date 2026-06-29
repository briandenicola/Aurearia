import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const soldPagePath = path.resolve(__dirname, '../SoldPage.vue')

describe('SoldPage', () => {
  it('continues to fetch only sold normal coins and not quick-capture drafts', () => {
    const source = fs.readFileSync(soldPagePath, 'utf8')

    expect(source).toContain("sold: 'true'")
    expect(source).toContain('store.fetchCoins')
    expect(source).not.toContain('listQuickCaptureDrafts')
    expect(source).not.toContain('QuickCapture')
  })
})
