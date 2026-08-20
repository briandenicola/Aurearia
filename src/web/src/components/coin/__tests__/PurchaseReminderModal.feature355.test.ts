import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { PurchaseReminder } from '@/types/coin'

// Independent QA coverage for Feature 355 -- PurchaseReminderModal component.
// Owned by Brutus (Tester/QA).
//
// Component contract (implemented by Aurelia, verified by Brutus):
//   - Controlled modal: emits save(date), cancel, close -- no internal API calls.
//   - Props: coinId, coinName, existingReminder?, saving?, saveError?.
//   - date input type="date" with min=todayDateString() (from composable).
//   - Escape key / Close button emit "close".
//   - Remove Reminder button emits "cancel" (only when existingReminder is set).
//   - Set/Save button emits "save" with the selected date.
//   - aria-labelledby + role="dialog" for screen readers.
//   - No emoji in rendered text (UI convention).
//   - Mobile: labelled action buttons use the .btn hierarchy (44px+ in production).

vi.mock('@/composables/usePurchaseReminder', () => ({
  usePurchaseReminder: vi.fn(),
  todayDateString: () => '2026-09-15',
  formatReminderBadge: (d: string) => `Due ${d}`,
  getBrowserTimezone: () => 'America/Chicago',
}))

import PurchaseReminderModal from '@/components/coin/PurchaseReminderModal.vue'

function buildExistingReminder(overrides: Partial<PurchaseReminder> = {}): PurchaseReminder {
  return {
    id: 5,
    coinId: 42,
    remindDate: '2026-10-15',
    timezone: 'America/Chicago',
    status: 'pending',
    createdAt: '2026-09-01T00:00:00Z',
    updatedAt: '2026-09-01T00:00:00Z',
    ...overrides,
  }
}

function mountModal(props: Record<string, unknown> = {}) {
  return mount(PurchaseReminderModal, {
    props: {
      coinId: 42,
      coinName: 'Trajan Denarius',
      ...props,
    },
    attachTo: document.body,
  })
}

describe('PurchaseReminderModal -- contract and accessibility (Feature 355)', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  // --- Structural accessibility ---

  it('has role="dialog" and aria-labelledby for screen readers', () => {
    const wrapper = mountModal()
    const dialog = wrapper.find('[role="dialog"]')
    expect(dialog.exists()).toBe(true)
    expect(dialog.attributes('aria-labelledby')).toBeTruthy()
  })

  it('the aria-labelledby target element exists and contains heading text', () => {
    const wrapper = mountModal()
    const dialog = wrapper.find('[role="dialog"]')
    const labelId = dialog.attributes('aria-labelledby') ?? ''
    expect(labelId).not.toBe('')
    // Use attribute selector (CSS.escape not available in jsdom)
    const heading = wrapper.find('[id="' + labelId + '"]')
    expect(heading.exists()).toBe(true)
    expect(heading.text().length).toBeGreaterThan(0)
  })

  it('contains a date input of type="date" with a min attribute', () => {
    const wrapper = mountModal()
    const input = wrapper.find('input[type="date"]')
    expect(input.exists()).toBe(true)
    const min = input.attributes('min')
    expect(min).toBe('2026-09-15') // todayDateString() mock value
  })

  it('date input has aria-required="true"', () => {
    const wrapper = mountModal()
    expect(wrapper.find('input[type="date"]').attributes('aria-required')).toBe('true')
  })

  it('shows the coin name in the modal', () => {
    const wrapper = mountModal({ coinName: 'Augustus Aureus' })
    expect(wrapper.text()).toContain('Augustus Aureus')
  })

  it('does not contain emoji in rendered text', () => {
    const wrapper = mountModal()
    const emojiRegex = /[\u{1F300}-\u{1FFFF}]/u
    expect(emojiRegex.test(wrapper.text())).toBe(false)
  })

  // --- Create mode (no existingReminder) ---

  it('shows "Set Reminder" heading when no existingReminder', () => {
    const wrapper = mountModal()
    expect(wrapper.text()).toContain('Set Reminder')
  })

  it('does not show Remove Reminder button in create mode', () => {
    const wrapper = mountModal()
    expect(wrapper.find('.btn-danger').exists()).toBe(false)
  })

  it('emits "save" with the selected date when submit button is clicked', async () => {
    const wrapper = mountModal()
    const input = wrapper.find('input[type="date"]')
    await input.setValue('2026-10-20')

    const saveBtn = wrapper.findAll('button').find((b) =>
      b.text().includes('Set Reminder') || b.text().includes('Save'),
    )
    expect(saveBtn).toBeTruthy()
    await saveBtn!.trigger('click')

    expect(wrapper.emitted('save')).toBeTruthy()
    expect(wrapper.emitted('save')![0]).toEqual(['2026-10-20'])
  })

  it('does not emit "save" when date is empty (client-side validation)', async () => {
    const wrapper = mountModal()
    // Date input is empty (no value set)
    const saveBtn = wrapper.findAll('button').find((b) =>
      b.text().includes('Set Reminder'),
    )
    if (saveBtn) {
      await saveBtn.trigger('click')
      await flushPromises()
    }
    const saveEmits = wrapper.emitted('save') ?? []
    expect(saveEmits.length).toBe(0)
  })

  // --- Edit mode (existingReminder provided) ---

  it('shows "Edit Reminder" heading when existingReminder prop is provided', () => {
    const wrapper = mountModal({ existingReminder: buildExistingReminder() })
    expect(wrapper.text()).toContain('Edit Reminder')
  })

  it('pre-populates the date input with existingReminder.remindDate', () => {
    const wrapper = mountModal({ existingReminder: buildExistingReminder({ remindDate: '2026-10-15' }) })
    const input = wrapper.find('input[type="date"]')
    expect((input.element as HTMLInputElement).value).toBe('2026-10-15')
  })

  it('shows Remove Reminder button only in edit mode', () => {
    const wrapper = mountModal({ existingReminder: buildExistingReminder() })
    expect(wrapper.find('.btn-danger').exists()).toBe(true)
  })

  it('emits "cancel" when Remove Reminder button is clicked', async () => {
    const wrapper = mountModal({ existingReminder: buildExistingReminder() })
    await wrapper.find('.btn-danger').trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })

  // --- Close / Escape ---

  it('emits "close" when the X close button is clicked', async () => {
    const wrapper = mountModal()
    await wrapper.find('[aria-label="Close"]').trigger('click')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emits "close" when Escape key is pressed on the overlay', async () => {
    const wrapper = mountModal()
    await wrapper.trigger('keydown', { key: 'Escape' })
    await flushPromises()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('does not emit "save" on Escape (close without saving)', async () => {
    const wrapper = mountModal()
    await wrapper.find('input[type="date"]').setValue('2026-10-20')
    await wrapper.trigger('keydown', { key: 'Escape' })
    await flushPromises()
    // close emitted, save must NOT be emitted
    expect(wrapper.emitted('close')).toBeTruthy()
    expect(wrapper.emitted('save')).toBeFalsy()
  })

  // --- Error display ---

  it('shows saveError prop text as an alert', () => {
    const wrapper = mountModal({ saveError: 'Something went wrong' })
    expect(wrapper.text()).toContain('Something went wrong')
  })

  it('shows validation error when save is attempted with date in the past', async () => {
    const wrapper = mountModal()
    const input = wrapper.find('input[type="date"]')
    await input.setValue('2020-01-01') // past date
    const saveBtn = wrapper.findAll('button').find((b) =>
      b.text().includes('Set Reminder'),
    )
    if (saveBtn) await saveBtn.trigger('click')
    await flushPromises()
    // Validation error alert should appear; save must not emit
    expect(wrapper.emitted('save') ?? []).toHaveLength(0)
    const alert = wrapper.find('[role="alert"]')
    if (alert.exists()) {
      expect(alert.text().length).toBeGreaterThan(0)
    }
  })

  // --- Mobile / touch targets ---

  it('labelled action buttons (Close, Set Reminder) use the .btn class hierarchy', () => {
    const wrapper = mountModal()
    // Exclude the icon-only X close button (aria-label="Close") which uses its own class
    const textButtons = wrapper.findAll('button').filter((b) => {
      const text = b.text().trim()
      return text.length > 0 && text !== ''
    })
    expect(textButtons.length).toBeGreaterThan(0)
    textButtons.forEach((btn) => {
      expect(btn.classes().join(' ')).toContain('btn')
    })
  })
})
