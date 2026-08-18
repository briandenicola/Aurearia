import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { useCoinValueEstimate } from '../useCoinValueEstimate'
import { estimateCoinValue, getAIJob, getCoinAIJobs, updateCoin } from '@/api/client'

vi.mock('@/api/client', () => ({
  estimateCoinValue: vi.fn(),
  getAIJob: vi.fn(),
  getCoinAIJobs: vi.fn().mockResolvedValue({ data: [] }),
  updateCoin: vi.fn(),
}))

const showAlert = vi.fn()
const refreshNotifications = vi.fn()
const showToast = vi.fn()

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showAlert }),
}))

vi.mock('@/composables/useNotifications', () => ({
  useNotifications: () => ({ refresh: refreshNotifications }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ showToast }),
}))

function mountEstimate(coinId = 42, onApplied = vi.fn()) {
  let api!: ReturnType<typeof useCoinValueEstimate>
  const host = defineComponent({
    setup() {
      api = useCoinValueEstimate(coinId, { onApplied })
      return () => h('div')
    },
  })
  const wrapper = mount(host)
  return { wrapper, get api() { return api }, onApplied }
}

const terminalJob = (overrides: Record<string, unknown> = {}) => ({
  id: 'job-1',
  coinId: 42,
  jobType: 'value_estimate',
  status: 'completed',
  result: { estimatedValue: 500, confidence: 'high', reasoning: 'Nice coin', comparables: [] },
  createdAt: '',
  updatedAt: '',
  ...overrides,
})

describe('useCoinValueEstimate', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getCoinAIJobs).mockResolvedValue({ data: [] } as Awaited<ReturnType<typeof getCoinAIJobs>>)
    sessionStorage.clear()
  })

  it('resolves an immediately-completed estimate job and shows the result', async () => {
    vi.mocked(estimateCoinValue).mockResolvedValue({
      data: { id: 'job-1', status: 'queued', jobType: 'value_estimate', coinId: 42 },
    } as Awaited<ReturnType<typeof estimateCoinValue>>)
    vi.mocked(getAIJob).mockResolvedValue({ data: terminalJob() } as Awaited<ReturnType<typeof getAIJob>>)

    const { wrapper, api } = mountEstimate()
    await flushPromises()

    await api.handleEstimateValue()
    await flushPromises()

    expect(api.estimating.value).toBe(false)
    expect(api.valueEstimate.value?.estimatedValue).toBe(500)
    expect(api.valueEstimate.value?.confidence).toBe('high')
    expect(showToast).toHaveBeenCalledWith('Value estimate ready.', 'success')
    wrapper.unmount()
  })

  it('surfaces a failed job with its error message', async () => {
    vi.mocked(estimateCoinValue).mockResolvedValue({
      data: { id: 'job-2', status: 'queued', jobType: 'value_estimate', coinId: 42 },
    } as Awaited<ReturnType<typeof estimateCoinValue>>)
    vi.mocked(getAIJob).mockResolvedValue({
      data: terminalJob({ id: 'job-2', status: 'failed', errorMessage: 'Provider timeout', result: undefined }),
    } as Awaited<ReturnType<typeof getAIJob>>)

    const { wrapper, api } = mountEstimate()
    await flushPromises()

    await api.handleEstimateValue()
    await flushPromises()

    expect(api.estimating.value).toBe(false)
    expect(api.estimateError.value).toBe('Provider timeout')
    expect(api.valueEstimate.value).toBeNull()
    wrapper.unmount()
  })

  it('resumes an in-flight estimate job on mount from the active-jobs endpoint', async () => {
    vi.mocked(getCoinAIJobs).mockResolvedValue({
      data: [{ id: 'job-3', coinId: 42, jobType: 'value_estimate', status: 'running', createdAt: '', updatedAt: '' }],
    } as Awaited<ReturnType<typeof getCoinAIJobs>>)
    vi.mocked(getAIJob).mockResolvedValue({ data: terminalJob({ id: 'job-3' }) } as Awaited<ReturnType<typeof getAIJob>>)

    const { wrapper, api } = mountEstimate()
    await flushPromises()

    expect(api.valueEstimate.value?.estimatedValue).toBe(500)
    wrapper.unmount()
  })

  it('applies the estimate to the coin and calls onApplied', async () => {
    vi.mocked(estimateCoinValue).mockResolvedValue({
      data: { id: 'job-4', status: 'queued', jobType: 'value_estimate', coinId: 42 },
    } as Awaited<ReturnType<typeof estimateCoinValue>>)
    vi.mocked(getAIJob).mockResolvedValue({ data: terminalJob({ id: 'job-4' }) } as Awaited<ReturnType<typeof getAIJob>>)
    vi.mocked(updateCoin).mockResolvedValue({} as Awaited<ReturnType<typeof updateCoin>>)
    const onApplied = vi.fn()

    const { wrapper, api } = mountEstimate(42, onApplied)
    await flushPromises()
    await api.handleEstimateValue()
    await flushPromises()

    await api.handleApplyEstimate()

    expect(updateCoin).toHaveBeenCalledWith(42, { currentValue: 500 }, { source: 'estimate' })
    expect(api.valueEstimate.value).toBeNull()
    expect(onApplied).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('shows an alert and keeps the estimate when applying fails', async () => {
    vi.mocked(estimateCoinValue).mockResolvedValue({
      data: { id: 'job-5', status: 'queued', jobType: 'value_estimate', coinId: 42 },
    } as Awaited<ReturnType<typeof estimateCoinValue>>)
    vi.mocked(getAIJob).mockResolvedValue({ data: terminalJob({ id: 'job-5' }) } as Awaited<ReturnType<typeof getAIJob>>)
    vi.mocked(updateCoin).mockRejectedValue(new Error('boom'))
    const onApplied = vi.fn()

    const { wrapper, api } = mountEstimate(42, onApplied)
    await flushPromises()
    await api.handleEstimateValue()
    await flushPromises()

    await api.handleApplyEstimate()

    expect(showAlert).toHaveBeenCalledWith('Failed to update coin value', { title: 'Error' })
    expect(api.valueEstimate.value).not.toBeNull()
    expect(onApplied).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('dismisses the estimate without applying it', async () => {
    vi.mocked(estimateCoinValue).mockResolvedValue({
      data: { id: 'job-6', status: 'queued', jobType: 'value_estimate', coinId: 42 },
    } as Awaited<ReturnType<typeof estimateCoinValue>>)
    vi.mocked(getAIJob).mockResolvedValue({ data: terminalJob({ id: 'job-6' }) } as Awaited<ReturnType<typeof getAIJob>>)

    const { wrapper, api } = mountEstimate()
    await flushPromises()
    await api.handleEstimateValue()
    await flushPromises()

    api.dismissEstimate()

    expect(api.valueEstimate.value).toBeNull()
    expect(updateCoin).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
