import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DeepAnalysisHistoryPage from '../DeepAnalysisHistoryPage.vue'
import { listDeepIdentificationJobs } from '@/api/client'
import type { DeepJob } from '@/types'

const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/client', () => ({
  listDeepIdentificationJobs: vi.fn(),
  getApiErrorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a><slot /></a>' },
  useRouter: () => ({ push: routerPush }),
}))

function buildJob(overrides: Partial<DeepJob> = {}): DeepJob {
  return {
    id: 1,
    source: 'intake',
    status: 'completed',
    partialSuccess: false,
    cancelRequested: false,
    lastSeq: 5,
    eventsAvailable: true,
    expiresAt: '2030-01-01T00:00:00Z',
    createdAt: '2026-06-01T12:00:00Z',
    ...overrides,
  }
}

afterEach(() => {
  vi.clearAllMocks()
})

describe('DeepAnalysisHistoryPage', () => {
  it('renders a list with status pill, source label, and applied-linkage badge', async () => {
    vi.mocked(listDeepIdentificationJobs).mockResolvedValue({
      data: {
        jobs: [
          buildJob({ id: 1, status: 'completed', source: 'saved_coin', appliedAt: '2026-06-01T12:05:00Z', appliedCoinId: 42, appliedCoinExists: true }),
          buildJob({ id: 2, status: 'partial', source: 'intake' }),
        ],
      },
    } as Awaited<ReturnType<typeof listDeepIdentificationJobs>>)

    const wrapper = mount(DeepAnalysisHistoryPage, { global: { stubs: { RouterLink: true, Search: true, ScanSearch: true, ChevronRight: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('Job #1')
    expect(wrapper.text()).toContain('completed')
    expect(wrapper.text()).toContain('Saved Coin')
    expect(wrapper.text()).toContain('Applied')
    expect(wrapper.text()).toContain('Job #2')
    expect(wrapper.text()).toContain('partial')
  })

  it('shows a "Coin removed" badge when the applied coin no longer exists', async () => {
    vi.mocked(listDeepIdentificationJobs).mockResolvedValue({
      data: {
        jobs: [
          buildJob({ id: 3, source: 'saved_coin', appliedAt: '2026-06-01T12:05:00Z', appliedCoinId: 99, appliedCoinExists: false }),
        ],
      },
    } as Awaited<ReturnType<typeof listDeepIdentificationJobs>>)

    const wrapper = mount(DeepAnalysisHistoryPage, { global: { stubs: { RouterLink: true, Search: true, ScanSearch: true, ChevronRight: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('Coin removed')
  })

  it('advances the cursor on "Load more"', async () => {
    vi.mocked(listDeepIdentificationJobs)
      .mockResolvedValueOnce({
        data: { jobs: [buildJob({ id: 1 })], nextCursor: 'cursor-1' },
      } as Awaited<ReturnType<typeof listDeepIdentificationJobs>>)
      .mockResolvedValueOnce({
        data: { jobs: [buildJob({ id: 2 })] },
      } as Awaited<ReturnType<typeof listDeepIdentificationJobs>>)

    const wrapper = mount(DeepAnalysisHistoryPage, { global: { stubs: { RouterLink: true, Search: true, ScanSearch: true, ChevronRight: true } } })
    await flushPromises()

    expect(wrapper.text()).toContain('Load more')
    await wrapper.find('button.btn-secondary').trigger('click')
    await flushPromises()

    expect(listDeepIdentificationJobs).toHaveBeenLastCalledWith({ cursor: 'cursor-1', limit: 20 })
    expect(wrapper.text()).toContain('Job #1')
    expect(wrapper.text()).toContain('Job #2')
    expect(wrapper.text()).not.toContain('Load more')
  })

  it('routes to the job detail page on row click', async () => {
    vi.mocked(listDeepIdentificationJobs).mockResolvedValue({
      data: { jobs: [buildJob({ id: 7 })] },
    } as Awaited<ReturnType<typeof listDeepIdentificationJobs>>)

    const wrapper = mount(DeepAnalysisHistoryPage, { global: { stubs: { RouterLink: true, Search: true, ScanSearch: true, ChevronRight: true } } })
    await flushPromises()

    await wrapper.find('button.card').trigger('click')

    expect(routerPush).toHaveBeenCalledWith({ name: 'deep-analysis', params: { jobId: '7' } })
  })
})
