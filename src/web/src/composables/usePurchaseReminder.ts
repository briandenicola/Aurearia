import { ref, watch } from 'vue'
import type { Ref } from 'vue'
import { createOrUpdatePurchaseReminder, getPurchaseReminder, deletePurchaseReminder } from '@/api/client'
import type { PurchaseReminder } from '@/types/coin'

/** Returns the browser's IANA timezone (e.g. "America/Chicago"). */
export function getBrowserTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone
}

/**
 * Formats a YYYY-MM-DD remind date into a compact badge label.
 * Returns "Due Today", "Due Tomorrow", or "Due {Mon DD}".
 */
export function formatReminderBadge(remindDate: string): string {
  const now = new Date()
  const todayStr = now.toISOString().slice(0, 10)

  const tomorrow = new Date(now)
  tomorrow.setDate(tomorrow.getDate() + 1)
  const tomorrowStr = tomorrow.toISOString().slice(0, 10)

  if (remindDate <= todayStr) return 'Due Today'
  if (remindDate === tomorrowStr) return 'Due Tomorrow'

  // Format as "Due Aug 25" using parts split from "YYYY-MM-DD"
  const parts = remindDate.split('-')
  const month = parseInt(parts[1] ?? '1', 10)
  const day = parseInt(parts[2] ?? '1', 10)
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  return `Due ${months[month - 1] ?? '?'} ${day}`
}

/**
 * Formats a YYYY-MM-DD remind date as a plain localized date string for detail-row
 * display (e.g. "9/1/2026"). Uses local date construction to avoid UTC midnight
 * timezone shift.
 */
export function formatReminderDateValue(remindDate: string): string {
  const parts = remindDate.split('-')
  const y = parseInt(parts[0] ?? '2000', 10)
  const m = parseInt(parts[1] ?? '1', 10)
  const d = parseInt(parts[2] ?? '1', 10)
  return new Date(y, m - 1, d).toLocaleDateString()
}

/** Returns today's date as YYYY-MM-DD in local time. */
export function todayDateString(): string {
  const now = new Date()
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, '0')
  const d = String(now.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

export interface UsePurchaseReminderReturn {
  reminder: Ref<PurchaseReminder | null>
  loading: Ref<boolean>
  saving: Ref<boolean>
  error: Ref<string>
  fetchReminder: () => Promise<void>
  saveReminder: (date: string) => Promise<PurchaseReminder | null>
  cancelReminder: () => Promise<void>
}

/**
 * Per-coin reminder state composable.
 * Fetches, creates/updates, and cancels the active reminder for a given coin.
 */
export function usePurchaseReminder(coinId: Ref<number>): UsePurchaseReminderReturn {
  const reminder = ref<PurchaseReminder | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')

  async function fetchReminder(): Promise<void> {
    if (!coinId.value) return
    loading.value = true
    error.value = ''
    try {
      const res = await getPurchaseReminder(coinId.value)
      reminder.value = res.data
    } catch (err: unknown) {
      // 404 = no active reminder; anything else surfaces the error
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 404) {
        reminder.value = null
      } else {
        error.value = 'Failed to load reminder'
      }
    } finally {
      loading.value = false
    }
  }

  async function saveReminder(date: string): Promise<PurchaseReminder | null> {
    saving.value = true
    error.value = ''
    try {
      const res = await createOrUpdatePurchaseReminder(coinId.value, {
        remindDate: date,
        timezone: getBrowserTimezone(),
      })
      reminder.value = res.data
      return res.data
    } catch (err: unknown) {
      const data = (err as { response?: { data?: { error?: string } } })?.response?.data
      error.value = data?.error ?? 'Failed to save reminder'
      return null
    } finally {
      saving.value = false
    }
  }

  async function cancelReminder(): Promise<void> {
    saving.value = true
    error.value = ''
    try {
      await deletePurchaseReminder(coinId.value)
      reminder.value = null
    } catch {
      error.value = 'Failed to cancel reminder'
    } finally {
      saving.value = false
    }
  }

  // Refetch when coinId changes
  watch(coinId, fetchReminder, { immediate: false })

  return { reminder, loading, saving, error, fetchReminder, saveReminder, cancelReminder }
}
