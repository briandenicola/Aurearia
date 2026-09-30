import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import AdminCoinOfDaySchedule from '../AdminCoinOfDaySchedule.vue'
import type { AppSettings } from '@/types'

const mocks = vi.hoisted(() => ({
  getCoinOfDayRuns: vi.fn(),
  getCoinOfDayRunDetail: vi.fn(),
  triggerCoinOfDayRun: vi.fn(),
  getBrowserTimezone: vi.fn(() => 'America/Chicago'),
}))

vi.mock('@/api/client', () => ({
  getCoinOfDayRuns: mocks.getCoinOfDayRuns,
  getCoinOfDayRunDetail: mocks.getCoinOfDayRunDetail,
  triggerCoinOfDayRun: mocks.triggerCoinOfDayRun,
}))
vi.mock('@/composables/usePurchaseReminder', () => ({ getBrowserTimezone: mocks.getBrowserTimezone }))

function mountSchedule(overrides: Record<string, string> = {}) {
  const settings = reactive({
    CoinOfDayEnabled: 'true',
    CoinOfDayStartTime: '12:00',
    ...overrides,
  }) as unknown as AppSettings
  const wrapper = mount(AdminCoinOfDaySchedule, { props: { settings, settingsSaving: false } })
  return { wrapper, settings }
}

describe('AdminCoinOfDaySchedule time zone', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getCoinOfDayRuns.mockResolvedValue({ data: { runs: [], total: 0 } })
  })

  it('defaults to server time and explains when the job runs', async () => {
    const { wrapper } = mountSchedule()
    await flushPromises()

    const select = wrapper.get<HTMLSelectElement>('#coin-of-day-timezone')
    expect(select.element.value).toBe('')
    expect(wrapper.text()).toContain('Runs daily at 12:00 server time.')
    expect(wrapper.text()).toContain('no restart needed')
  })

  it('stores the chosen zone and offers a one-click browser zone', async () => {
    const { wrapper, settings } = mountSchedule()
    await flushPromises()

    await wrapper.get('button.text-gold').trigger('click')
    expect(settings.CoinOfDayTimezone).toBe('America/Chicago')
    expect(wrapper.text()).toContain('Runs daily at 12:00 America/Chicago.')
    expect(wrapper.find('button.text-gold').exists()).toBe(false)

    await wrapper.get('#coin-of-day-timezone').setValue('Europe/London')
    expect(settings.CoinOfDayTimezone).toBe('Europe/London')
  })

  it('keeps a saved zone selectable even if the browser does not list it', async () => {
    const { wrapper } = mountSchedule({ CoinOfDayTimezone: 'Etc/GMT+6' })
    await flushPromises()

    expect(wrapper.get<HTMLSelectElement>('#coin-of-day-timezone').element.value).toBe('Etc/GMT+6')
  })

  it('falls back to the shared schedule time zone when no override is set', async () => {
    const { wrapper } = mountSchedule({ ScheduleTimezone: 'America/New_York' })
    await flushPromises()

    expect(wrapper.get('#coin-of-day-timezone option[value=""]').text()).toBe('Same as schedules (America/New_York)')
    expect(wrapper.text()).toContain('Runs daily at 12:00 America/New_York.')
  })
})
