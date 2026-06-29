import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const collectionPagePath = path.resolve(__dirname, '../CollectionPage.vue')

describe('CollectionPage', () => {
  it('does not include a floating add button in PWA mode', () => {
    const source = fs.readFileSync(collectionPagePath, 'utf8')
    expect(source).not.toContain('class="add-fab"')
    expect(source).not.toMatch(/\.add-fab\s*\{/)
  })

  it('continues to use the normal collection filters so draft rows stay excluded', () => {
    const source = fs.readFileSync(collectionPagePath, 'utf8')
    const filterSource = fs.readFileSync(path.resolve(__dirname, '../../composables/useCollectionFilters.ts'), 'utf8')

    expect(source).toContain('useCollectionFilters')
    expect(source).not.toContain('listQuickCaptureDrafts')
    expect(filterSource).toContain("wishlist: 'false'")
    expect(filterSource).toContain("sold: 'false'")
  })
})
