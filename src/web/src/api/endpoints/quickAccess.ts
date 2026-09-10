import { api } from '@/api/http'
import type { QuickAccessItem, QuickAccessListDTO, QuickAccessTargetType } from '@/types'

export const getQuickAccess = () => api.get<QuickAccessListDTO>('/quick-access')

export const pinQuickAccess = (type: QuickAccessTargetType, id: number) =>
  api.put<QuickAccessItem>(`/quick-access/${type}/${id}`)

export const unpinQuickAccess = (type: QuickAccessTargetType, id: number) =>
  api.delete<void>(`/quick-access/${type}/${id}`)
