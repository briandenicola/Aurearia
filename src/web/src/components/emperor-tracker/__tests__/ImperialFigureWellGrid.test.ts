import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ImperialFigureWellGrid from '@/components/emperor-tracker/ImperialFigureWellGrid.vue'
import type { ImperialFigureSlot } from '@/types'

const mockPush = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

const ownedSlot: ImperialFigureSlot = {
  figure: { id: 1, name: 'Augustus', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: -27, reignEnd: 14, sortOrder: 1, rarityTier: 'common' },
  coin: { id: 42, name: 'My Augustus Denarius', diameterMm: 18.5, images: [] } as ImperialFigureSlot['coin'],
}

const unownedSlot: ImperialFigureSlot = {
  figure: { id: 2, name: 'Tiberius', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: 14, reignEnd: 37, sortOrder: 2, rarityTier: 'common' },
  coin: null,
}

describe('ImperialFigureWellGrid', () => {
  it('shows a visible name caption for every figure', () => {
    const wrapper = mount(ImperialFigureWellGrid, { props: { slots: [ownedSlot, unownedSlot] } })
    expect(wrapper.text()).toContain('Augustus')
    expect(wrapper.text()).toContain('Tiberius')
  })

  it('renders an owned figure as interactive and navigates to the coin on click', async () => {
    const wrapper = mount(ImperialFigureWellGrid, { props: { slots: [ownedSlot] } })
    const well = wrapper.find('.tray-well')
    expect(well.attributes('role')).toBe('button')

    await well.trigger('click')
    expect(mockPush).toHaveBeenCalledWith({ name: 'coin-detail', params: { id: 42 } })
  })

  it('renders an unowned figure as a non-interactive placeholder that never navigates', async () => {
    mockPush.mockClear()
    const wrapper = mount(ImperialFigureWellGrid, { props: { slots: [unownedSlot] } })
    const well = wrapper.find('.tray-well')
    expect(well.attributes('role')).toBeUndefined()

    await well.trigger('click')
    expect(mockPush).not.toHaveBeenCalled()
  })

  it('uses a synthetic negative id for unowned placeholder wells', () => {
    const wrapper = mount(ImperialFigureWellGrid, { props: { slots: [unownedSlot] } })
    // No real coin id, so the placeholder must never collide with one.
    expect(wrapper.find('.well-placeholder').exists()).toBe(true)
  })
})
