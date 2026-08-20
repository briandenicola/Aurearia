// notifications types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.

export interface Notification {
  id: number
  userId: number
  type: 'wishlist_unavailable' | 'wishlist_availability_run' | 'friend_new_coin' | 'follow_request' | 'coin_of_day' | 'api_key_rotation_required' | 'set_milestone' | 'agentic_set_proposal_ready' | 'agentic_set_proposal_failed' | 'agentic_set_created' | 'agentic_set_creation_failed' | 'ai_job_completed' | 'ai_job_failed' | 'valuation_complete' | 'purchase_reminder'
  title: string
  message: string
  referenceId: number
  referenceUrl?: string
  isRead: boolean
  createdAt: string
}

export interface NotificationListResponse {
  notifications: Notification[]
  total: number
  page: number
  limit: number
}
