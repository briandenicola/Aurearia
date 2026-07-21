import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const appPath = path.resolve(__dirname, '../App.vue')

describe('App sidebar navigation', () => {
  it('renders Stats as a collapsible parent with dedicated route children', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    expect(source).toContain("label: 'Stats'")
    expect(source).toContain("label: 'Timeline', to: '/stats/timeline'")
    expect(source).toContain("label: 'Map', to: '/stats/mint-map'")
    expect(source).toContain("label: 'Health', to: '/stats/health'")
    expect(source).toContain("label: 'Value Details', to: '/stats/value-trends'")
    expect(source).not.toContain("id: 'stats-emperors'")
    expect(source).not.toContain("to: '/stats/emperors'")
    expect(source).not.toContain("id: 'timeline'")
    expect(source).not.toContain("label: 'Collection Distribution'")
    expect(source).not.toContain('#collection-health')
    expect(source).not.toContain('#value-over-time')
  })

  it('renders Sets as a collapsible parent with Emperors gated under it', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    expect(source).toContain("label: 'Sets'")
    expect(source).toContain("label: 'My Sets', to: '/sets'")
    expect(source).toContain("label: 'Emperors', to: '/sets/emperors'")
    expect(source).toContain("child.id !== 'sets-emperors' || auth.user?.emperorTrackerEnabled")
  })

  it('uses Identify Coin as the single merged quick capture entry point', () => {
    const source = fs.readFileSync(appPath, 'utf8')

    expect(source).not.toContain("id: 'quick-capture'")
    expect(source).not.toContain("label: 'Quick Capture'")
    expect(source).not.toContain("to: '/quick-capture'")
    expect(source).toContain("id: 'lookup'")
    expect(source).toContain("label: 'Identify Coin'")
    expect(source).toContain("to: '/lookup'")
    expect(source).toContain("id: 'add-coin'")
    expect(source).toContain("label: 'Add Coin'")
  })

  it('preserves AI intake identification navigation without adding Quick Capture AI expansion', () => {
    const source = fs.readFileSync(appPath, 'utf8')
    const routerSource = fs.readFileSync(path.resolve(__dirname, '../router/index.ts'), 'utf8')

    expect(source).toContain("id: 'lookup'")
    expect(source).toContain("label: 'Identify Coin'")
    expect(source).toContain("to: '/lookup'")
    expect(routerSource).toContain("path: '/lookup'")
    expect(routerSource).not.toContain('/quick-capture/intake')
    expect(routerSource).not.toContain('/quick-capture/ai')
  })
})
