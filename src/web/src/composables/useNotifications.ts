import { ref } from 'vue'
import { getUnreadNotificationCount } from '@/api/client'

const unreadCount = ref(0)
let pollTimer: ReturnType<typeof setInterval> | null = null
let polling = false

async function refresh() {
  try {
    const res = await getUnreadNotificationCount()
    unreadCount.value = res.data.count
  } catch {
    // Silently ignore — poll will retry
  }
}

function startPolling() {
  if (polling) return
  polling = true
  refresh()
  pollTimer = setInterval(refresh, 60_000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  polling = false
}

export function useNotifications() {
  return {
    unreadCount,
    refresh,
    startPolling,
    stopPolling,
  }
}
