import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetDetailPage from '@/pages/SetDetailPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const mockAddCoinToSet = vi.fn()
const mockCompareSets = vi.fn()
const mockCreateSetSnapshot = vi.fn()
const mockDeleteSet = vi.fn()
const mockGetCoins = vi.fn()
const mockGetCoinsInSet = vi.fn()
const mockGetSet = vi.fn()
const mockGetSetAnalytics = vi.fn()
const mockGetSetCompletion = vi.fn()
const mockGetSets = vi.fn()
const mockGetSetTrends = vi.fn()
const mockReorderSetCoins = vi.fn()
const mockRemoveCoinFromSet = vi.fn()
const mockUpdateSet = vi.fn()
const mockPush = vi.fn()

vi.mock('@/api/client', () => ({
  addCoinToSet: (...args: unknown[]) => mockAddCoinToSet(...args),
  compareSets: (...args: unknown[]) => mockCompareSets(...args),
  createSetSnapshot: (...args: unknown[]) => mockCreateSetSnapshot(...args),
  deleteSet: (...args: unknown[]) => mockDeleteSet(...args),
  getCoins: (...args: unknown[]) => mockGetCoins(...args),
  getCoinsInSet: (...args: unknown[]) => mockGetCoinsInSet(...args),
  getSet: (...args: unknown[]) => mockGetSet(...args),
  getSetAnalytics: (...args: unknown[]) => mockGetSetAnalytics(...args),
  getSetCompletion: (...args: unknown[]) => mockGetSetCompletion(...args),
  getSets: (...args: unknown[]) => mockGetSets(...args),
  getSetTrends: (...args: unknown[]) => mockGetSetTrends(...args),
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
  setType: 'defined',
  coinCount: 13,
  totalValue: 1300,
  totalInvested: 900,
}

const defaultStubs = {
  SetCompletionChecklist: true,
  SetTrendChart: true,
  SetComparePanel: true,
  MuseumTray: true,
  TrayControls: true,
}

function mockSetDetailLoad(coins = [
  buildRomanDenariusCore({ id: 1, name: 'Augustus Denarius', diameterMm: 19 }),
  buildRomanDenariusCore({ id: 2, name: 'Tiberius Denarius', diameterMm: null }),
]) {
  mockGetSet.mockResolvedValue({ data: defaultSet })
  mockGetCoinsInSet.mockResolvedValue({ data: { coins } })
  mockGetSetTrends.mockResolvedValue({ data: { snapshots: [] } })
  mockGetSetAnalytics.mockResolvedValue({ data: { roiPercent: 12.5, acquisitionRatePerMonth: 1.2 } })
  mockGetSets.mockResolvedValue({ data: { sets: [defaultSet] } })
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
    expect(tray.props('coins')).toEqual([
      {
        id: 1,
        name: 'Augustus Denarius',
        diameterMm: 19,
        images: expect.any(Array),
      },
      {
        id: 2,
        name: 'Tiberius Denarius',
        diameterMm: null,
        images: expect.any(Array),
      },
    ])
    expect(wrapper.find('.coin-card').exists()).toBe(false)
    expect(wrapper.find('.coin-image').exists()).toBe(false)
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
})
