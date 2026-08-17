import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DeepAnalysisPage from '../DeepAnalysisPage.vue'
import {
  applyDeepIdentificationProposal,
  cancelDeepIdentificationJob,
  getDeepIdentificationJob,
  patchDeepIdentificationProposal,
  retryDeepIdentificationJob,
} from '@/api/client'

const routerPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/client', () => ({
  getDeepIdentificationJob: vi.fn(),
  createDeepIdentificationJob: vi.fn(),
  cancelDeepIdentificationJob: vi.fn(),
  retryDeepIdentificationJob: vi.fn(),
  patchDeepIdentificationProposal: vi.fn(),
  applyDeepIdentificationProposal: vi.fn(),
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

  it('never shows "Reconnecting…" when nothing is scheduled, and the Retry control actually recovers the stream (T085/B6)', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: { job: runningJob() },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    const fetchMock = vi.fn()
      // Initial connect fails outright (e.g. transient 5xx) - never
      // becomes connected, so no automatic reconnect is scheduled.
      .mockResolvedValueOnce(new Response(null, { status: 500 }))
      // A manual Retry click reconnects and this time succeeds.
      .mockResolvedValueOnce(new Response(new ReadableStream(), { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mount(DeepAnalysisPage, { global: { stubs: { RouterLink: true, Search: true } } })
    await flushPromises()

    // Honest "Disconnected" label, never a lying "Reconnecting…" with
    // nothing actually scheduled to recover the stream.
    expect(wrapper.text()).not.toContain('Reconnecting')
    expect(wrapper.text()).toContain('Disconnected')
    const retryButton = wrapper.find('button[aria-label="Retry Deep Analysis connection"]')
    expect(retryButton.exists()).toBe(true)

    await retryButton.trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('Live')
    expect(wrapper.find('button[aria-label="Retry Deep Analysis connection"]').exists()).toBe(false)
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

  it('labels intake completion as Save as Draft and opens the created draft', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: {
        job: terminalJob('completed'),
        report: {
          schemaVersion: 1,
          narrative: 'The evidence supports a Roman denarius.',
          coverage: [],
          partialSuccess: false,
          generatedAt: '2030-01-01T00:00:00Z',
        },
        proposal: {
          schemaVersion: 1,
          fields: {
            notes: {
              proposed: 'The evidence supports a Roman denarius.',
              ownerEdited: false,
              ownerValue: null,
              accepted: true,
            },
          },
        },
      },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.mocked(applyDeepIdentificationProposal).mockResolvedValue({
      data: {
        jobId: 9,
        draftId: 27,
        appliedFields: ['notes'],
        appliedAt: '2030-01-01T00:00:00Z',
      },
    } as Awaited<ReturnType<typeof applyDeepIdentificationProposal>>)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
    await flushPromises()

    const saveButton = wrapper.findAll('button').find((button) => button.text().includes('Save as Draft'))
    expect(saveButton).toBeDefined()
    await saveButton!.trigger('click')
    await flushPromises()

    expect(applyDeepIdentificationProposal).toHaveBeenCalledWith(9, { target: 'draft' })
    expect(routerPush).toHaveBeenCalledWith({
      name: 'quick-capture-draft',
      params: { id: '27' },
    })
  })

  it('does not apply when a pre-Apply confidence-default persist fails, and surfaces the failure instead of silently dropping it (RD-3)', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: {
        job: terminalJob('completed'),
        report: {
          schemaVersion: 1,
          narrative: 'The evidence supports a Roman denarius.',
          coverage: [],
          partialSuccess: false,
          generatedAt: '2030-01-01T00:00:00Z',
        },
        proposal: {
          schemaVersion: 1,
          fields: {
            // Never explicitly decided, but confidence-qualifies for the
            // RD-3 default — the page must try to persist this before Apply.
            ruler: {
              proposed: 'Maximinus I (Thrax)',
              confidence: 0.86,
              evidence: [],
              ownerEdited: false,
              ownerValue: null,
              accepted: null,
            },
          },
        },
      },
    } as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.mocked(patchDeepIdentificationProposal).mockRejectedValue(new Error('network down'))
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
    await flushPromises()

    const saveButton = wrapper.findAll('button').find((button) => button.text().includes('Save as Draft'))
    expect(saveButton).toBeDefined()
    await saveButton!.trigger('click')
    await flushPromises()

    // The batched pre-Apply persist was attempted...
    expect(patchDeepIdentificationProposal).toHaveBeenCalledWith(9, { fields: { ruler: { accepted: true } } })
    // ...and because it failed, Apply must never have been called, and the
    // page must say so rather than reporting success on a partial write.
    expect(applyDeepIdentificationProposal).not.toHaveBeenCalled()
    expect(routerPush).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('network down')
  })

  it('never renders hint artifact data returned outside the public job contract', async () => {
    vi.mocked(getDeepIdentificationJob).mockResolvedValue({
      data: {
        job: {
          ...terminalJob('completed'),
          hintImages: [{ path: '/uploads/deep-jobs/job-9/private-hint.jpg' }],
        },
        report: {
          schemaVersion: 1,
          narrative: 'Identification completed.',
          coverage: [],
          partialSuccess: false,
          generatedAt: '2030-01-01T00:00:00Z',
        },
      },
    } as unknown as Awaited<ReturnType<typeof getDeepIdentificationJob>>)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(new ReadableStream(), { status: 200 })))

    const wrapper = mount(DeepAnalysisPage, { global: { stubs } })
    await flushPromises()

    expect(wrapper.text()).not.toContain('private-hint.jpg')
    expect(wrapper.find('img[src*="private-hint"]').exists()).toBe(false)
  })
})
