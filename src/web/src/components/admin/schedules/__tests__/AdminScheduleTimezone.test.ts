import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { reactive } from 'vue'
import AdminScheduleTimezone from '../AdminScheduleTimezone.vue'
import AdminScheduleSummary from '../AdminScheduleSummary.vue'
import type { AppSettings } from '@/types'

vi.mock('@/composables/usePurchaseReminder', () => ({ getBrowserTimezone: () => 'America/Chicago' }))

function mountTimezone(overrides: Record<string, string> = {}, settingsSaving = false) {
  const settings = reactive({ ...overrides }) as unknown as AppSettings
  const wrapper = mount(AdminScheduleTimezone, { props: { settings, settingsSaving } })
  return { wrapper, settings }
}

describe('AdminScheduleTimezone', () => {
  it('defaults to server time and stores a chosen zone', async () => {
    const { wrapper, settings } = mountTimezone()

    const select = wrapper.get<HTMLSelectElement>('#schedule-timezone')
    expect(select.element.value).toBe('')
    expect(wrapper.text()).toContain('apply within a minute')

    await select.setValue('Europe/London')
    expect(settings.ScheduleTimezone).toBe('Europe/London')
  })

  it('offers the browser zone in one click', async () => {
    const { wrapper, settings } = mountTimezone()

    await wrapper.get('button.text-gold').trigger('click')
    expect(settings.ScheduleTimezone).toBe('America/Chicago')
    expect(wrapper.find('button.text-gold').exists()).toBe(false)
  })

  it('keeps an unlisted saved zone selectable', () => {
    const { wrapper } = mountTimezone({ ScheduleTimezone: 'Etc/GMT+6' })
    expect(wrapper.get<HTMLSelectElement>('#schedule-timezone').element.value).toBe('Etc/GMT+6')
  })

  it('emits save and disables the button while saving', async () => {
    const { wrapper } = mountTimezone()
    await wrapper.get('button.btn-primary').trigger('click')
    expect(wrapper.emitted('save')).toHaveLength(1)

    const saving = mountTimezone({}, true).wrapper
    expect(saving.get('button.btn-primary').attributes('disabled')).toBeDefined()
    expect(saving.get('button.btn-primary').text()).toBe('Saving...')
  })
})

describe('AdminScheduleSummary', () => {
  it('describes the start time, interval and zone', () => {
    const wrapper = mount(AdminScheduleSummary, {
      props: { startTime: '06:00', defaultStartTime: '02:00', intervalMinutes: '120', zone: 'America/Chicago' },
    })
    expect(wrapper.text()).toBe("Runs at 06:00 America/Chicago, then every 2 hours until the next day's start.")
  })

  it('falls back to the default start time and server time', () => {
    const wrapper = mount(AdminScheduleSummary, { props: { defaultStartTime: '04:30' } })
    expect(wrapper.text()).toBe('Runs daily at 04:30 server time.')
  })
})
