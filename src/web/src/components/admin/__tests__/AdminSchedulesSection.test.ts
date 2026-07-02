import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminSchedulesSection from '../AdminSchedulesSection.vue'

const mocks = vi.hoisted(() => ({
  getAvailabilityRuns: vi.fn(),
  getAvailabilityRunDetail: vi.fn(),
  triggerAvailabilityCheck: vi.fn(),
  getValuationRuns: vi.fn(),
  getValuationRunDetail: vi.fn(),
  triggerValuation: vi.fn(),
  cancelValuationRun: vi.fn(),
  getAuctionEndingRuns: vi.fn(),
  triggerAuctionEndingCheck: vi.fn(),
  getAuctionAlertReminderRuns: vi.fn(),
  triggerAuctionAlertReminderCheck: vi.fn(),
  getAuctionWatchBidDigestRuns: vi.fn(),
  triggerAuctionWatchBidDigest: vi.fn(),
  triggerCollectionHealthSnapshots: vi.fn(),
  triggerCoinOfDayRun: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

vi.mock('@/composables/useSafeExternalLink', () => ({
  sanitizeExternalUrl: (url: string | null | undefined) => url ?? null,
}))

describe('AdminSchedulesSection', () => {
  beforeEach(() => {
    Object.values(mocks).forEach(mock => mock.mockReset())
    mocks.getAvailabilityRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
    mocks.getValuationRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
    mocks.getAuctionEndingRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
    mocks.getAuctionWatchBidDigestRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
    mocks.getAuctionAlertReminderRuns.mockResolvedValue({
      data: {
        runs: [{
          id: 37,
          triggerType: 'manual',
          triggerUserId: 1,
          status: 'success',
          lotsChecked: 4,
          priceAlertsTriggered: 2,
          bidRemindersSent: 1,
          durationMs: 1200,
          startedAt: '2026-07-02T12:00:00Z',
          completedAt: '2026-07-02T12:00:01Z',
          createdAt: '2026-07-02T12:00:00Z',
        }],
        total: 1,
      },
    })
    mocks.triggerAuctionAlertReminderCheck.mockResolvedValue({
      data: { runId: 38, priceAlertsTriggered: 3, bidRemindersSent: 2, status: 'success', durationMs: 1500 },
    })
  })

  it('shows auction alert and reminder run history and triggers a manual run', async () => {
    const wrapper = mount(AdminSchedulesSection, {
      props: buildProps(),
      global: {
        stubs: {
          SafeExternalLink: { template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Auction Price Alert and Reminder Run History')
    expect(wrapper.text()).toContain('2')
    expect(wrapper.text()).toContain('1')

    const runButtons = wrapper.findAll('button').filter(button => button.text() === 'Run Now')
    await runButtons[2]?.trigger('click')
    await flushPromises()

    expect(mocks.triggerAuctionAlertReminderCheck).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('update:alertReminderSettingsMsg')?.at(-1)?.[0]).toContain('3 alerts, 2 reminders')
  })

  it('binds price alert scheduler controls to backend AuctionAlerts setting keys', async () => {
    const props = buildProps()
    const wrapper = mount(AdminSchedulesSection, {
      props,
      global: {
        stubs: {
          SafeExternalLink: { template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()

    const alertReminderSection = wrapper.findAll('.avail-settings')[2]
    const enabled = alertReminderSection?.find('input[type="checkbox"]')
    const time = alertReminderSection?.find('input[type="time"]')
    const interval = alertReminderSection?.find('input[type="number"]')

    await enabled?.setValue(false)
    await time?.setValue('09:30')
    await interval?.setValue('120')

    expect(props.settings.AuctionAlertsCheckEnabled).toBe('false')
    expect(props.settings.AuctionAlertsCheckStartTime).toBe('09:30')
    expect(String(props.settings.AuctionAlertsCheckInterval)).toBe('120')
    expect('PriceAlertCheckEnabled' in props.settings).toBe(false)
    expect('PriceAlertCheckStartTime' in props.settings).toBe(false)
    expect('PriceAlertCheckInterval' in props.settings).toBe(false)
  })
})

function buildProps() {
  return {
    settings: {
      AuctionEndingCheckEnabled: 'false',
      AuctionEndingCheckStartTime: '08:00',
      AuctionEndingCheckInterval: '1440',
      AuctionAlertsCheckEnabled: 'true',
      AuctionAlertsCheckStartTime: '08:00',
      AuctionAlertsCheckInterval: '60',
      AuctionWatchBidDigestEnabled: 'false',
      AuctionWatchBidDigestStartTime: '08:00',
      AuctionWatchBidDigestInterval: '1440',
    },
    settingsSaving: false,
    availSettingsMsg: '',
    availSettingsError: false,
    auctionSettingsMsg: '',
    auctionSettingsError: false,
    alertReminderSettingsMsg: '',
    alertReminderSettingsError: false,
    watchBidDigestSettingsMsg: '',
    watchBidDigestSettingsError: false,
    healthSettingsMsg: '',
    healthSettingsError: false,
    valSettingsMsg: '',
    valSettingsError: false,
  }
}
