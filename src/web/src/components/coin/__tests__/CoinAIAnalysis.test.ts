import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import CoinAIAnalysis from '../CoinAIAnalysis.vue'

const mocks = vi.hoisted(() => ({
  analyzeCoin: vi.fn(),
  deleteAnalysis: vi.fn(),
  getAIJob: vi.fn(),
  getAIStatus: vi.fn(),
  getCoinAIJobs: vi.fn(),
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
    await wrapper.findAll('button').find((button) => button.text().includes('Analyze Obverse'))!.trigger('click')
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
    await wrapper.findAll('button').find((button) => button.text().includes('Analyze Obverse'))!.trigger('click')
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
    await wrapper.findAll('button').find((button) => button.text().includes('Analyze Obverse'))!.trigger('click')
    await flushPromises()

    expect(mocks.analyzeCoin).toHaveBeenCalledWith(42, 'obverse')
    expect(wrapper.text()).toContain('Obverse analysis in progress.')

    await vi.advanceTimersByTimeAsync(3_000)
    await flushPromises()

    expect(wrapper.emitted('analysisUpdated')).toHaveLength(1)
    expect(mocks.refreshNotifications).toHaveBeenCalled()
    expect(sessionStorage.getItem('aiJob:analysis:42:obverse')).toBeNull()
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
    await wrapper.findAll('button').find((button) => button.text().includes('Analyze Reverse'))!.trigger('click')
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
    await wrapper.findAll('button').find((button) => button.text().includes('Analyze Obverse'))!.trigger('click')
    await flushPromises()

    expect(mocks.getAIJob).toHaveBeenCalledTimes(1)
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(3_000)

    expect(mocks.getAIJob).toHaveBeenCalledTimes(1)
  })
})

function mountAnalysis() {
  return mount(CoinAIAnalysis, {
    props: {
      coinId: 42,
      hasObverse: true,
      hasReverse: true,
      obverseAnalysis: null,
      reverseAnalysis: null,
      aiAnalysis: null,
    },
  })
}
