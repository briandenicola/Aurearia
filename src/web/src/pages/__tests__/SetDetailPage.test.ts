import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetDetailPage from '@/pages/SetDetailPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const mockAddCoinToSet = vi.fn()
const mockDeleteSet = vi.fn()
const mockGetCoins = vi.fn()
const mockGetCoinsInSet = vi.fn()
const mockGetSet = vi.fn()
const mockGetSetCompletion = vi.fn()
const mockReorderSetCoins = vi.fn()
const mockRemoveCoinFromSet = vi.fn()
const mockUpdateSet = vi.fn()
const mockPush = vi.fn()

vi.mock('@/api/client', () => ({
  addCoinToSet: (...args: unknown[]) => mockAddCoinToSet(...args),
  deleteSet: (...args: unknown[]) => mockDeleteSet(...args),
  getCoins: (...args: unknown[]) => mockGetCoins(...args),
  getCoinsInSet: (...args: unknown[]) => mockGetCoinsInSet(...args),
  getSet: (...args: unknown[]) => mockGetSet(...args),
  getSetCompletion: (...args: unknown[]) => mockGetSetCompletion(...args),
  reorderSetCoins: (...args: unknown[]) => mockReorderSetCoins(...args),
  removeCoinFromSet: (...args: unknown[]) => mockRemoveCoinFromSet(...args),
  updateSet: (...args: unknown[]) => mockUpdateSet(...args),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '7' } }),
  useRouter: () => ({ push: mockPush }),
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

vi.mock('@/composables/useTrayPreference', () => ({
  useTrayPreference: () => ({ feltColor: 'navy' }),
}))

const defaultSet = {
  id: 7,
  name: 'Twelve Caesars',
  color: '#c9a84c',
  setType: 'goal',
  coinCount: 13,
  totalValue: 1300,
  totalInvested: 900,
}

const defaultStubs = {
  SetCompletionChecklist: true,
  MuseumTray: true,
  TrayControls: true,
}

function mockSetDetailLoad(coins = [
  buildRomanDenariusCore({ id: 1, name: 'Augustus Denarius', diameterMm: 19 }),
  buildRomanDenariusCore({ id: 2, name: 'Tiberius Denarius', diameterMm: null }),
]) {
  mockGetSet.mockResolvedValue({ data: defaultSet })
  mockGetCoinsInSet.mockResolvedValue({ data: { coins } })
  mockGetCoins.mockResolvedValue({ data: { coins: [], total: 0 } })
  mockGetSetCompletion.mockResolvedValue({
    data: { totalTargets: 12, completedTargets: 2, completionPercentage: 16.7, missingTargets: [] },
  })
}

describe('SetDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockSetDetailLoad()
  })

  it('displays set coins in the museum tray instead of image cards', async () => {
    const wrapper = shallowMount(SetDetailPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    const tray = wrapper.findComponent({ name: 'MuseumTray' })
    expect(tray.exists()).toBe(true)
    expect(tray.props('feltTheme')).toBe('navy')
    expect(tray.props('showNames')).toBe(true)
    expect(tray.props('coins')).toEqual([
      {
        id: 1,
        name: 'Augustus Denarius',
        diameterMm: 19,
        images: expect.any(Array),
        purchaseDate: expect.anything(),
        wishlistPlaceholder: false,
      },
      {
        id: 2,
        name: 'Tiberius Denarius',
        diameterMm: null,
        images: expect.any(Array),
        purchaseDate: expect.anything(),
        wishlistPlaceholder: false,
      },
    ])
    expect(wrapper.find('.coin-card').exists()).toBe(false)
    expect(wrapper.find('.coin-image').exists()).toBe(false)
  })

  it('ghosts wishlist coins in goal set tray data', async () => {
    mockSetDetailLoad([
      buildRomanDenariusCore({ id: 1, name: 'Owned Denarius', isWishlist: false, purchaseDate: '2026-07-26T12:00:00Z' }),
      buildRomanDenariusCore({ id: 2, name: 'Wishlist Denarius', isWishlist: true }),
    ])

    const wrapper = shallowMount(SetDetailPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    const tray = wrapper.findComponent({ name: 'MuseumTray' })
    expect(tray.props('coins')).toEqual([
      expect.objectContaining({ id: 1, purchaseDate: '2026-07-26T12:00:00Z', wishlistPlaceholder: false }),
      expect.objectContaining({ id: 2, wishlistPlaceholder: true }),
    ])
  })

  it('embeds tray controls for multi-drawer sets', async () => {
    const coins = Array.from({ length: 13 }, (_, index) =>
      buildRomanDenariusCore({ id: index + 1, name: `Set Coin ${index + 1}`, diameterMm: 18 }),
    )
    mockSetDetailLoad(coins)

    const wrapper = shallowMount(SetDetailPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    const controls = wrapper.findComponent({ name: 'TrayControls' })
    expect(controls.exists()).toBe(true)
    expect(controls.props('fixed')).toBe(false)
    expect(controls.props('drawerIndex')).toBe(0)
    expect(controls.props('totalDrawers')).toBe(2)

    controls.vm.$emit('next')
    await wrapper.vm.$nextTick()

    const tray = wrapper.findComponent({ name: 'MuseumTray' })
    expect(controls.props('drawerIndex')).toBe(1)
    expect(tray.props('coins')).toHaveLength(1)
    expect(tray.props('coins')?.[0]?.name).toBe('Set Coin 13')
  })

  it('opens coin detail from a tray well click', async () => {
    const wrapper = shallowMount(SetDetailPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    wrapper.findComponent({ name: 'MuseumTray' }).vm.$emit('coin-clicked', 2)

    expect(mockPush).toHaveBeenCalledWith({ name: 'coin-detail', params: { id: 2 } })
  })

  it('opens analytics and trends page from actions menu', async () => {
    const wrapper = shallowMount(SetDetailPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    const actionsButton = wrapper.findAll('button').find((node) => node.text().includes('Actions'))
    expect(actionsButton?.exists()).toBe(true)
    await actionsButton!.trigger('click')

    const infoAction = wrapper.findAll('button').find((node) => node.text().includes('Analytics & Value Trend'))
    expect(infoAction?.exists()).toBe(true)
    await infoAction!.trigger('click')

    expect(mockPush).toHaveBeenCalledWith({ name: 'set-insights', params: { id: 7 } })
  })
})
