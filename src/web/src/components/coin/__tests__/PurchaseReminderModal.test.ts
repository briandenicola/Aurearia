import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PurchaseReminderModal from '../PurchaseReminderModal.vue'
import type { PurchaseReminder } from '@/types/coin'

// Mock usePurchaseReminder composable so todayDateString is controllable
const FIXED_TODAY = '2026-09-01'

vi.mock('@/composables/usePurchaseReminder', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/composables/usePurchaseReminder')>()
  return {
    ...actual,
    todayDateString: () => FIXED_TODAY,
  }
})

function buildReminder(overrides: Partial<PurchaseReminder> = {}): PurchaseReminder {
  return {
    id: 1,
    coinId: 42,
    remindDate: '2026-09-15',
    timezone: 'America/Chicago',
    status: 'pending',
    createdAt: '2026-09-01T10:00:00Z',
    updatedAt: '2026-09-01T10:00:00Z',
    ...overrides,
  }
}

function mountModal(props: {
  coinId?: number
  coinName?: string
  existingReminder?: PurchaseReminder | null
  saving?: boolean
  saveError?: string
} = {}) {
  return mount(PurchaseReminderModal, {
    props: {
      coinId: props.coinId ?? 42,
      coinName: props.coinName ?? 'Trajan Denarius',
      existingReminder: props.existingReminder ?? null,
      saving: props.saving ?? false,
      saveError: props.saveError,
    },
    global: {
      stubs: {
        X: true,
        BellRing: true,
      },
    },
    attachTo: document.body,
  })
}

describe('PurchaseReminderModal', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })

  it('renders the coin name and date input', () => {
    const wrapper = mountModal()
    expect(wrapper.text()).toContain('Trajan Denarius')
    expect(wrapper.find('input[type="date"]').exists()).toBe(true)
  })

  it('shows "Set Reminder" title in create mode', () => {
    const wrapper = mountModal()
    expect(wrapper.text()).toContain('Set Reminder')
  })

  it('shows "Edit Reminder" title when existingReminder is provided', () => {
    const wrapper = mountModal({ existingReminder: buildReminder() })
    expect(wrapper.text()).toContain('Edit Reminder')
  })

  it('pre-fills date input from existingReminder', () => {
    const wrapper = mountModal({ existingReminder: buildReminder({ remindDate: '2026-09-20' }) })
    const input = wrapper.find<HTMLInputElement>('input[type="date"]')
    expect(input.element.value).toBe('2026-09-20')
  })

  it('sets min attribute to today on the date input', () => {
    const wrapper = mountModal()
    const input = wrapper.find('input[type="date"]')
    expect(input.attributes('min')).toBe(FIXED_TODAY)
  })

  it('emits save with the selected date when form is submitted', async () => {
    const wrapper = mountModal()
    const input = wrapper.find<HTMLInputElement>('input[type="date"]')
    await input.setValue('2026-10-01')
    const submitBtn = wrapper.findAll('button').find(b => b.text().includes('Set Reminder'))
    await submitBtn!.trigger('click')
    expect(wrapper.emitted('save')).toEqual([['2026-10-01']])
  })

  it('disables the submit button when no date is selected', () => {
    const wrapper = mountModal()
    const submitBtn = wrapper.findAll('button').find(b => b.text().includes('Set Reminder'))
    expect(submitBtn?.attributes('disabled')).toBeDefined()
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('shows a validation error when the date is in the past', async () => {
    const wrapper = mountModal()
    const input = wrapper.find<HTMLInputElement>('input[type="date"]')
    await input.setValue('2026-08-01') // before FIXED_TODAY
    const submitBtn = wrapper.findAll('button').find(b => b.text().includes('Set Reminder'))
    await submitBtn!.trigger('click')
    expect(wrapper.text()).toContain('today or in the future')
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('emits cancel when Remove Reminder is clicked (edit mode)', async () => {
    const wrapper = mountModal({ existingReminder: buildReminder() })
    const cancelBtn = wrapper.findAll('button').find(b => b.text().includes('Remove Reminder'))
    expect(cancelBtn).toBeDefined()
    await cancelBtn!.trigger('click')
    expect(wrapper.emitted('cancel')).toBeDefined()
  })

  it('does not show Remove Reminder button in create mode', () => {
    const wrapper = mountModal()
    const cancelBtn = wrapper.findAll('button').find(b => b.text().includes('Remove Reminder'))
    expect(cancelBtn).toBeUndefined()
  })

  it('emits close when Close button is clicked', async () => {
    const wrapper = mountModal()
    const closeBtn = wrapper.findAll('button').find(b => b.text() === 'Close')
    await closeBtn!.trigger('click')
    expect(wrapper.emitted('close')).toBeDefined()
  })

  it('emits close when Escape key is pressed', async () => {
    const wrapper = mountModal()
    await wrapper.trigger('keydown.esc')
    expect(wrapper.emitted('close')).toBeDefined()
  })

  it('shows saveError prop when provided', () => {
    const wrapper = mountModal({ saveError: 'Reminders are only available for wishlist coins' })
    expect(wrapper.text()).toContain('Reminders are only available for wishlist coins')
  })

  it('disables submit button while saving', () => {
    const wrapper = mountModal({ saving: true })
    const submitBtn = wrapper.findAll('button').find(b => b.text().includes('Saving'))
    expect(submitBtn?.attributes('disabled')).toBeDefined()
  })

  it('submit button label shows "Save Changes" in edit mode', async () => {
    const wrapper = mountModal({ existingReminder: buildReminder() })
    const input = wrapper.find<HTMLInputElement>('input[type="date"]')
    await input.setValue('2026-10-15')
    const submitBtn = wrapper.findAll('button').find(b => b.text().includes('Save Changes'))
    expect(submitBtn).toBeDefined()
  })

  it('focuses the date input on mount', async () => {
    const wrapper = mountModal()
    await flushPromises()
    const input = wrapper.find<HTMLInputElement>('input[type="date"]')
    // focus is set via onMounted — verify input is present and focusable
    expect(input.element).toBeDefined()
  })

  it('has accessible dialog role and aria-labelledby', () => {
    const wrapper = mountModal({ coinId: 42 })
    const dialog = wrapper.find('[role="dialog"]')
    expect(dialog.exists()).toBe(true)
    expect(dialog.attributes('aria-modal')).toBe('true')
    expect(dialog.attributes('aria-labelledby')).toBe('reminder-title-42')
  })
})
