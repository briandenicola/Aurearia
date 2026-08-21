import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import CoinTagsSection from '../CoinTagsSection.vue'
import { getCoinRecommendations, getTags, getSets } from '@/api/client'
import type { CoinRecommendation } from '@/types'

vi.mock('@/api/client', () => ({
  getTags: vi.fn(),
  getSets: vi.fn(),
  getCoinRecommendations: vi.fn(),
  addTagToCoin: vi.fn(),
  removeTagFromCoin: vi.fn(),
  addCoinToSet: vi.fn(),
  removeCoinFromSet: vi.fn(),
  acceptCoinRecommendation: vi.fn(),
  rejectCoinRecommendation: vi.fn(),
}))

const defaultProps = {
  tags: [],
  sets: [],
  coinId: 42,
}

function makeRecommendation(overrides: Partial<CoinRecommendation> = {}): CoinRecommendation {
  return {
    id: 1,
    targetType: 'tag',
    targetId: 10,
    targetName: 'Roman Bronze',
    score: 0.8,
    confidence: 'high',
    reasons: ['Same ruler', 'Same category'],
    status: 'pending',
    ...overrides,
  }
}

describe('CoinTagsSection recommendations', () => {
  beforeEach(() => {
    vi.mocked(getTags).mockResolvedValue({ data: { tags: [] } } as never)
    vi.mocked(getSets).mockResolvedValue({ data: { sets: [] } } as never)
  })

  it('shows loading while fetching recommendations', async () => {
    // Simulate a never-resolving request to observe the loading state
    vi.mocked(getCoinRecommendations).mockImplementation(() => new Promise(() => {}))
    const wrapper = mount(CoinTagsSection, { props: defaultProps })
    // Allow onMounted microtask to run so loadRecommendations sets loading=true
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('Loading suggestions...')
  })

  it('shows recommendations on successful load', async () => {
    vi.mocked(getCoinRecommendations).mockResolvedValue({
      data: { recommendations: [makeRecommendation()] },
    } as never)
    const wrapper = mount(CoinTagsSection, { props: defaultProps })
    await flushPromises()
    expect(wrapper.text()).toContain('Roman Bronze')
    expect(wrapper.text()).not.toContain('Loading suggestions...')
    expect(wrapper.text()).not.toContain('No suggestions yet')
  })

  it('shows "No suggestions yet" when server returns an empty list', async () => {
    vi.mocked(getCoinRecommendations).mockResolvedValue({
      data: { recommendations: [] },
    } as never)
    const wrapper = mount(CoinTagsSection, { props: defaultProps })
    await flushPromises()
    expect(wrapper.text()).toContain('No suggestions yet')
    // Error path is not active — no retry button
    const retryBtn = wrapper.findAll('button').find((b) => b.text().toLowerCase() === 'retry')
    expect(retryBtn).toBeUndefined()
  })

  it('shows an error message distinct from "No suggestions yet" when the API call fails', async () => {
    vi.mocked(getCoinRecommendations).mockRejectedValue(new Error('Network Error'))
    const wrapper = mount(CoinTagsSection, { props: defaultProps })
    await flushPromises()
    expect(wrapper.text()).not.toContain('No suggestions yet')
    expect(wrapper.text()).toContain('Could not load suggestions')
  })

  it('shows a Retry button when recommendations fail to load', async () => {
    vi.mocked(getCoinRecommendations).mockRejectedValue(new Error('500'))
    const wrapper = mount(CoinTagsSection, { props: defaultProps })
    await flushPromises()
    const retryBtn = wrapper.findAll('button').find((b) => b.text().toLowerCase() === 'retry')
    expect(retryBtn).toBeDefined()
    expect(retryBtn!.exists()).toBe(true)
  })

  it('clears error and shows recommendations after a successful retry', async () => {
    vi.mocked(getCoinRecommendations)
      .mockRejectedValueOnce(new Error('Network Error'))
      .mockResolvedValueOnce({ data: { recommendations: [makeRecommendation()] } } as never)

    const wrapper = mount(CoinTagsSection, { props: defaultProps })
    await flushPromises()
    expect(wrapper.text()).toContain('Could not load suggestions')

    const retryBtn = wrapper.findAll('button').find((b) => b.text().toLowerCase() === 'retry')
    await retryBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('Could not load suggestions')
    expect(wrapper.text()).toContain('Roman Bronze')
  })

  it('shows loading state while a retry is in progress', async () => {
    let resolve!: (v: unknown) => void
    vi.mocked(getCoinRecommendations)
      .mockRejectedValueOnce(new Error('Network Error'))
      .mockImplementationOnce(() => new Promise((r) => { resolve = r }))

    const wrapper = mount(CoinTagsSection, { props: defaultProps })
    await flushPromises()

    const retryBtn = wrapper.findAll('button').find((b) => b.text().toLowerCase() === 'retry')
    await retryBtn!.trigger('click')
    // Do not flush — the retry request is still in flight
    expect(wrapper.text()).toContain('Loading suggestions...')

    // Cleanup: resolve the pending promise to avoid open handles
    resolve({ data: { recommendations: [] } })
    await flushPromises()
  })
})
