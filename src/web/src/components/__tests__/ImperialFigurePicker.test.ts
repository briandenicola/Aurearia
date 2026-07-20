import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ImperialFigurePicker from '@/components/ImperialFigurePicker.vue'

const mockSearch = vi.fn()
const mockGetById = vi.fn()

vi.mock('@/api/client', () => ({
  searchRomanImperialFigures: (...args: unknown[]) => mockSearch(...args),
  getRomanImperialFigure: (...args: unknown[]) => mockGetById(...args),
}))

const augustus = { id: 1, name: 'Augustus', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: -27, reignEnd: 14, sortOrder: 1, rarityTier: 'common' }
const livia = { id: 2, name: 'Livia', aliases: [], role: 'empress', region: 'west', dynasty: 'Julio-Claudian', reignStart: -27, reignEnd: 14, sortOrder: 2, rarityTier: 'common' }

describe('ImperialFigurePicker', () => {
  beforeEach(() => {
    mockSearch.mockReset()
    mockGetById.mockReset()
    mockSearch.mockResolvedValue({ data: { figures: [augustus, livia] } })
  })

  it('does not resolve a figure name on mount when modelValue is null', async () => {
    mount(ImperialFigurePicker, { props: { modelValue: null } })
    await flushPromises()
    expect(mockGetById).not.toHaveBeenCalled()
  })

  it('resolves and displays the figure name on mount when modelValue is set', async () => {
    mockGetById.mockResolvedValue({ data: augustus })
    const wrapper = mount(ImperialFigurePicker, { props: { modelValue: 1 } })
    await flushPromises()

    expect(mockGetById).toHaveBeenCalledWith(1)
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('Augustus')
  })

  it('searches on focus and lists results', async () => {
    const wrapper = mount(ImperialFigurePicker, { props: { modelValue: null } })
    await wrapper.find('input').trigger('focus')
    await flushPromises()

    expect(mockSearch).toHaveBeenCalledWith({ q: undefined, role: undefined, limit: 50 })
    expect(wrapper.text()).toContain('Augustus')
    expect(wrapper.text()).toContain('Livia')
  })

  it('selecting a figure emits its id and fills the input with its name', async () => {
    const wrapper = mount(ImperialFigurePicker, { props: { modelValue: null } })
    await wrapper.find('input').trigger('focus')
    await flushPromises()

    const items = wrapper.findAll('li')
    await items[0]!.trigger('mousedown')

    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([1])
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('Augustus')
  })

  it('clicking a role tab filters by that role', async () => {
    const wrapper = mount(ImperialFigurePicker, { props: { modelValue: null } })
    const empressTab = wrapper.findAll('button').find(b => b.text() === 'Empress')!
    await empressTab.trigger('click')
    await flushPromises()

    expect(mockSearch).toHaveBeenCalledWith({ q: undefined, role: 'empress', limit: 50 })
  })

  it('typing after a selection clears the previous selection', async () => {
    mockGetById.mockResolvedValue({ data: augustus })
    const wrapper = mount(ImperialFigurePicker, { props: { modelValue: 1 } })
    await flushPromises()

    await wrapper.find('input').setValue('Aug')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([null])
  })

  it('clear button unsets the selection', async () => {
    mockGetById.mockResolvedValue({ data: augustus })
    const wrapper = mount(ImperialFigurePicker, { props: { modelValue: 1 } })
    await flushPromises()

    const clearButton = wrapper.findAll('button').find(b => b.text() === 'Clear')
    expect(clearButton).toBeTruthy()
    await clearButton!.trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([null])
    expect((wrapper.find('input').element as HTMLInputElement).value).toBe('')
  })

  it('does not show a Clear button when nothing is selected', () => {
    const wrapper = mount(ImperialFigurePicker, { props: { modelValue: null } })
    const clearButton = wrapper.findAll('button').find(b => b.text() === 'Clear')
    expect(clearButton).toBeUndefined()
  })
})
