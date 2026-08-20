import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AdminPurchaseReminderSchedule from '../AdminPurchaseReminderSchedule.vue'
import type { AppSettings } from '@/types'

function buildSettings(overrides: Partial<AppSettings> = {}): AppSettings {
  return {
    AIProvider: 'anthropic',
    OllamaURL: '',
    OllamaModel: '',
    ObversePrompt: '',
    ReversePrompt: '',
    TextExtractionPrompt: '',
    OllamaTimeout: '',
    SearXNGURL: '',
    LogLevel: 'info',
    ReminderCheckEnabled: 'true',
    ReminderCheckStartTime: '08:00',
    ...overrides,
  }
}

function mountComponent(props: { settings: AppSettings; settingsSaving?: boolean } = { settings: buildSettings() }) {
  return mount(AdminPurchaseReminderSchedule, {
    props: {
      settingsSaving: false,
      ...props,
    },
  })
}

describe('AdminPurchaseReminderSchedule', () => {
  describe('render', () => {
    it('renders the section heading', () => {
      const wrapper = mountComponent()
      expect(wrapper.text()).toContain('Purchase Reminder Delivery')
    })

    it('renders a description paragraph', () => {
      const wrapper = mountComponent()
      expect(wrapper.text()).toContain('Sends in-app notifications')
    })

    it('renders the enable toggle checkbox', () => {
      const wrapper = mountComponent()
      const checkbox = wrapper.find('#reminder-check-enabled')
      expect(checkbox.exists()).toBe(true)
      expect(checkbox.attributes('type')).toBe('checkbox')
    })

    it('renders the start-time input', () => {
      const wrapper = mountComponent()
      const input = wrapper.find('#reminder-check-start-time')
      expect(input.exists()).toBe(true)
      expect(input.attributes('type')).toBe('time')
    })

    it('renders the save button', () => {
      const wrapper = mountComponent()
      const btn = wrapper.find('button')
      expect(btn.exists()).toBe(true)
      expect(btn.text()).toBe('Save Reminder Settings')
    })
  })

  describe('accessible labels', () => {
    it('toggle label references the checkbox id', () => {
      const wrapper = mountComponent()
      const label = wrapper.find('label[for="reminder-check-enabled"]')
      expect(label.exists()).toBe(true)
      expect(label.text()).toContain('Enable Reminder Delivery')
    })

    it('start-time label references the input id', () => {
      const wrapper = mountComponent()
      const label = wrapper.find('label[for="reminder-check-start-time"]')
      expect(label.exists()).toBe(true)
    })

    it('start-time input has aria-describedby linked to the hint', () => {
      const wrapper = mountComponent()
      const input = wrapper.find('#reminder-check-start-time')
      expect(input.attributes('aria-describedby')).toBe('reminder-check-start-time-hint')
      const hint = wrapper.find('#reminder-check-start-time-hint')
      expect(hint.exists()).toBe(true)
    })
  })

  describe('load / binding', () => {
    it('toggle reflects ReminderCheckEnabled = "true" as checked', () => {
      const wrapper = mountComponent({ settings: buildSettings({ ReminderCheckEnabled: 'true' }) })
      const checkbox = wrapper.find<HTMLInputElement>('#reminder-check-enabled')
      expect(checkbox.element.checked).toBe(true)
    })

    it('toggle reflects ReminderCheckEnabled = "false" as unchecked', () => {
      const wrapper = mountComponent({ settings: buildSettings({ ReminderCheckEnabled: 'false' }) })
      const checkbox = wrapper.find<HTMLInputElement>('#reminder-check-enabled')
      expect(checkbox.element.checked).toBe(false)
    })

    it('start-time input reflects ReminderCheckStartTime value', () => {
      const wrapper = mountComponent({ settings: buildSettings({ ReminderCheckStartTime: '09:30' }) })
      const input = wrapper.find<HTMLInputElement>('#reminder-check-start-time')
      expect(input.element.value).toBe('09:30')
    })
  })

  describe('toggle interaction', () => {
    it('checking the toggle sets ReminderCheckEnabled to "true"', async () => {
      const settings = buildSettings({ ReminderCheckEnabled: 'false' })
      const wrapper = mountComponent({ settings })
      const checkbox = wrapper.find<HTMLInputElement>('#reminder-check-enabled')
      // Simulate the @change handler directly
      checkbox.element.checked = true
      await checkbox.trigger('change')
      expect(settings.ReminderCheckEnabled).toBe('true')
    })

    it('unchecking the toggle sets ReminderCheckEnabled to "false"', async () => {
      const settings = buildSettings({ ReminderCheckEnabled: 'true' })
      const wrapper = mountComponent({ settings })
      const checkbox = wrapper.find<HTMLInputElement>('#reminder-check-enabled')
      checkbox.element.checked = false
      await checkbox.trigger('change')
      expect(settings.ReminderCheckEnabled).toBe('false')
    })
  })

  describe('time update', () => {
    it('updating the start-time input reflects in settings via v-model', async () => {
      const settings = buildSettings({ ReminderCheckStartTime: '08:00' })
      const wrapper = mountComponent({ settings })
      const input = wrapper.find<HTMLInputElement>('#reminder-check-start-time')
      await input.setValue('11:00')
      expect(settings.ReminderCheckStartTime).toBe('11:00')
    })
  })

  describe('save behavior', () => {
    it('clicking save emits the save event', async () => {
      const wrapper = mountComponent()
      await wrapper.find('button').trigger('click')
      expect(wrapper.emitted('save')).toHaveLength(1)
    })

    it('save button is disabled and shows "Saving..." when settingsSaving is true', () => {
      const wrapper = mountComponent({ settings: buildSettings(), settingsSaving: true })
      const btn = wrapper.find('button')
      expect(btn.attributes('disabled')).toBeDefined()
      expect(btn.text()).toBe('Saving...')
    })

    it('save button is enabled and shows label when not saving', () => {
      const wrapper = mountComponent({ settings: buildSettings(), settingsSaving: false })
      const btn = wrapper.find('button')
      expect(btn.attributes('disabled')).toBeUndefined()
      expect(btn.text()).toBe('Save Reminder Settings')
    })
  })
})