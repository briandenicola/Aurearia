// notifications endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  NotificationListResponse,
} from '@/types'

// Pushover notifications
export const testPushover = () =>
  api.post<{ message: string }>('/notifications/test-pushover')

// Notifications
export const getNotifications = (page = 1, limit = 20) =>
  api.get<NotificationListResponse>('/notifications', { params: { page, limit } })

export const getUnreadNotificationCount = () =>
  api.get<{ count: number }>('/notifications/unread-count')

export const markNotificationRead = (id: number) =>
  api.put(`/notifications/${id}/read`)

export const markAllNotificationsRead = () =>
  api.put('/notifications/read-all')

export const deleteNotification = (id: number) =>
  api.delete(`/notifications/${id}`)
