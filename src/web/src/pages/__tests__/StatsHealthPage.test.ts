import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatsHealthPage from '@/pages/StatsHealthPage.vue'
import type { CollectionHealth } from '@/types'

const mockHealth: CollectionHealth = {
  overallScore: 85,
  score: 85,
  grade: 'B',
  fields: {
    denomination: { score: 90, filled: 45, total: 50 },
    ruler: { score: 80, filled: 40, total: 50 },
  },
  trend30d: { direction: 'up', change: 5 },
}

const store = vi.hoisted(() => ({
  collectionHealth: null as CollectionHealth | null,
  healthLoading: false,
  fetchCollectionHealth: vi.fn(() => Promise.resolve()),
}))

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => store,
}))

describe('StatsHealthPage', () => {
  it('renders health scorecard when data is available', () => {
    store.collectionHealth = mockHealth
    store.healthLoading = false

    const wrapper = mount(StatsHealthPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.text()).toContain('Health')
    expect(wrapper.findComponent({ name: 'CollectionHealthScorecard' }).exists()).toBe(true)
  })

  it('shows empty state when no health data exists', () => {
    store.collectionHealth = null
    store.healthLoading = false

    const wrapper = mount(StatsHealthPage, {
      global: {
        stubs: {
          PullToRefresh: { template: '<div><slot /></div>' },
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })

    expect(wrapper.findComponent({ name: 'CollectionHealthEmptyState' }).exists()).toBe(true)
  })
})
