import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const headerPath = path.resolve(__dirname, '../DesktopCollectionHeader.vue')

describe('DesktopCollectionHeader', () => {
  it('links to the authenticated mint map view', () => {
    const source = fs.readFileSync(headerPath, 'utf8')

    expect(source).toContain('to="/mint-map"')
    expect(source).toContain('Mint Map')
    expect(source).toContain('MapPin')
  })
})
