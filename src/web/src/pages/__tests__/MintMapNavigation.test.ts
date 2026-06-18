import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const srcRoot = path.resolve(__dirname, '..', '..')

describe('MintMap navigation entry points', () => {
  it('registers an authenticated mint map route', () => {
    const routerSource = fs.readFileSync(path.resolve(srcRoot, 'router/index.ts'), 'utf8')

    expect(routerSource).toContain("path: '/mint-map'")
    expect(routerSource).toContain("name: 'mint-map'")
    expect(routerSource).toContain("component: () => import('@/pages/MintMapPage.vue')")
    expect(routerSource).toContain('meta: { requiresAuth: true }')
  })

  it('links to the mint map from stats near collection distribution surfaces', () => {
    const statsSource = fs.readFileSync(path.resolve(srcRoot, 'pages/StatsPage.vue'), 'utf8')

    expect(statsSource).toContain('Collection Geography')
    expect(statsSource).toContain('to="/mint-map"')
    expect(statsSource).toContain('Open Mint Map')
  })
})
