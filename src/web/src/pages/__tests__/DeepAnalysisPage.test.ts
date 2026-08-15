import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DeepAnalysisPage from '../DeepAnalysisPage.vue'
import { getDeepIdentificationJob, cancelDeepIdentificationJob } from '@/api/client'

vi.mock('@/api/client', () => ({
  getDeepIdentificationJob: vi.fn(),
  createDeepIdentificationJob: vi.fn(),
  cancelDeepIdentificationJob: vi.fn(),
  retryDeepIdentificationJob: vi.fn(),
  refreshAccessToken: vi.fn(),
  getApiErrorMessage: (error: unknown) => (error instanceof Error ? error.message : String(error)),
}))

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a><slot /></a>' },
  useRoute: () => ({ params: { jobId: '9' } }),
}))

function runningJob() {
  return {
    id: 9,
    source: 'intake' as const,
    status: 'running' as const,
    partialSuccess: false,
    cancelRequested: false,
    lastSeq: 3,
    eventsAvailable: true,
    expiresAt: '2030-01-01T00:00:00Z',
    createdAt: '2030-01-01T00:00:00Z',
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
  sessionStorage.clear()
  vi.clearAllMocks()
})

describe('DeepAnalysisPage', () => {
  it('loads and displays the job status by route param', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: runningJob() },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs: { RouterLink: true, Search: true } } })
    await flushPromises()

    expect(getDeepIdentificationJob).toHaveBeenCalledWith(9)
    expect(wrapper.text()).toContain('Job #9')
    expect(wrapper.text()).toContain('running')
  })

  it('resumes the SSE stream from the last seen seq stored for this jobId', async () => {
    sessionStorage.setItem('deep-identification-last-seq:9', '42')
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: runningJob() },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    const fetchMock = vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    mount(DeepAnalysisPage, { global: { stubs: { RouterLink: true, Search: true } } })
    await flushPromises()

    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain('since=42')
  })

  it('wires the timeline cancel button to the cancel composable action', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: runningJob() },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.mocked(cancelDeepIdentificationJob).mockResolvedValue({
      data: { job: { ...runningJob(), status: 'cancelled' as const } },
    } as Awaited<ReturnType<typeof cancelDeepIdentificationJob>>)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs: { RouterLink: true, Search: true } } })
    await flushPromises()

    const button = wrapper.find('button[aria-label="Cancel Deep Analysis"]')
    expect(button.exists()).toBe(true)
    await button.trigger('click')
    await flushPromises()

    expect(cancelDeepIdentificationJob).toHaveBeenCalledWith(9)
  })
})

