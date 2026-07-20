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
    const mainCssPath = path.resolve(__dirname, '../../assets/styles/main.css')
    const mainCss = fs.readFileSync(mainCssPath, 'utf8')

    // Section titles are <h2>s (not <legend>s) styled with the Tailwind
    // `text-lg` utility instead of a dedicated `.form-section-title` class.
    expect(source).toContain('<h2 class="mb-4 font-display text-lg font-medium text-gold">Basic Information</h2>')
    expect(source).not.toContain('<legend>Basic Information</legend>')
    // `text-lg` resolves to the same 1.2rem the old CSS block hard-coded, and
    // `mb-4` is the Tailwind 1rem spacing step used for the old bottom margin.
    expect(mainCss).toMatch(/--text-lg:\s*1\.2rem/)
  })

  it('allows purchase form fields to shrink within grid columns', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).toContain('<input v-model="form.purchaseDate" class="form-input" type="date" />')
    // `.form-group { min-width: 0 }` is now the `min-w-0` utility applied
    // directly alongside `form-group` on each field wrapper.
    expect(source).toContain('class="form-group min-w-0"')
  })

  it('keeps a current custom era selectable in the edit form', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).toContain('v-for="era in displayedEraOptions"')
    expect(source).toContain('const displayedEraOptions = computed(() => {')
    expect(source).toContain('return [currentEra, ...eraOptions.value]')
  })

  it('shows the Imperial figure picker only for Roman coins, alongside the free-text Ruler field', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).toContain('v-if="form.category === \'Roman\'"')
    expect(source).toContain('<ImperialFigurePicker v-model="form.romanImperialFigureId!" />')
    expect(source).toContain('<AutocompleteInput v-model="form.ruler!" field="ruler" placeholder="e.g. Augustus" />')
    expect(source).toContain("import ImperialFigurePicker from '@/components/ImperialFigurePicker.vue'")
    expect(source).toContain("if (category !== 'Roman')")
    expect(source).toContain('props.form.romanImperialFigureId = null')
  })
})
