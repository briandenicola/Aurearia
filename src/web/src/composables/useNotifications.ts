import { ref } from 'vue'
import { getUnreadNotificationCount } from '@/api/client'

const unreadCount = ref(0)
const error = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null
let polling = false
let generation = 0
let request = 0

async function refresh() {
  if (!polling) return
  const currentGeneration = generation
  const currentRequest = ++request
  try {
    const res = await getUnreadNotificationCount()
    if (currentGeneration !== generation || currentRequest !== request) return
    unreadCount.value = res.data.count
    error.value = ''
  } catch {
    if (currentGeneration === generation && currentRequest === request) {
      error.value = 'Unable to refresh notification count. Retry or wait for the next automatic update.'
    }
  }
}

function startPolling() {
  if (polling) return
  polling = true
  void refresh()
  pollTimer = setInterval(refresh, 60_000)
}

function stopPolling() {
  generation += 1
  request += 1
  unreadCount.value = 0
  error.value = ''
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  polling = false
}

export function useNotifications() {
  return {
    unreadCount,
    error,
    refresh,
    startPolling,
    stopPolling,
  }
}
