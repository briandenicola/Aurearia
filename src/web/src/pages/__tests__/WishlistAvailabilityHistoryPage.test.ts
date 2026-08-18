import { flushPromises, mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import WishlistAvailabilityHistoryPage from '../WishlistAvailabilityHistoryPage.vue'

const mocks = vi.hoisted(() => ({
  listMyAvailabilityRuns: vi.fn(),
  getMyAvailabilityRunDetail: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

vi.mock('@/composables/useSafeExternalLink', () => ({
  sanitizeExternalUrl: (url: string | null | undefined) => url ?? null,
}))

function run(overrides: Partial<Record<string, unknown>> = {}) {
  return {
    id: 1,
    userId: 3,
    triggerType: 'owner',
    triggerUserId: 3,
    status: 'completed',
    coinsChecked: 4,
    available: 3,
    unavailable: 1,
    unknown: 0,
    errors: 0,
    durationMs: 900,
    startedAt: '2026-08-01T10:00:00Z',
    completedAt: '2026-08-01T10:00:01Z',
    createdAt: '2026-08-01T10:00:00Z',
    ...overrides,
  }
}

async function buildRouter(initialPath = '/wishlist/availability-runs') {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/wishlist', name: 'wishlist', component: { template: '<div />' } },
      { path: '/wishlist/availability-runs', name: 'wishlist-availability-runs', component: WishlistAvailabilityHistoryPage },
      { path: '/wishlist/availability-runs/:id', name: 'wishlist-availability-run-detail', component: WishlistAvailabilityHistoryPage },
    ],
  })
  router.push(initialPath)
  await router.isReady()
  return router
}

describe('WishlistAvailabilityHistoryPage', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset())
  })

  it('shows a loading state while fetching', async () => {
    let resolvePromise: (value: unknown) => void = () => {}
    mocks.listMyAvailabilityRuns.mockReturnValue(new Promise((resolve) => { resolvePromise = resolve }))
    const router = await buildRouter()
    const wrapper = mount(WishlistAvailabilityHistoryPage, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.find('.spinner').exists()).toBe(true)
    resolvePromise({ data: { runs: [], total: 0, page: 1, limit: 20 } })
    await flushPromises()
  })

  it('shows an empty state when there are no runs', async () => {
    mocks.listMyAvailabilityRuns.mockResolvedValue({ data: { runs: [], total: 0, page: 1, limit: 20 } })
    const router = await buildRouter()
    const wrapper = mount(WishlistAvailabilityHistoryPage, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.text()).toContain('No availability runs yet')
  })

  it('renders a populated list and drills into a run on click', async () => {
    mocks.listMyAvailabilityRuns.mockResolvedValue({ data: { runs: [run()], total: 1, page: 1, limit: 20 } })
    mocks.getMyAvailabilityRunDetail.mockResolvedValue({
      data: {
        ...run(),
        results: [
          { id: 1, runId: 1, coinId: 5, coinName: 'Denarius of Trajan', url: 'https://example.com/lot', status: 'unavailable', reason: 'sold', httpStatus: 200, agentUsed: false, checkedAt: '2026-08-01T10:00:01Z' },
        ],
      },
    })
    const router = await buildRouter()
    const wrapper = mount(WishlistAvailabilityHistoryPage, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.text()).toContain('4 checked')

    const row = wrapper.find('.card')
    await row.trigger('click')
    await flushPromises()

    expect(mocks.getMyAvailabilityRunDetail).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('Denarius of Trajan')
  })

  it('loads a deep-linked run detail directly from the route param', async () => {
    mocks.getMyAvailabilityRunDetail.mockResolvedValue({
      data: { ...run({ id: 42 }), results: [] },
    })
    const router = await buildRouter('/wishlist/availability-runs/42')
    const wrapper = mount(WishlistAvailabilityHistoryPage, { global: { plugins: [router] } })
    await flushPromises()

    expect(mocks.getMyAvailabilityRunDetail).toHaveBeenCalledWith(42)
    expect(mocks.listMyAvailabilityRuns).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Run #42')
  })

  it('handles a 401 by prompting the user to sign in', async () => {
    mocks.listMyAvailabilityRuns.mockRejectedValue({ response: { status: 401 } })
    const router = await buildRouter()
    const wrapper = mount(WishlistAvailabilityHistoryPage, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.text()).toContain('Sign in required')
  })
})
