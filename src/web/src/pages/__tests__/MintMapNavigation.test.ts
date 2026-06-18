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
    expect(routerSource).toContain("path: '/stats/distribution'")
    expect(routerSource).toContain("name: 'stats-distribution'")
    expect(routerSource).toContain("path: '/mint-map'")
    expect(routerSource).toContain("redirect: '/stats/mint-map'")
    expect(routerSource).toContain("path: '/timeline'")
    expect(routerSource).toContain("redirect: '/stats/timeline'")
  })

  it('links to stats subviews from the Stats landing page', () => {
    const statsSource = fs.readFileSync(path.resolve(srcRoot, 'pages/StatsPage.vue'), 'utf8')

    expect(statsSource).toContain('Collection Geography')
    expect(statsSource).toContain("to: '/stats/mint-map'")
    expect(statsSource).toContain("to: '/stats/timeline'")
    expect(statsSource).toContain("to: '/stats/distribution'")
  })
})
