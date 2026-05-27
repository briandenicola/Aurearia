import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const coinFormPath = path.resolve(__dirname, '../CoinForm.vue')

describe('CoinForm', () => {
  it('renders section titles inside the form sections with larger heading styles', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).toContain('<h2 class="form-section-title">Basic Information</h2>')
    expect(source).not.toContain('<legend>Basic Information</legend>')
    expect(source).toContain('.form-section-title {')
    expect(source).toContain('font-size: 1.2rem;')
    expect(source).toContain('margin: 0 0 1rem;')
  })

  it('allows purchase form fields to shrink within grid columns', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).toContain('<input v-model="form.purchaseDate" class="form-input" type="date" />')
    expect(source).toContain('.form-group {')
    expect(source).toContain('min-width: 0;')
  })
})
