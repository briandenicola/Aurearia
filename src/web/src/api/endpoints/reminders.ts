// reminders endpoints — purchase reminder CRUD (Feature 355).
// Re-exported from '@/api/client' so existing imports keep working.
// Note: "listReminders" / "deleteReminder" / "createReminder" are already exported
// from auctions.ts (bid reminders). These are prefixed to avoid collision.
import { api } from '@/api/http'
import type { PurchaseReminder, PurchaseReminderListResponse } from '@/types/coin'

export const createOrUpdatePurchaseReminder = (
  coinId: number,
  data: { remindDate: string; timezone: string },
) => api.post<PurchaseReminder>(`/coins/${coinId}/reminder`, data)

export const getPurchaseReminder = (coinId: number) =>
  api.get<PurchaseReminder>(`/coins/${coinId}/reminder`)

export const deletePurchaseReminder = (coinId: number) =>
  api.delete<void>(`/coins/${coinId}/reminder`)

export const listPurchaseReminders = () =>
  api.get<PurchaseReminderListResponse>('/purchase-reminders')
