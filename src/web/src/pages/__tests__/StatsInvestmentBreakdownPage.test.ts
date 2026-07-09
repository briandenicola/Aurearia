import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StatsInvestmentBreakdownPage from '@/pages/StatsInvestmentBreakdownPage.vue'
import type { InvestmentBreakdownSegment } from '@/types'

const mockGetInvestmentBreakdown = vi.fn()
vi.mock('@/api/client', () => ({
  getInvestmentBreakdown: (dimension: string) => mockGetInvestmentBreakdown(dimension),
}))

const defaultStubs = {
  PullToRefresh: { template: '<div><slot /></div>' },
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
  ArrowLeft: { template: '<span />' },
}

function segment(label: string, overrides: Partial<InvestmentBreakdownSegment> = {}): InvestmentBreakdownSegment {
  return {
    label,
    year: null,
    month: null,
    invested: 100,
    currentValue: 125,
    gainLoss: 25,
    gainLossPct: 25,
    coinCount: 1,
    missingCurrentValueCount: 0,
    missingPurchasePriceCount: 0,
    ...overrides,
  }
}

describe('StatsInvestmentBreakdownPage', () => {
  beforeEach(() => {
    mockGetInvestmentBreakdown.mockReset()
    mockGetInvestmentBreakdown.mockImplementation((dimension: string) => {
      if (dimension === 'purchase-year') {
        return Promise.resolve({
          data: {
            segments: [segment('2024', { year: 2024 })],
            topIncreases: [
              {
                coinId: 11,
                name: 'Aureus of Hadrian',
                initialValue: 1000,
                currentValue: 1800,
                changeAmount: 800,
                changePct: 80,
                changeExplanation: 'The value increased because recent comparable sales are stronger.',
              },
            ],
            topDrops: [
              { coinId: 12, name: 'Denarius of Trajan', initialValue: 500, currentValue: 350, changeAmount: -150, changePct: -30 },
            ],
            staleValuations: [
              { coinId: 13, name: 'Stale Sestertius', lastValuationAt: '2025-01-02T12:00:00Z' },
              { coinId: 14, name: 'Never Valued Drachm', lastValuationAt: null },
            ],
          },
        })
      }
      return Promise.resolve({ data: { segments: [segment('Silver')] } })
    })
  })

  it('loads purchase-year and material investment breakdowns', async () => {
    const wrapper = mount(StatsInvestmentBreakdownPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    expect(mockGetInvestmentBreakdown).toHaveBeenCalledWith('purchase-year')
    expect(mockGetInvestmentBreakdown).toHaveBeenCalledWith('material')
    expect(wrapper.text()).toContain('Investment Breakdown')
    expect(wrapper.text()).toContain('Acquisition Performance by Year')
    expect(wrapper.text()).toContain('Material Allocation')
    expect(wrapper.text()).toContain('2024')
    expect(wrapper.text()).toContain('Silver')
  })

  it('renders valuation movement and stale valuation links', async () => {
    const wrapper = mount(StatsInvestmentBreakdownPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Biggest Value Gains')
    expect(wrapper.text()).toContain('Aureus of Hadrian')
    expect(wrapper.text()).toContain('$1,000.00')
    expect(wrapper.text()).toContain('$1,800.00')
    expect(wrapper.text()).toContain('The value increased because recent comparable sales are stronger.')
    expect(wrapper.find('a[href="/coin/11"]').exists()).toBe(true)

    expect(wrapper.text()).toContain('Biggest Value Declines')
    expect(wrapper.text()).toContain('Denarius of Trajan')
    expect(wrapper.text()).toContain('$500.00')
    expect(wrapper.text()).toContain('$350.00')
    expect(wrapper.text()).not.toContain('undefined')
    expect(wrapper.find('a[href="/coin/12"]').exists()).toBe(true)

    expect(wrapper.text()).toContain('Needs Refresh')
    expect(wrapper.text()).toContain('Stale Sestertius')
    expect(wrapper.text()).toContain('Jan 2, 2025')
    expect(wrapper.text()).toContain('Never Valued Drachm')
    expect(wrapper.text()).toContain('Never valued')
    expect(wrapper.find('a[href="/coin/13/actions"]').exists()).toBe(true)
    expect(wrapper.find('a[href="/coin/14/actions"]').exists()).toBe(true)
  })

  it('renders confidence callouts from missing-value counts returned by the API', async () => {
    mockGetInvestmentBreakdown.mockImplementation((dimension: string) => {
      if (dimension === 'purchase-year') {
        return Promise.resolve({
          data: {
            segments: [
              segment('2024', {
                year: 2024,
                missingCurrentValueCount: 2,
                missingPurchasePriceCount: 1,
              }),
            ],
          },
        })
      }
      return Promise.resolve({ data: { segments: [segment('Silver')] } })
    })

    const wrapper = mount(StatsInvestmentBreakdownPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    const confidenceCallouts = wrapper.findAll('.confidence-callout')
    expect(confidenceCallouts).toHaveLength(1)
    expect(confidenceCallouts[0]?.text()).toContain('1 missing purchase price')
    expect(confidenceCallouts[0]?.text()).toContain('2 missing current value')
  })

  it('renders an icon back button to Stats', async () => {
    const wrapper = mount(StatsInvestmentBreakdownPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    expect(wrapper.find('a[href="/stats"]').exists()).toBe(true)
  })

  it('shows an error card if breakdown loading fails', async () => {
    mockGetInvestmentBreakdown.mockRejectedValue(new Error('network'))
    const wrapper = mount(StatsInvestmentBreakdownPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    expect(wrapper.find('.error-card').exists()).toBe(true)
    expect(wrapper.text()).toContain('Investment breakdown data could not be loaded.')
  })
})
