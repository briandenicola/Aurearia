import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const collectionPagePath = path.resolve(__dirname, '../CollectionPage.vue')

describe('CollectionPage', () => {
  it('anchors the PWA add button on the left side of the screen', () => {
    const source = fs.readFileSync(collectionPagePath, 'utf8')
    const addFabStyle = source.match(/\.add-fab\s*\{[\s\S]*?\n\}/)?.[0]

    expect(addFabStyle).toContain('left: 24px;')
    expect(addFabStyle).not.toContain('right: 88px;')
  })
})
