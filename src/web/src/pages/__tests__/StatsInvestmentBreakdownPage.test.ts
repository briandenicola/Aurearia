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
      if (dimension === 'purchase-month') {
        return Promise.resolve({ data: [segment('2024-01', { year: 2024, month: 1 })] })
      }
      return Promise.resolve({ data: { segments: [segment('Silver')] } })
    })
  })

  it('loads purchase-month and material investment breakdowns', async () => {
    const wrapper = mount(StatsInvestmentBreakdownPage, {
      global: { stubs: defaultStubs },
    })
    await flushPromises()

    expect(mockGetInvestmentBreakdown).toHaveBeenCalledWith('purchase-month')
    expect(mockGetInvestmentBreakdown).toHaveBeenCalledWith('material')
    expect(wrapper.text()).toContain('Investment Breakdown')
    expect(wrapper.text()).toContain('Purchase Year to Month')
    expect(wrapper.text()).toContain('Material')
    expect(wrapper.text()).toContain('2024 Jan')
    expect(wrapper.text()).toContain('Silver')
  })

  it('renders confidence callouts from missing-value counts returned by the API', async () => {
    mockGetInvestmentBreakdown.mockImplementation((dimension: string) => {
      if (dimension === 'purchase-month') {
        return Promise.resolve({
          data: {
            segments: [
              segment('2024-01', {
                year: 2024,
                month: 1,
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
