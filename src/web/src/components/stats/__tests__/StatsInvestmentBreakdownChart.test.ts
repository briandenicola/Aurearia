import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StatsInvestmentBreakdownChart from '@/components/stats/StatsInvestmentBreakdownChart.vue'
import type { InvestmentBreakdownSegment } from '@/types'

function segment(overrides: Partial<InvestmentBreakdownSegment>): InvestmentBreakdownSegment {
  return {
    label: 'Silver',
    year: null,
    month: null,
    invested: 100,
    currentValue: 125,
    gainLoss: 25,
    gainLossPct: 25,
    coinCount: 2,
    missingCurrentValueCount: 0,
    missingPurchasePriceCount: 0,
    ...overrides,
  }
}

describe('StatsInvestmentBreakdownChart', () => {
  it('renders summary values, legend, flow SVG, and segment cards', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [
          segment({ label: 'Silver', invested: 300, currentValue: 360, gainLoss: 60, coinCount: 3 }),
          segment({ label: 'Bronze', invested: 200, currentValue: 180, gainLoss: -20, gainLossPct: -10, coinCount: 2 }),
        ],
      },
    })

    expect(wrapper.text()).toContain('Material')
    expect(wrapper.text()).toContain('Invested')
    expect(wrapper.text()).toContain('Current Value')
    expect(wrapper.text()).toContain('Return')
    expect(wrapper.text()).toContain('5 coins')
    expect(wrapper.find('.investment-flow-svg').exists()).toBe(true)
    expect(wrapper.findAll('.flow-band')).toHaveLength(2)
    expect(wrapper.findAll('.segment-card')).toHaveLength(2)
  })

  it('formats purchase year and month labels', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Purchase Year to Month',
        eyebrow: 'Acquisition Timing',
        rows: [
          segment({ label: '2024-01', year: 2024, month: 1, invested: 150, currentValue: 175, coinCount: 1 }),
        ],
      },
    })

    expect(wrapper.text()).toContain('2024 Jan')
  })

  it('shows confidence callout when price or value data is missing', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [
          segment({ missingPurchasePriceCount: 1, missingCurrentValueCount: 2 }),
        ],
      },
    })

    expect(wrapper.find('.confidence-callout').exists()).toBe(true)
    expect(wrapper.text()).toContain('1 missing purchase price')
    expect(wrapper.text()).toContain('2 missing current value')
  })

  it('shows an empty state when no rows are provided', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [],
      },
    })

    expect(wrapper.find('.empty-state').exists()).toBe(true)
    expect(wrapper.find('.investment-flow-svg').exists()).toBe(false)
  })

  it('renders mobile aggregate summary with correct values', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [
          segment({ label: 'Silver', invested: 300, currentValue: 360, gainLoss: 60, gainLossPct: 20, coinCount: 3 }),
          segment({ label: 'Bronze', invested: 200, currentValue: 180, gainLoss: -20, gainLossPct: -10, coinCount: 2 }),
        ],
      },
    })

    const mobileSummary = wrapper.find('.mobile-aggregate-summary')
    expect(mobileSummary.exists()).toBe(true)
    expect(mobileSummary.text()).toContain('Invested: $500.00')
    expect(mobileSummary.text()).toContain('Current: $540.00')
    expect(mobileSummary.text()).toContain('Gain/Loss: +$40.00')
    expect(mobileSummary.text()).toContain('(+8.0%)')
  })

  it('renders both mobile aggregate summary and segment list in DOM', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [
          segment({ label: 'Silver', invested: 300, currentValue: 360, gainLoss: 60, coinCount: 3 }),
        ],
      },
    })

    expect(wrapper.find('.mobile-aggregate-summary').exists()).toBe(true)
    expect(wrapper.find('.segment-list').exists()).toBe(true)
    expect(wrapper.findAll('.segment-card')).toHaveLength(1)
  })

  it('displays negative values correctly in mobile aggregate summary', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [
          segment({ label: 'Bronze', invested: 500, currentValue: 400, gainLoss: -100, gainLossPct: -20, coinCount: 5 }),
        ],
      },
    })

    const mobileSummary = wrapper.find('.mobile-aggregate-summary')
    expect(mobileSummary.text()).toContain('Invested: $500.00')
    expect(mobileSummary.text()).toContain('Current: $400.00')
    expect(mobileSummary.text()).toContain('Gain/Loss: -$100.00')
    expect(mobileSummary.text()).toContain('(-20.0%)')
    expect(mobileSummary.find('.negative').exists()).toBe(true)
  })
})
