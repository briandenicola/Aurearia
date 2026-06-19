import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatsValueTrendsPage from '@/pages/StatsValueTrendsPage.vue'
import type { ValueSnapshot, StatsResponse } from '@/types'

const mockHistory: ValueSnapshot[] = [
  { id: 1, userId: 1, totalValue: 1200, totalInvested: 1000, coinCount: 10, recordedAt: '2024-01-01T00:00:00Z' },
  { id: 2, userId: 1, totalValue: 1350, totalInvested: 1050, coinCount: 11, recordedAt: '2024-02-01T00:00:00Z' },
]

const mockStats: StatsResponse = {
  totalCoins: 11,
  totalWishlist: 2,
  byCategory: [],
  byMaterial: [],
  byGrade: [],
  byEra: [],
  byRuler: [],
  byPriceRange: [],
  values: {
    totalPurchasePrice: 1050,
    totalCurrentValue: 1350,
    avgPurchasePrice: 95,
    avgCurrentValue: 122,
  },
}

const store = vi.hoisted(() => ({
  valueHistory: [] as ValueSnapshot[],
  stats: null as StatsResponse | null,
  fetchValueHistory: vi.fn(),
  fetchStats: vi.fn(),
}))

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => store,
}))

describe('StatsValueTrendsPage', () => {
  it('renders Value Details title with value summary after loading completes', async () => {
    store.valueHistory = mockHistory
    store.stats = mockStats
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { template: '<a><slot /></a>' },
          ArrowLeft: { template: '<span />' },
        },
      },
    })

    // Initially shows loading spinner
    expect(wrapper.find('.loading-overlay').exists()).toBe(true)

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(wrapper.text()).toContain('Value Details')
    expect(wrapper.text()).toContain('Total Value')
    expect(wrapper.text()).toContain('Total Invested')
    expect(wrapper.text()).toContain('ROI')
    expect(wrapper.findComponent({ name: 'StatsValueOverTime' }).exists()).toBe(true)
  })

  it('renders the upgraded value chart treatment for history data', async () => {
    store.valueHistory = mockHistory
    store.stats = mockStats
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { template: '<a><slot /></a>' },
          ArrowLeft: { template: '<span />' },
        },
      },
    })

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(wrapper.find('.value-chart-card').exists()).toBe(true)
    expect(wrapper.find('.chart-summary-strip').exists()).toBe(true)
    expect(wrapper.findAll('.summary-pill')).toHaveLength(3)
    expect(wrapper.findAll('.chart-grid-line').length).toBeGreaterThan(0)
    expect(wrapper.find('.chart-area-fill').exists()).toBe(true)
    expect(wrapper.find('.chart-line-value').exists()).toBe(true)
    expect(wrapper.find('.chart-line-invested').exists()).toBe(true)
    expect(wrapper.find('.endpoint-dot-value').exists()).toBe(true)
    expect(wrapper.text()).toContain('Portfolio Trajectory')
  })

  it('renders StatsValueOverTime after loading even when no data', async () => {
    store.valueHistory = []
    store.stats = null
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { template: '<a><slot /></a>' },
          ArrowLeft: { template: '<span />' },
        },
      },
    })

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(wrapper.findComponent({ name: 'StatsValueOverTime' }).exists()).toBe(true)
  })

  it('renders an arrow icon back button to /stats', async () => {
    store.valueHistory = []
    store.stats = null
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
          ArrowLeft: { template: '<span />' },
        },
      },
    })

    await wrapper.vm.$nextTick()

    // Back button is always visible (outside loading v-if)
    const backLink = wrapper.find('a[href="/stats"]')
    expect(backLink.exists()).toBe(true)
    // Icon-only — no text content
    expect(backLink.text().trim()).toBe('')
  })
})
