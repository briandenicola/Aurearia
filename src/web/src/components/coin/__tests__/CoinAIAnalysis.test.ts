import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CoinAIAnalysis from '../CoinAIAnalysis.vue'

const mocks = vi.hoisted(() => ({
  analyzeCoin: vi.fn(),
  deleteAnalysis: vi.fn(),
  getAIStatus: vi.fn(),
  showAlert: vi.fn(),
  showConfirm: vi.fn(),
}))

vi.mock('@/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/client')>()
  return {
    ...actual,
    analyzeCoin: mocks.analyzeCoin,
    deleteAnalysis: mocks.deleteAnalysis,
    getAIStatus: mocks.getAIStatus,
  }
})

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showAlert: mocks.showAlert,
    showConfirm: mocks.showConfirm,
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
