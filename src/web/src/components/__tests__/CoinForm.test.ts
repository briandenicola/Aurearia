import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CoinForm from '../CoinForm.vue'

const apiMocks = vi.hoisted(() => ({
  getStorageLocations: vi.fn(),
  getStorageLocationOccupancy: vi.fn(),
  getMintLocations: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  getStorageLocations: apiMocks.getStorageLocations,
  getStorageLocationOccupancy: apiMocks.getStorageLocationOccupancy,
  getMintLocations: apiMocks.getMintLocations,
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

vi.mock('@/composables/useCoinOptions', () => ({
  useCoinOptions: () => ({
    categoryOptions: { value: ['Roman', 'Greek'] },
    eraOptions: { value: ['ancient'] },
    materialOptions: { value: ['Silver'] },
    loadOptions: vi.fn(),
  }),
}))

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const coinFormPath = path.resolve(__dirname, '../CoinForm.vue')

describe('CoinForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    apiMocks.getMintLocations.mockResolvedValue({ data: { mintLocations: [] } })
    apiMocks.getStorageLocations.mockResolvedValue({
      data: {
        storageLocations: [
          { id: 1, name: 'Safe', type: 'standard', rows: null, columns: null, capacity: 0, occupied: 0 },
          { id: 2, name: 'Cabinet A', type: 'tray', rows: 2, columns: 2, capacity: 4, occupied: 2 },
          { id: 3, name: 'Cabinet B', type: 'tray', rows: 1, columns: 2, capacity: 2, occupied: 0 },
        ],
      },
    })
    apiMocks.getStorageLocationOccupancy.mockResolvedValue({
      data: { id: 2, rows: 2, columns: 2, capacity: 4, occupiedSlots: [2, 4], currentCoinSlot: 4 },
    })
  })

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

  it('replaces the free-text mint input with a managed dropdown offering Unknown, My Mints, Mints, and Create new', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).not.toContain('<input v-model="form.mint" class="form-input" placeholder="e.g. Rome" />')
    expect(source).toContain('v-model="mintLocationIdModel"')
    expect(source).toContain('<option value="">Unknown</option>')
    expect(source).toContain('label="My Mints"')
    expect(source).toContain('label="Mints"')
    expect(source).toContain('<option value="__create__">+ Create new mint…</option>')
    expect(source).toContain("import CreateMintModal from '@/components/CreateMintModal.vue'")
  })

  it('nudges the user to link a legacy free-text mint that has no matching location, without blocking the form', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).toContain('form.mint && !form.mintLocationId')
    expect(source).toContain('Unlinked legacy mint')
  })

  it('opens the create-mint modal from the dropdown and adopts the newly created mint', () => {
    const source = fs.readFileSync(coinFormPath, 'utf8')

    expect(source).toContain("value === '__create__'")
    expect(source).toContain('showCreateMintModal.value = true')
    expect(source).toContain('function onMintCreated(mintLocation: MintLocation)')
    expect(source).toContain('props.form.mintLocationId = mintLocation.id')
  })

  it('shows exact coordinates only for trays and keeps the current coin slot available', async () => {
    const form = { name: 'Coin', category: 'Roman', material: 'Silver', storageLocationId: 2, storageSlot: 4 }
    const wrapper = mount(CoinForm, {
      props: { form, submitLabel: 'Save', coinId: 99 },
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          AutocompleteInput: true,
          ImperialFigurePicker: true,
          CreateMintModal: true,
          AuthenticatedImage: true,
          X: true,
          Camera: true,
        },
      },
    })
    await flushPromises()

    expect(apiMocks.getStorageLocationOccupancy).toHaveBeenCalledWith(2, 99)
    expect(wrapper.findAll('[role="gridcell"]')).toHaveLength(4)
    expect(wrapper.get('[aria-label="Row 1, column 2, occupied"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[aria-label="Row 2, column 2, selected"]').attributes('disabled')).toBeUndefined()
  })

  it('resets stale slots on location changes and requires a new tray slot before submit', async () => {
    const form = { name: 'Coin', category: 'Roman', material: 'Silver', storageLocationId: 2, storageSlot: 4 }
    const wrapper = mount(CoinForm, {
      props: { form, submitLabel: 'Save', coinId: 99 },
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          AutocompleteInput: true,
          ImperialFigurePicker: true,
          CreateMintModal: true,
          AuthenticatedImage: true,
          X: true,
          Camera: true,
        },
      },
    })
    await flushPromises()

    apiMocks.getStorageLocationOccupancy.mockResolvedValueOnce({
      data: { id: 3, rows: 1, columns: 2, capacity: 2, occupiedSlots: [], currentCoinSlot: null },
    })
    const storageSelect = wrapper.findAll('select.form-select').find((select) => select.text().includes('Cabinet B'))
    await storageSelect!.setValue('3')
    await flushPromises()

    expect(form.storageSlot).toBeNull()
    await wrapper.find('form').trigger('submit')
    expect(wrapper.text()).toContain('Choose an exact tray slot before saving.')
    expect(wrapper.emitted('submit')).toBeUndefined()

    await wrapper.get('[aria-label="Row 1, column 1, available"]').trigger('click')
    await wrapper.find('form').trigger('submit')
    expect(form.storageSlot).toBe(1)
    expect(wrapper.emitted('submit')).toHaveLength(1)
  })

  it('refreshes stale occupancy after a conflict without losing other form data', async () => {
    const form = {
      name: 'Preserved name',
      category: 'Roman',
      material: 'Silver',
      notes: 'Preserved notes',
      storageLocationId: 2,
      storageSlot: 1,
    }
    const wrapper = mount(CoinForm, {
      props: { form, submitLabel: 'Save', coinId: 99 },
      global: {
        stubs: {
          RouterLink: { template: '<a><slot /></a>' },
          AutocompleteInput: true,
          ImperialFigurePicker: true,
          CreateMintModal: true,
          AuthenticatedImage: true,
          X: true,
          Camera: true,
        },
      },
    })
    await flushPromises()

    apiMocks.getStorageLocationOccupancy.mockRejectedValueOnce(new Error('stale'))
    await (wrapper.vm as unknown as { refreshStorageOccupancy: () => Promise<void> }).refreshStorageOccupancy()
    await flushPromises()
    expect(wrapper.text()).toContain('Tray occupancy changed or could not be loaded.')

    apiMocks.getStorageLocationOccupancy.mockResolvedValueOnce({
      data: { id: 2, rows: 2, columns: 2, capacity: 4, occupiedSlots: [1, 2], currentCoinSlot: null },
    })
    await wrapper.findAll('button').find((button) => button.text() === 'Retry')!.trigger('click')
    await flushPromises()

    expect(form.name).toBe('Preserved name')
    expect(form.notes).toBe('Preserved notes')
    expect(form.storageSlot).toBe(1)
    expect(wrapper.get('[aria-label="Row 1, column 1, occupied"]').attributes('disabled')).toBeDefined()
  })
})
