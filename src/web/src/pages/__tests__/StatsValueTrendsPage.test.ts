import { mount } from '@vue/test-utils'
import { describe, expect, it, vi, afterEach } from 'vitest'
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

const defaultStubs = {
  PullToRefresh: { template: '<div><slot /></div>' },
  RouterLink: { template: '<a><slot /></a>' },
  ArrowLeft: { template: '<span />' },
}

// Stub StatsValueOverTime so timeframe filter tests can inspect the `history` prop
// without depending on the chart component's rendering internals.
const chartStub = {
  name: 'StatsValueOverTime',
  props: ['history'],
  template: '<div class="chart-stub" />',
}

// Extended history spanning >1 year for timeframe filter tests.
// All dates are relative to the pinned fake system time of 2024-03-01:
//   cutoff(1Y)  = 2023-03-02  → excludes id:10 (2022-06-01), includes ids 11–13
//   cutoff(6M)  = 2023-09-02  → excludes ids 10–11, includes ids 12–13
//   cutoff(3M)  = 2023-12-01  → excludes ids 10–11, includes ids 12–13
const extendedHistory: ValueSnapshot[] = [
  { id: 10, userId: 1, totalValue: 800, totalInvested: 750, coinCount: 6, recordedAt: '2022-06-01T00:00:00Z' },
  { id: 11, userId: 1, totalValue: 1000, totalInvested: 900, coinCount: 8, recordedAt: '2023-06-01T00:00:00Z' },
  { id: 12, userId: 1, totalValue: 1200, totalInvested: 1000, coinCount: 10, recordedAt: '2024-01-15T00:00:00Z' },
  { id: 13, userId: 1, totalValue: 1350, totalInvested: 1050, coinCount: 11, recordedAt: '2024-02-15T00:00:00Z' },
]

afterEach(() => {
  vi.useRealTimers()
})

describe('StatsValueTrendsPage', () => {
  it('renders Value Details title with value summary after loading completes', async () => {
    store.valueHistory = mockHistory
    store.stats = mockStats
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: { stubs: defaultStubs },
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
      global: { stubs: defaultStubs },
    })

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    // Chart card and infographic elements from StatsValueOverTime
    expect(wrapper.find('.stats-section').exists()).toBe(true)
    expect(wrapper.find('.stats-section').text()).toContain('Period Value Change')
    expect(wrapper.findAll('.rounded-sm.bg-input')).toHaveLength(3)
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
      global: { stubs: defaultStubs },
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
          ...defaultStubs,
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
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

  // ── Timeframe / zoom controls ────────────────────────────────────────────

  it('renders All/1Y/6M/3M timeframe zoom chips after loading', async () => {
    store.valueHistory = mockHistory
    store.stats = mockStats
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: { stubs: { ...defaultStubs, StatsValueOverTime: chartStub } },
    })

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    const chips = wrapper.findAll('.chip')
    expect(chips.length).toBeGreaterThan(0)
    const labels = chips.map((c) => c.text())
    expect(labels).toContain('All')
    expect(labels).toContain('1Y')
    expect(labels).toContain('6M')
    expect(labels).toContain('3M')
  })

  it('defaults to All timeframe with All chip marked active', async () => {
    store.valueHistory = mockHistory
    store.stats = mockStats
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: { stubs: { ...defaultStubs, StatsValueOverTime: chartStub } },
    })

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    const chips = wrapper.findAll('.chip')
    const allChip = chips.find((c) => c.text() === 'All')
    expect(allChip?.classes()).toContain('border-gold')

    // No other chip should be active initially
    const otherActive = chips.filter((c) => c.text() !== 'All' && c.classes().includes('border-gold'))
    expect(otherActive).toHaveLength(0)
  })

  it('clicking a timeframe chip filters history passed to StatsValueOverTime', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-03-01T00:00:00Z'))

    store.valueHistory = extendedHistory
    store.stats = mockStats
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: { stubs: { ...defaultStubs, StatsValueOverTime: chartStub } },
    })

    // Flush microtasks (onMounted Promise.all resolves) then Vue reactivity
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()
    // Advance any pending macrotasks so the loading guard clears
    vi.runAllTimers()
    await wrapper.vm.$nextTick()

    const chart = wrapper.findComponent({ name: 'StatsValueOverTime' })

    // Default "All" passes the full history
    expect(chart.props('history')).toHaveLength(4)

    // Click "1Y" — only items within the last 365 days (cutoff ~2023-03-02)
    const chips = wrapper.findAll('.chip')
    const oneYearChip = chips.find((c) => c.text() === '1Y')
    await oneYearChip!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(chart.props('history')).toHaveLength(3)

    // Click "6M" — only items within the last 180 days (cutoff ~2023-09-02)
    const sixMonthChip = chips.find((c) => c.text() === '6M')
    await sixMonthChip!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(chart.props('history')).toHaveLength(2)

    // Clicking "All" restores the full set
    const allChip = chips.find((c) => c.text() === 'All')
    await allChip!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(chart.props('history')).toHaveLength(4)
  })

  it('selected chip receives active class and previously active chip loses it', async () => {
    store.valueHistory = mockHistory
    store.stats = mockStats
    store.fetchValueHistory.mockResolvedValue(undefined)
    store.fetchStats.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: { stubs: { ...defaultStubs, StatsValueOverTime: chartStub } },
    })

    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    const chips = wrapper.findAll('.chip')
    const oneYearChip = chips.find((c) => c.text() === '1Y')
    const allChip = chips.find((c) => c.text() === 'All')

    await oneYearChip!.trigger('click')
    await wrapper.vm.$nextTick()

    expect(oneYearChip!.classes()).toContain('border-gold')
    expect(allChip!.classes()).not.toContain('border-gold')
  })
})
