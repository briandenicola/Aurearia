import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import StatsTimeMachinePage from '@/pages/StatsTimeMachinePage.vue'
import type { TimeMachineBounds, TimeMachineSnapshot } from '@/types'

const getTimeMachineBounds = vi.fn()
const getTimeMachineSnapshot = vi.fn()

vi.mock('@/api/client', () => ({
  getTimeMachineBounds: (...args: unknown[]) => getTimeMachineBounds(...args),
  getTimeMachineSnapshot: (...args: unknown[]) => getTimeMachineSnapshot(...args),
  getApiErrorMessage: (err: unknown) => (err instanceof Error ? err.message : ''),
}))

const stubs = {
  'router-link': { props: ['to'], template: '<a :href="to"><slot /></a>' },
  ArrowLeft: true,
}

function bounds(overrides: Partial<TimeMachineBounds> = {}): TimeMachineBounds {
  return { earliestDate: '2020-01-01', latestDate: '2026-08-20', hasData: true, ...overrides }
}

function snapshot(overrides: Partial<TimeMachineSnapshot> = {}): TimeMachineSnapshot {
  return {
    asOfDate: '2026-08-20',
    coinCount: 3,
    totalValue: 1000,
    totalInvested: 700,
    unrealizedGain: 300,
    byCategory: [{ label: 'Roman', count: 2, value: 800 }, { label: 'Greek', count: 1, value: 200 }],
    byMaterial: [{ label: 'Silver', count: 3, value: 1000 }],
    byEra: [{ label: 'ancient', count: 3, value: 1000 }],
    topCoins: [{ id: 7, name: 'Trajan denarius', value: 600, valueFromHistory: true }],
    acquiredInYear: 1,
    healthScore: 72,
    valueBasis: { fromValuationHistory: 3, fromPurchasePrice: 0 },
    undatedCoinCount: 0,
    ...overrides,
  }
}

async function mountPage() {
  const wrapper = mount(StatsTimeMachinePage, { global: { stubs } })
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  vi.useFakeTimers({ shouldAdvanceTime: true })
  getTimeMachineBounds.mockReset()
  getTimeMachineSnapshot.mockReset()
  getTimeMachineBounds.mockResolvedValue({ data: bounds() })
  getTimeMachineSnapshot.mockResolvedValue({ data: snapshot() })
})

afterEach(() => {
  vi.useRealTimers()
})

describe('StatsTimeMachinePage', () => {
  it('opens on the latest date and renders that snapshot', async () => {
    const wrapper = await mountPage()

    expect(getTimeMachineSnapshot).toHaveBeenCalledWith('2026-08-20')
    expect(wrapper.text()).toContain('Coins Owned')
    expect(wrapper.text()).toContain('Trajan denarius')
  })

  it('explains itself instead of rendering a scrubber when nothing is dated', async () => {
    getTimeMachineBounds.mockResolvedValue({ data: bounds({ hasData: false }) })

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('No dated acquisitions yet')
    expect(wrapper.find('input[type="range"]').exists()).toBe(false)
    expect(getTimeMachineSnapshot).not.toHaveBeenCalled()
  })

  it('debounces scrubbing into a single request', async () => {
    const wrapper = await mountPage()
    getTimeMachineSnapshot.mockClear()

    const slider = wrapper.find('input[type="range"]')
    for (const value of ['100', '200', '300', '400']) {
      await slider.setValue(value)
      await slider.trigger('input')
    }

    // Nothing fires while the user is still dragging.
    expect(getTimeMachineSnapshot).not.toHaveBeenCalled()

    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(getTimeMachineSnapshot).toHaveBeenCalledTimes(1)
  })

  it('says which coins fall back to purchase price', async () => {
    getTimeMachineSnapshot.mockResolvedValue({
      data: snapshot({ valueBasis: { fromValuationHistory: 1, fromPurchasePrice: 4 } }),
    })

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('4 of 5 coins had no recorded valuation')
  })

  it('states plainly when no valuations existed at all', async () => {
    getTimeMachineSnapshot.mockResolvedValue({
      data: snapshot({ valueBasis: { fromValuationHistory: 0, fromPurchasePrice: 5 } }),
    })

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('No valuations had been recorded by this date')
  })

  it('discloses undated coins rather than silently omitting them', async () => {
    getTimeMachineSnapshot.mockResolvedValue({ data: snapshot({ undatedCoinCount: 2 }) })

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('2 coins have')
    expect(wrapper.text()).toContain('no purchase date')
  })

  it('reports owning nothing on a date before the first acquisition', async () => {
    getTimeMachineSnapshot.mockResolvedValue({
      data: snapshot({ coinCount: 0, totalValue: 0, byCategory: [], byMaterial: [], byEra: [], topCoins: [] }),
    })

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('You owned no coins on this date')
  })

  it('surfaces a bounds failure with a retry', async () => {
    getTimeMachineBounds.mockRejectedValue(new Error('network down'))

    const wrapper = await mountPage()

    expect(wrapper.text()).toContain('network down')
    expect(wrapper.find('button').text()).toBe('Try again')
  })

  // Dragging quickly can leave an older request in flight; it must not
  // overwrite the newer snapshot when it eventually resolves.
  it('ignores a stale in-flight response', async () => {
    const wrapper = await mountPage()
    getTimeMachineSnapshot.mockClear()

    let resolveSlow: (value: unknown) => void = () => {}
    const slow = new Promise((resolve) => { resolveSlow = resolve })

    getTimeMachineSnapshot.mockReturnValueOnce(slow)
    getTimeMachineSnapshot.mockResolvedValueOnce({
      data: snapshot({ coinCount: 99, topCoins: [{ id: 1, name: 'Newest result', value: 1, valueFromHistory: true }] }),
    })

    const slider = wrapper.find('input[type="range"]')
    await slider.setValue('100')
    await slider.trigger('input')
    vi.advanceTimersByTime(300)
    await flushPromises()

    await slider.setValue('200')
    await slider.trigger('input')
    vi.advanceTimersByTime(300)
    await flushPromises()

    // The first request finally lands, carrying outdated data.
    resolveSlow({
      data: snapshot({ coinCount: 1, topCoins: [{ id: 2, name: 'Stale result', value: 1, valueFromHistory: true }] }),
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Newest result')
    expect(wrapper.text()).not.toContain('Stale result')
  })
})
