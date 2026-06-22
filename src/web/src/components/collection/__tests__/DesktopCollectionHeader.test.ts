import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const headerPath = path.resolve(__dirname, '../DesktopCollectionHeader.vue')

describe('DesktopCollectionHeader', () => {
  it('does not launch Mint Map from the collection toolbar', () => {
    const source = fs.readFileSync(headerPath, 'utf8')

    expect(source).not.toContain('to="/mint-map"')
    expect(source).not.toContain('Mint Map')
    expect(source).not.toContain('MapPin')
  })

  it('keeps global collection actions out of the desktop command bar', () => {
    const source = fs.readFileSync(headerPath, 'utf8')

    expect(source).not.toContain('to="/add"')
    expect(source).not.toContain('toggle-select-mode')
    expect(source).not.toContain('CirclePlus')
    expect(source).not.toContain('CheckSquare')
    expect(source).toContain('face-toggle')
  })
})
