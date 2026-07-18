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
  it('renders summary values and responsive investment rows', () => {
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
    expect(wrapper.text()).toContain('ROI')
    expect(wrapper.text()).toContain('5 coins')
    expect(wrapper.findAll('article')).toHaveLength(2)
    expect(wrapper.findAll('.bg-gold')).toHaveLength(2)
  })

  it('formats purchase year labels', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Acquisition Performance by Year',
        eyebrow: 'Purchase Timing',
        rows: [
          segment({ label: '2024', year: 2024, invested: 150, currentValue: 175, coinCount: 1 }),
        ],
      },
    })

    expect(wrapper.text()).toContain('2024')
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

    expect(wrapper.find('[role="note"]').exists()).toBe(true)
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
    expect(wrapper.find('article').exists()).toBe(false)
  })

  it('renders aggregate summary with correct values', () => {
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

    const summary = wrapper.find('[aria-label="Investment summary"]')
    expect(summary.exists()).toBe(true)
    expect(summary.text()).toContain('$500.00')
    expect(summary.text()).toContain('$540.00')
    expect(summary.text()).toContain('+$40.00')
    expect(summary.text()).toContain('+8.0%')
  })

  it('renders row list in DOM', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [
          segment({ label: 'Silver', invested: 300, currentValue: 360, gainLoss: 60, coinCount: 3 }),
        ],
      },
    })

    expect(wrapper.find('[aria-label="Investment performance rows"]').exists()).toBe(true)
    expect(wrapper.findAll('article')).toHaveLength(1)
  })

  it('displays negative values correctly in aggregate summary', () => {
    const wrapper = mount(StatsInvestmentBreakdownChart, {
      props: {
        title: 'Material',
        eyebrow: 'Portfolio Composition',
        rows: [
          segment({ label: 'Bronze', invested: 500, currentValue: 400, gainLoss: -100, gainLossPct: -20, coinCount: 5 }),
        ],
      },
    })

    const summary = wrapper.find('[aria-label="Investment summary"]')
    expect(summary.text()).toContain('$500.00')
    expect(summary.text()).toContain('$400.00')
    expect(summary.text()).toContain('-$100.00')
    expect(summary.text()).toContain('-20.0%')
    expect(summary.find('.text-byzantine').exists()).toBe(true)
  })
})
