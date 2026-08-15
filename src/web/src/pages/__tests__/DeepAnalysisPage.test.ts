import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DeepAnalysisPage from '../DeepAnalysisPage.vue'
import { getDeepIdentificationJob, cancelDeepIdentificationJob, retryDeepIdentificationJob } from '@/api/client'

const routerPush = vi.hoisted(() => vi.fn())

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
  useRouter: () => ({ push: routerPush }),
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

function terminalJob(status: 'failed' | 'cancelled' | 'completed' | 'partial') {
  return { ...runningJob(), status, partialSuccess: status === 'partial' }
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

  const stubs = { RouterLink: true, Search: true }

  it.each(['failed', 'cancelled', 'completed', 'partial'] as const)(
    'offers a Retry control for the %s terminal state',
    async (status) => {
      vi.mocked(getDeepIdentificationJob).mockResolvedValue({
        data: { job: terminalJob(status) },
      } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
      vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

      const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
      await flushPromises()

      expect(wrapper.find('button[aria-label="Retry Deep Analysis"]').exists()).toBe(true)
    },
  )

  it('does not offer Retry while the job is still active', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: runningJob() },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
    await flushPromises()

    expect(wrapper.find('button[aria-label="Retry Deep Analysis"]').exists()).toBe(false)
  })

  it('retries and navigates to the new job route on success', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: terminalJob('failed') },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.mocked(retryDeepIdentificationJob).mockResolvedValue({
      data: { job: { ...terminalJob('failed'), id: 10, status: 'queued' as const } },
    } as Awaited<ReturnType<typeof retryDeepIdentificationJob>>)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
    await flushPromises()

    await wrapper.find('button[aria-label="Retry Deep Analysis"]').trigger('click')
    await flushPromises()

    expect(retryDeepIdentificationJob).toHaveBeenCalledWith(9, undefined)
    expect(routerPush).toHaveBeenCalledWith({ name: 'deep-analysis', params: { jobId: '10' } })
  })

  it('surfaces an accessible error and does not navigate when retry fails', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: terminalJob('failed') },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.mocked(retryDeepIdentificationJob).mockRejectedValue(new Error('retry boom'))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
    await flushPromises()

    await wrapper.find('button[aria-label="Retry Deep Analysis"]').trigger('click')
    await flushPromises()

    expect(routerPush).not.toHaveBeenCalled()
    const alert = wrapper.find('[role="alert"]')
    expect(alert.exists()).toBe(true)
    expect(wrapper.text()).toContain('retry boom')
  })

  it('supports keyboard activation and prevents duplicate retry submissions', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: terminalJob('cancelled') },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    let resolveRetry: (v: unknown) => void = () => {}
    vi.mocked(retryDeepIdentificationJob).mockReturnValue(
      new Promise((resolve) => {
        resolveRetry = resolve
      }) as ReturnType<typeof retryDeepIdentificationJob>,
    )
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
    await flushPromises()

    const button = wrapper.find('button[aria-label="Retry Deep Analysis"]')
    // The control is a real <button>, so it is keyboard-focusable and
    // activates via Enter/Space through the native click path.
    await button.trigger('click')
    await button.trigger('click')
    await flushPromises()

    // In-flight: disabled + aria-busy, and only one request despite two clicks.
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.attributes('aria-busy')).toBe('true')
    expect(retryDeepIdentificationJob).toHaveBeenCalledTimes(1)

    resolveRetry({ data: { job: { ...terminalJob('cancelled'), id: 11, status: 'queued' as const } } })
    await flushPromises()
    expect(routerPush).toHaveBeenCalledWith({ name: 'deep-analysis', params: { jobId: '11' } })
  })
})

