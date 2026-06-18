import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const pagePath = path.resolve(__dirname, '../CollectionDistributionPage.vue')

describe('CollectionDistributionPage', () => {
  it('owns distribution content and preserved stats sections', () => {
    const source = fs.readFileSync(pagePath, 'utf8')

    expect(source).toContain('StatsHeatMap')
    expect(source).toContain('StatsBarChart')
    expect(source).toContain('StatsValueOverTime')
    expect(source).toContain('id="collection-health"')
    expect(source).not.toContain('to="/mint-map"')
  })
})
