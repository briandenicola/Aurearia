import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import CoinAIAnalysis from '../CoinAIAnalysis.vue'

const mocks = vi.hoisted(() => ({
  analyzeCoin: vi.fn(),
  deleteAnalysis: vi.fn(),
  getAIJob: vi.fn(),
  getAIStatus: vi.fn(),
  getCoinAIJobs: vi.fn(),
  gradeCoin: vi.fn(),
  refreshNotifications: vi.fn(),
  showAlert: vi.fn(),
  showConfirm: vi.fn(),
  showToast: vi.fn(),
}))

vi.mock('@/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/client')>()
  return {
    ...actual,
    analyzeCoin: mocks.analyzeCoin,
    deleteAnalysis: mocks.deleteAnalysis,
    getAIJob: mocks.getAIJob,
    getAIStatus: mocks.getAIStatus,
    getCoinAIJobs: mocks.getCoinAIJobs,
    gradeCoin: mocks.gradeCoin,
  }
})

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showAlert: mocks.showAlert,
    showConfirm: mocks.showConfirm,
  }),
}))

vi.mock('@/composables/useNotifications', () => ({
  useNotifications: () => ({
    refresh: mocks.refreshNotifications,
  }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({
    showToast: mocks.showToast,
  }),
}))

describe('CoinAIAnalysis', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getAIStatus.mockResolvedValue({
      data: {
        available: true,
        provider: 'anthropic',
        model: 'claude-sonnet-4',
        message: 'Anthropic provider configured',
      },
    })
    mocks.showAlert.mockResolvedValue(undefined)
    mocks.showConfirm.mockResolvedValue(true)
    mocks.getCoinAIJobs.mockResolvedValue({ data: [] })
    mocks.getAIJob.mockResolvedValue({
      data: {
        id: 'job-default',
        coinId: 42,
        jobType: 'coin_analysis',
        side: 'obverse',
        status: 'queued',
        createdAt: '',
        updatedAt: '',
      },
    })
    sessionStorage.clear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('does not send agent service failures to Anthropic provider settings', async () => {
    mocks.analyzeCoin.mockRejectedValue({
      response: {
        data: {
          error: 'Agent service unavailable',
        },
      },
    })

    const wrapper = mountAnalysis()
    await flushPromises()
    await findActionButton(wrapper, 'Analyze obverse').trigger('click')
    await flushPromises()

    expect(mocks.showAlert).toHaveBeenCalledWith(
      'AI analysis failed for obverse. Agent service unavailable. Check the internal agent service configuration.',
      { title: 'Analysis Failed' },
    )
  })

  it('surfaces missing internal credential configuration as service setup, not provider setup', async () => {
    mocks.analyzeCoin.mockRejectedValue({
      response: {
        data: {
          detail: 'Internal service credential is not configured',
        },
      },
    })

    const wrapper = mountAnalysis()
    await flushPromises()
    await findActionButton(wrapper, 'Analyze obverse').trigger('click')
    await flushPromises()

    expect(mocks.showAlert).toHaveBeenCalledWith(
      'AI analysis failed for obverse. Internal agent service credential is not configured. Check the internal agent service configuration.',
      { title: 'Analysis Failed' },
    )
    expect(mocks.showAlert.mock.calls[0]?.[0]).not.toMatch(/Anthropic API|provider in Admin/i)
  })

  it('submits an analysis job, polls, and emits refresh on completion', async () => {
    vi.useFakeTimers()
    mocks.analyzeCoin.mockResolvedValue({
      data: {
        jobId: 'job-1',
        coinId: 42,
        jobType: 'coin_analysis',
        side: 'obverse',
        status: 'queued',
      },
    })
    mocks.getAIJob
      .mockResolvedValueOnce({
        data: {
          id: 'job-1',
          coinId: 42,
          jobType: 'coin_analysis',
          side: 'obverse',
          status: 'running',
          createdAt: '',
          updatedAt: '',
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 'job-1',
          coinId: 42,
          jobType: 'coin_analysis',
          side: 'obverse',
          status: 'completed',
          createdAt: '',
          updatedAt: '',
        },
      })

    const wrapper = mountAnalysis()
    await flushPromises()
    await findActionButton(wrapper, 'Analyze obverse').trigger('click')
    await flushPromises()

    expect(mocks.analyzeCoin).toHaveBeenCalledWith(42, 'obverse')
    expect(wrapper.text()).toContain('Obverse analysis in progress.')

    await vi.advanceTimersByTimeAsync(3_000)
    await flushPromises()

    expect(wrapper.emitted('analysisUpdated')).toHaveLength(1)
    expect(mocks.refreshNotifications).toHaveBeenCalled()
    expect(sessionStorage.getItem('aiJob:analysis:42:obverse')).toBeNull()
  })

  it('disables coin grading until at least one coin image is available', async () => {
    const wrapper = mountAnalysis({ hasObverse: false, hasReverse: false })
    await flushPromises()

    const gradeButton = findActionButton(wrapper, 'Grade coin')

    expect(gradeButton.attributes('disabled')).toBeDefined()
    expect(gradeButton.attributes('title')).toBe('Add coin photos before requesting grading')
    await gradeButton.trigger('click')
    expect(mocks.gradeCoin).not.toHaveBeenCalled()
  })

  it('allows coin grading when only one side image is available', async () => {
    vi.useFakeTimers()
    mocks.gradeCoin.mockResolvedValue({
      data: {
        jobId: 'job-grade-one-side',
        coinId: 42,
        jobType: 'coin_grading',
        status: 'queued',
      },
    })
    mocks.getAIJob.mockResolvedValueOnce({
      data: {
        id: 'job-grade-one-side',
        coinId: 42,
        jobType: 'coin_grading',
        status: 'completed',
        result: { gradingReport: 'Estimated grade: Fine. Limited by missing reverse image.' },
        createdAt: '',
        updatedAt: '',
      },
    })

    const wrapper = mountAnalysis({ hasReverse: false })
    await flushPromises()

    const gradeButton = findActionButton(wrapper, 'Grade coin')
    expect(gradeButton.attributes('disabled')).toBeUndefined()
    await gradeButton.trigger('click')
    await flushPromises()

    expect(mocks.gradeCoin).toHaveBeenCalledWith(42)
    expect(wrapper.text()).toContain('Limited by missing reverse image')
  })

  it('submits a grading job, polls, and displays the completed report', async () => {
    vi.useFakeTimers()
    mocks.gradeCoin.mockResolvedValue({
      data: {
        jobId: 'job-grade',
        coinId: 42,
        jobType: 'coin_grading',
        status: 'queued',
      },
    })
    mocks.getAIJob
      .mockResolvedValueOnce({
        data: {
          id: 'job-grade',
          coinId: 42,
          jobType: 'coin_grading',
          status: 'running',
          createdAt: '',
          updatedAt: '',
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 'job-grade',
          coinId: 42,
          jobType: 'coin_grading',
          status: 'completed',
          result: {
            gradingReport: 'Estimated grade: VF. Moderate wear with clear major details.',
          },
          createdAt: '',
          updatedAt: '',
        },
      })

    const wrapper = mountAnalysis()
    await flushPromises()
    await findActionButton(wrapper, 'Grade coin').trigger('click')
    await flushPromises()

    expect(mocks.gradeCoin).toHaveBeenCalledWith(42)
    expect(wrapper.text()).toContain('Coin grading in progress.')

    await vi.advanceTimersByTimeAsync(3_000)
    await flushPromises()

    expect(wrapper.text()).toContain('Grading Report')
    expect(wrapper.text()).toContain('Estimated grade: VF')
    expect(wrapper.text()).toContain('saved coin grade is not changed automatically')
    expect(mocks.refreshNotifications).toHaveBeenCalled()
    expect(sessionStorage.getItem('aiJob:grading:42')).toBeNull()
  })

  it('recovers the most recent completed grading report without session storage', async () => {
    mocks.getCoinAIJobs
      .mockResolvedValueOnce({ data: [] })
      .mockResolvedValueOnce({
        data: [
          {
            id: 'old-grade',
            coinId: 42,
            jobType: 'coin_grading',
            status: 'completed',
            result: { gradingReport: 'Estimated grade: F. Older report.' },
            createdAt: '2026-07-01T10:00:00Z',
            updatedAt: '2026-07-01T10:01:00Z',
            completedAt: '2026-07-01T10:01:00Z',
          },
          {
            id: 'new-grade',
            coinId: 42,
            jobType: 'coin_grading',
            status: 'completed',
            result: { gradingReport: 'Estimated grade: VF. Newest completed report.' },
            createdAt: '2026-07-02T10:00:00Z',
            updatedAt: '2026-07-02T10:01:00Z',
            completedAt: '2026-07-02T10:01:00Z',
          },
        ],
      })

    const wrapper = mountAnalysis()
    await flushPromises()

    expect(sessionStorage.getItem('aiJob:grading:42')).toBeNull()
    expect(mocks.getCoinAIJobs).toHaveBeenNthCalledWith(1, 42, true)
    expect(mocks.getCoinAIJobs).toHaveBeenNthCalledWith(2, 42, false)
    expect(wrapper.text()).toContain('Grading Report')
    expect(wrapper.text()).toContain('Estimated grade: VF')
    expect(wrapper.text()).not.toContain('Older report')
  })

  it('shows grading job failures without changing saved analysis', async () => {
    mocks.gradeCoin.mockResolvedValue({
      data: {
        jobId: 'job-grade-failed',
        coinId: 42,
        jobType: 'coin_grading',
        status: 'queued',
      },
    })
    mocks.getAIJob.mockResolvedValueOnce({
      data: {
        id: 'job-grade-failed',
        coinId: 42,
        jobType: 'coin_grading',
        status: 'failed',
        errorMessage: 'Coin images are required for grading',
        createdAt: '',
        updatedAt: '',
      },
    })

    const wrapper = mountAnalysis()
    await flushPromises()
    await findActionButton(wrapper, 'Grade coin').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Coin images are required for grading')
    expect(mocks.showAlert).toHaveBeenCalledWith('Coin images are required for grading', { title: 'Grading Failed' })
    expect(wrapper.emitted('analysisUpdated')).toBeUndefined()
  })

  it('shows a failed job status and does not emit refresh', async () => {
    mocks.analyzeCoin.mockResolvedValue({
      data: {
        jobId: 'job-failed',
        coinId: 42,
        jobType: 'coin_analysis',
        side: 'reverse',
        status: 'queued',
      },
    })
    mocks.getAIJob.mockResolvedValueOnce({
      data: {
        id: 'job-failed',
        coinId: 42,
        jobType: 'coin_analysis',
        side: 'reverse',
        status: 'failed',
        errorMessage: 'Analysis could not complete',
        createdAt: '',
        updatedAt: '',
      },
    })

    const wrapper = mountAnalysis()
    await flushPromises()
    await findActionButton(wrapper, 'Analyze reverse').trigger('click')
    await flushPromises()

    expect(mocks.showAlert).toHaveBeenCalledWith('Analysis could not complete', { title: 'Analysis Failed' })
    expect(wrapper.emitted('analysisUpdated')).toBeUndefined()
  })

  it('cleans up polling when unmounted', async () => {
    vi.useFakeTimers()
    mocks.analyzeCoin.mockResolvedValue({
      data: {
        jobId: 'job-cleanup',
        coinId: 42,
        jobType: 'coin_analysis',
        side: 'obverse',
        status: 'queued',
      },
    })
    mocks.getAIJob.mockResolvedValue({
      data: {
        id: 'job-cleanup',
        coinId: 42,
        jobType: 'coin_analysis',
        side: 'obverse',
        status: 'running',
        createdAt: '',
        updatedAt: '',
      },
    })

    const wrapper = mountAnalysis()
    await flushPromises()
    await findActionButton(wrapper, 'Analyze obverse').trigger('click')
    await flushPromises()

    expect(mocks.getAIJob).toHaveBeenCalledTimes(1)
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(3_000)

    expect(mocks.getAIJob).toHaveBeenCalledTimes(1)
  })
})

function mountAnalysis(propOverrides: Partial<InstanceType<typeof CoinAIAnalysis>['$props']> = {}) {
  return mount(CoinAIAnalysis, {
    props: {
      coinId: 42,
      hasObverse: true,
      hasReverse: true,
      obverseAnalysis: null,
      reverseAnalysis: null,
      aiAnalysis: null,
      ...propOverrides,
    },
  })
}

function findActionButton(wrapper: ReturnType<typeof mountAnalysis>, label: string) {
  const button = wrapper.findAll('button').find((candidate) => candidate.attributes('aria-label') === label)
  if (!button) throw new Error(`Missing action button: ${label}`)
  return button
}
