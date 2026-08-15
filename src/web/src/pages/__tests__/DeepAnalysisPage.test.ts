import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DeepAnalysisPage from '../DeepAnalysisPage.vue'
import { getDeepIdentificationJob } from '@/api/client'

vi.mock('@/api/client', () => ({
  getDeepIdentificationJob: vi.fn(),
  createDeepIdentificationJob: vi.fn(),
  getApiErrorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a><slot /></a>' },
  useRoute: () => ({ params: { jobId: '9' } }),
}))

describe('DeepAnalysisPage', () => {
  it('loads and displays the job status by route param', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: {
        job: {
          id: 9,
          source: 'intake',
          status: 'running',
          partialSuccess: false,
          cancelRequested: false,
          lastSeq: 3,
          eventsAvailable: true,
          expiresAt: '2030-01-01T00:00:00Z',
          createdAt: '2030-01-01T00:00:00Z',
        },
      },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)

    const wrapper = mount(DeepAnalysisPage, { global: { stubs: { RouterLink: true, Search: true } } })
    await flushPromises()

    expect(getDeepIdentificationJob).toHaveBeenCalledWith(9)
    expect(wrapper.text()).toContain('Job #9')
    expect(wrapper.text()).toContain('running')
  })
})
