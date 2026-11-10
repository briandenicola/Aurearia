import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import CoinDetailNotesPage from '../CoinDetailNotesPage.vue'
import { updateCoin } from '@/api/client'

const refresh = vi.fn().mockResolvedValue(undefined)
let currentCoin: { id: number; notes?: string } = { id: 7, notes: 'Original note' }

vi.mock('@/components/coin/CoinDetailSectionPageShell.vue', () => ({
  default: defineComponent({
    setup(_, { slots }) {
      return () => h('div', slots.default?.({ coin: currentCoin, refresh }))
    },
  }),
}))

vi.mock('@/api/client', () => ({
  updateCoin: vi.fn(),
}))

describe('CoinDetailNotesPage', () => {
  beforeEach(() => {
    vi.mocked(updateCoin).mockReset()
    refresh.mockClear()
    currentCoin = { id: 7, notes: 'Original note' }
  })

  it('edits and saves notes inline', async () => {
    vi.mocked(updateCoin).mockResolvedValue({ data: {} } as never)
    const wrapper = mount(CoinDetailNotesPage)

    await wrapper.get('[data-testid="notes-edit"]').trigger('click')
    const editor = wrapper.get('[data-testid="notes-editor"]')
    expect((editor.element as HTMLTextAreaElement).value).toBe('Original note')

    await editor.setValue('Updated note')
    await wrapper.get('[data-testid="notes-save"]').trigger('click')
    await flushPromises()

    expect(updateCoin).toHaveBeenCalledWith(7, { notes: 'Updated note' })
    expect(refresh).toHaveBeenCalled()
    expect(wrapper.find('[data-testid="notes-editor"]').exists()).toBe(false)
  })

  it('opens an empty editor from Add Notes when the coin has no notes', async () => {
    currentCoin = { id: 7, notes: '' }
    const wrapper = mount(CoinDetailNotesPage)

    await wrapper.get('[data-testid="notes-add"]').trigger('click')
    expect((wrapper.get('[data-testid="notes-editor"]').element as HTMLTextAreaElement).value).toBe('')
  })

  it('keeps the editor open and shows an error when saving fails', async () => {
    vi.mocked(updateCoin).mockRejectedValue(new Error('boom'))
    const wrapper = mount(CoinDetailNotesPage)

    await wrapper.get('[data-testid="notes-edit"]').trigger('click')
    await wrapper.get('[data-testid="notes-editor"]').setValue('Changed')
    await wrapper.get('[data-testid="notes-save"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to save notes')
    expect(wrapper.find('[data-testid="notes-editor"]').exists()).toBe(true)
  })
})
