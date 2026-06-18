import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const appPath = path.resolve(__dirname, '../App.vue')

describe('App sidebar navigation', () => {
  it('renders Stats as the parent for the clarified submenu only', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    expect(source).toContain("label: 'Stats'")
    expect(source).toContain("label: 'Timeline', to: '/stats/timeline'")
    expect(source).toContain("label: 'Map', to: '/stats/mint-map'")
    expect(source).toContain("label: 'Health', to: '/stats/distribution#collection-health'")
    expect(source).toContain("label: 'Value Trends', to: '/stats/distribution#value-over-time'")
    expect(source).not.toContain("id: 'timeline'")
    expect(source).not.toContain("label: 'Collection Distribution'")
  })
})
