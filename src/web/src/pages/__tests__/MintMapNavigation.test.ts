import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const srcRoot = path.resolve(__dirname, '..', '..')

describe('MintMap navigation entry points', () => {
  it('moves mint map and timeline under Stats with legacy redirects', () => {
    const routerSource = fs.readFileSync(path.resolve(srcRoot, 'router/index.ts'), 'utf8')

    expect(routerSource).toContain("path: '/stats/mint-map'")
    expect(routerSource).toContain("name: 'stats-mint-map'")
    expect(routerSource).toContain("path: '/stats/timeline'")
    expect(routerSource).toContain("name: 'stats-timeline'")
    expect(routerSource).toContain("path: '/stats/health'")
    expect(routerSource).toContain("name: 'stats-health'")
    expect(routerSource).toContain("path: '/stats/value-trends'")
    expect(routerSource).toContain("name: 'stats-value-trends'")
    expect(routerSource).toContain("path: '/stats/distribution'")
    expect(routerSource).toContain("name: 'stats-distribution'")
    expect(routerSource).toContain("path: '/mint-map'")
    expect(routerSource).toContain("redirect: '/stats/mint-map'")
    expect(routerSource).toContain("path: '/timeline'")
    expect(routerSource).toContain("redirect: '/stats/timeline'")
  })

  it('keeps stats subviews nested under the Stats sidebar item', () => {
    const appSource = fs.readFileSync(path.resolve(srcRoot, 'App.vue'), 'utf8')

    expect(appSource).toContain("id: 'stats'")
    expect(appSource).toContain("label: 'Stats'")
    expect(appSource).toContain("label: 'Timeline', to: '/stats/timeline'")
    expect(appSource).toContain("label: 'Map', to: '/stats/mint-map'")
    expect(appSource).toContain("label: 'Health', to: '/stats/health'")
    expect(appSource).toContain("label: 'Value Trends', to: '/stats/value-trends'")
    expect(appSource).not.toContain("id: 'timeline'")
    expect(appSource).not.toContain("label: 'Collection Distribution'")
    expect(appSource).not.toContain('#collection-health')
    expect(appSource).not.toContain('#value-over-time')
  })

  it('sidebar and overlay z-index stack above Leaflet map controls (fixes #294)', () => {
    // Leaflet controls sit at z-index ≤1000; MintCoinDrawer is at 1100.
    // sidebar-overlay must exceed both; sidebar must exceed the overlay.
    const appSource = fs.readFileSync(path.resolve(srcRoot, 'App.vue'), 'utf8')

    const overlayMatch = appSource.match(/\.sidebar-overlay\s*{[^}]*z-index:\s*(\d+)/s)
    const sidebarMatch = appSource.match(/\.sidebar\s*{[^}]*z-index:\s*(\d+)/s)

    expect(overlayMatch).not.toBeNull()
    expect(sidebarMatch).not.toBeNull()

    const overlayZ = parseInt(overlayMatch![1], 10)
    const sidebarZ = parseInt(sidebarMatch![1], 10)

    // Must clear Leaflet controls (≤1000) and MintCoinDrawer (1100)
    expect(overlayZ).toBeGreaterThan(1100)
    // Sidebar must be above its own overlay
    expect(sidebarZ).toBeGreaterThan(overlayZ)
  })
})
