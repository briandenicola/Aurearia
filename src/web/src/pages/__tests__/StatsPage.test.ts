import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatsPage from '@/pages/StatsPage.vue'
import type { StatsResponse } from '@/types'

const stats: StatsResponse = {
  totalCoins: 42,
  totalWishlist: 7,
  byCategory: [{ category: 'Roman', count: 20 }, { category: 'Greek', count: 15 }],
  byMaterial: [],
  byGrade: [],
  byEra: [{ era: 'Republic', count: 10 }],
  byRuler: [{ ruler: 'Augustus', count: 5 }],
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
  it('renders summary metrics and distribution charts; excludes value detail block', () => {
    store.stats = stats

    const wrapper = mount(StatsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          StatsHeatMap: { template: '<div class="stub-heatmap" />', methods: { fetchDistribution: () => {} } },
          StatsCoinFlowChart: { template: '<div class="stub-coin-flow" />' },
        },
      },
    })

    expect(wrapper.text()).toContain('Coins Owned')
    expect(wrapper.text()).toContain('On Wishlist')
    expect(wrapper.text()).toContain('42')
    expect(wrapper.text()).toContain('7')

    // Value detail block has moved to Value Details page
    expect(wrapper.text()).not.toContain('Value Summary')
    expect(wrapper.text()).not.toContain('Total Invested')

    // Distribution charts are present
    expect(wrapper.text()).toContain('By Coin Type')
    expect(wrapper.text()).toContain('By Era')
    expect(wrapper.text()).toContain('Top Rulers')

    expect(wrapper.text()).not.toContain('Open Mint Map')
    expect(wrapper.findAll('a[href^="/stats/"]')).toHaveLength(0)
  })

  it('renders stats from the normal stats store without querying quick-capture drafts', () => {
    store.stats = stats

    const wrapper = mount(StatsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          StatsHeatMap: { template: '<div />', methods: { fetchDistribution: () => {} } },
          StatsCoinFlowChart: { template: '<div />' },
        },
      },
    })

    expect(store.fetchStats).toHaveBeenCalled()
    expect(wrapper.text()).not.toContain('Quick Capture Draft')
  })
})
