import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatsPage from '@/pages/StatsPage.vue'
import type { StatsResponse } from '@/types'

const stats: StatsResponse = {
  totalCoins: 42,
  totalWishlist: 7,
  byCategory: [],
  byMaterial: [],
  byGrade: [],
  byEra: [],
  byRuler: [],
  byPriceRange: [],
  values: {
    totalPurchasePrice: 1200,
    totalCurrentValue: 1500,
    avgPurchasePrice: 120,
    avgCurrentValue: 150,
  },
}

const store = vi.hoisted(() => ({
  stats: null as StatsResponse | null,
  fetchStats: vi.fn(),
}))

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => store,
}))

describe('StatsPage', () => {
  it('renders summary metrics instead of stats navigation cards', () => {
    store.stats = stats

    const wrapper = mount(StatsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
        },
      },
    })

    expect(wrapper.text()).toContain('Coins Owned')
    expect(wrapper.text()).toContain('On Wishlist')
    expect(wrapper.text()).toContain('Total Value')
    expect(wrapper.text()).toContain('Value Summary')
    expect(wrapper.text()).not.toContain('Open Mint Map')
    expect(wrapper.text()).not.toContain('Collection Distribution')
    expect(wrapper.findAll('a[href^="/stats/"]')).toHaveLength(0)
  })
})
