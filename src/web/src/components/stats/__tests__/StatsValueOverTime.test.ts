import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import StatsValueOverTime from '@/components/stats/StatsValueOverTime.vue'
import type { ValueSnapshot } from '@/types'

// Minimal store mock — component prefers props.history over store.valueHistory,
// but it still calls useCoinsStore(), so the mock must exist.
vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => ({ valueHistory: [] }),
}))

const jan2024: ValueSnapshot = {
  id: 1, userId: 1, totalValue: 1000, totalInvested: 900, coinCount: 8,
  recordedAt: '2024-01-01T00:00:00Z',
}
const feb2024: ValueSnapshot = {
  id: 2, userId: 1, totalValue: 1350, totalInvested: 950, coinCount: 10,
  recordedAt: '2024-02-01T00:00:00Z',
}
const dec2023: ValueSnapshot = {
  id: 3, userId: 1, totalValue: 800, totalInvested: 850, coinCount: 7,
  recordedAt: '2023-12-01T00:00:00Z',
}

describe('StatsValueOverTime', () => {
  // ── Minimum data guard ──────────────────────────────────────────────────

  it('renders nothing when fewer than 2 history snapshots are provided', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024] } })
    expect(wrapper.find('.value-chart-card').exists()).toBe(false)
  })

  it('renders nothing with an empty history array', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [] } })
    expect(wrapper.find('.value-chart-card').exists()).toBe(false)
  })

  // ── Chart anatomy ───────────────────────────────────────────────────────

  it('renders full chart anatomy when 2 or more snapshots are provided', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })

    expect(wrapper.find('.value-chart-card').exists()).toBe(true)
    expect(wrapper.find('.chart-summary-strip').exists()).toBe(true)
    expect(wrapper.findAll('.summary-pill')).toHaveLength(3)
    expect(wrapper.find('.chart-area-fill').exists()).toBe(true)
    expect(wrapper.find('.chart-line-value').exists()).toBe(true)
    expect(wrapper.find('.chart-line-invested').exists()).toBe(true)
    expect(wrapper.find('.endpoint-dot-value').exists()).toBe(true)
    expect(wrapper.find('.endpoint-dot-invested').exists()).toBe(true)
    expect(wrapper.findAll('.chart-grid-line').length).toBeGreaterThan(0)
  })

  it('shows Portfolio Trajectory label, Value Over Time heading, and timeframe explanation', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })
    expect(wrapper.text()).toContain('Portfolio Trajectory')
    expect(wrapper.text()).toContain('Value Over Time')
    expect(wrapper.text()).toContain('Active collection value movement')
    expect(wrapper.text()).toContain('Period Value Change')
  })

  it('shows legend items for Current Value and Invested', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })
    expect(wrapper.text()).toContain('Current Value')
    expect(wrapper.text()).toContain('Invested')
  })

  // ── Summary strip values ────────────────────────────────────────────────

  it('summary strip first pill shows latest total value', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })
    const pills = wrapper.findAll('.summary-pill')
    expect(pills[0]?.text()).toContain('Latest Snapshot')
    expect(pills[0]?.text()).toContain('$1,350')
  })

  it('summary strip second pill shows latest invested amount', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })
    const pills = wrapper.findAll('.summary-pill')
    // Second pill: "Invested" with feb2024.totalInvested = 950
    expect(pills[1]?.text()).toContain('$950')
  })

  // ── Headline change direction ────────────────────────────────────────────

  it('applies positive class to ROI panel when latest value exceeds first', () => {
    // jan2024 (1000) → feb2024 (1350): +350
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })
    expect(wrapper.find('.panel-roi-number.positive').exists()).toBe(true)
    expect(wrapper.find('.panel-roi-number.negative').exists()).toBe(false)
  })

  it('applies negative class to ROI panel when latest value is below first', () => {
    // feb2024 (1350) → dec2023 ordered as first-then-last, so pass high-first, low-last
    const downHistory: ValueSnapshot[] = [
      { ...feb2024, id: 20, recordedAt: '2024-01-01T00:00:00Z' },
      { ...jan2024, id: 21, totalValue: 800, recordedAt: '2024-02-01T00:00:00Z' },
    ]
    const wrapper = mount(StatsValueOverTime, { props: { history: downHistory } })
    expect(wrapper.find('.panel-roi-number.negative').exists()).toBe(true)
    expect(wrapper.find('.panel-roi-number.positive').exists()).toBe(false)
  })

  // ── Change percent display ───────────────────────────────────────────────

  it('displays percentage change in the ROI panel when starting value is non-zero', () => {
    // jan2024 (1000) → feb2024 (1350): +35.0%
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })
    expect(wrapper.find('.panel-roi-number').text()).toContain('%')
  })

  it('shows the side panel ROI number and summary strip with 3 pills', () => {
    const wrapper = mount(StatsValueOverTime, { props: { history: [jan2024, feb2024] } })
    expect(wrapper.find('.chart-side-panel').exists()).toBe(true)
    expect(wrapper.find('.panel-roi-number').exists()).toBe(true)
    expect(wrapper.findAll('.summary-pill')).toHaveLength(3)
    expect(wrapper.text()).toContain('Period Change')
  })
})
