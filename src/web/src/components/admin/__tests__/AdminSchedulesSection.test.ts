import { DOMWrapper, flushPromises, mount } from '@vue/test-utils'
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
  getCollectionHealthSnapshotRuns: vi.fn(),
  getCollectionHealthSnapshotStatus: vi.fn(),
  triggerCoinOfDayRun: vi.fn(),
  getCoinOfDayRuns: vi.fn(),
  getCoinOfDayRunDetail: vi.fn(),
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
    mocks.getCoinOfDayRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
    mocks.getCollectionHealthSnapshotRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
    mocks.getCollectionHealthSnapshotStatus.mockResolvedValue({
      data: { name: 'collection-health', enabled: true, isRunning: false, nextRunIn: 49320000000000 },
    })
    mocks.getCoinOfDayRunDetail.mockResolvedValue({ data: { id: 1, status: 'queued', picked: 0, skipped: 0, errors: 0 } })
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

  it('shows collection health snapshot run history and triggers a manual run', async () => {
    mocks.getCollectionHealthSnapshotRuns.mockResolvedValue({
      data: {
        runs: [{
          id: 5,
          triggerType: 'scheduled',
          status: 'success',
          usersEligible: 3,
          usersSnapshotted: 3,
          usersFailed: 0,
          durationMs: 800,
          startedAt: '2026-07-16T04:30:00Z',
          completedAt: '2026-07-16T04:30:01Z',
          createdAt: '2026-07-16T04:30:00Z',
        }],
        total: 1,
      },
    })
    mocks.triggerCollectionHealthSnapshots.mockResolvedValue({
      data: { message: 'Snapshot run complete', users: 3, snapshotsCreated: 3, skipped: 0, errors: 0, durationMs: 900 },
    })

    const wrapper = mount(AdminSchedulesSection, {
      props: buildProps(),
      global: {
        stubs: {
          SafeExternalLink: { template: '<a><slot /></a>' },
        },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Collection Health Snapshot Run History')
    expect(mocks.getCollectionHealthSnapshotRuns).toHaveBeenCalled()
    expect(mocks.getCollectionHealthSnapshotStatus).toHaveBeenCalled()
    expect(wrapper.text()).toContain('Server status:')
    expect(wrapper.text()).toContain('Enabled')
    expect(wrapper.text()).toContain('13h 42m')

    // Run Now buttons in template order: Availability(0), AuctionEnding(1),
    // AlertReminder(2), WatchBidDigest(3), Valuation(4), Health(5), CoinOfDay(6).
    const runButtons = wrapper.findAll('button').filter(button => button.text() === 'Run Now')
    await runButtons[5]?.trigger('click')
    await flushPromises()

    expect(mocks.triggerCollectionHealthSnapshots).toHaveBeenCalledTimes(1)
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

    // The Auction Price Alerts/Reminders interval input is the only one with min="15";
    // use it to locate the enclosing settings block for that section.
    const intervalMarker = wrapper.find('input[type="number"][min="15"]')
    expect(intervalMarker.exists()).toBe(true)
    const sectionEl = intervalMarker.element.closest('.mb-4')
    expect(sectionEl).toBeTruthy()
    const alertReminderSection = new DOMWrapper(sectionEl as Element)
    const enabled = alertReminderSection.find('input[type="checkbox"]')
    const time = alertReminderSection.find('input[type="time"]')
    const interval = alertReminderSection.find('input[type="number"]')

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
