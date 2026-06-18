import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatsValueTrendsPage from '@/pages/StatsValueTrendsPage.vue'
import type { ValueHistoryEntry } from '@/types'

const mockHistory: ValueHistoryEntry[] = [
  { date: '2024-01-01', totalValue: 1200 },
  { date: '2024-02-01', totalValue: 1350 },
]

const store = vi.hoisted(() => ({
  valueHistory: [] as ValueHistoryEntry[],
  fetchValueHistory: vi.fn(),
}))

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => store,
}))

describe('StatsValueTrendsPage', () => {
  it('renders value over time chart after loading completes', async () => {
    store.valueHistory = mockHistory
    store.fetchValueHistory.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    // Initially shows loading spinner
    expect(wrapper.find('.loading-overlay').exists()).toBe(true)
    expect(wrapper.find('.spinner').exists()).toBe(true)

    // Wait for onMounted to complete
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(wrapper.text()).toContain('Value Trends')
    expect(wrapper.findComponent({ name: 'StatsValueOverTime' }).exists()).toBe(true)
  })

  it('renders StatsValueOverTime after loading even when no data', async () => {
    store.valueHistory = []
    store.fetchValueHistory.mockResolvedValue(undefined)

    const wrapper = mount(StatsValueTrendsPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    // Wait for onMounted to complete
    await wrapper.vm.$nextTick()
    await new Promise((resolve) => setTimeout(resolve, 0))

    // Loading complete, component decides whether to show chart
    expect(wrapper.findComponent({ name: 'StatsValueOverTime' }).exists()).toBe(true)
  })
})
