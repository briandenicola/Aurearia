import { computed, ref } from 'vue'
import { getApiErrorMessage, getQuickAccess, pinQuickAccess, unpinQuickAccess } from '@/api/client'
import type { QuickAccessItem, QuickAccessTargetType } from '@/types'

const items = ref<QuickAccessItem[]>([])
const loading = ref(false)
const error = ref('')
const busyKeys = ref(new Set<string>())
let generation = 0
let stateRevision = 0
let refreshPromise: Promise<void> | null = null
let refreshRequestId = 0

function key(type: QuickAccessTargetType, id: number) {
  return `${type}:${id}`
}

function setBusy(type: QuickAccessTargetType, id: number, busy: boolean) {
  const next = new Set(busyKeys.value)
  if (busy) next.add(key(type, id))
  else next.delete(key(type, id))
  busyKeys.value = next
}

function refresh() {
  const requestGeneration = generation
  const requestRevision = stateRevision
  if (refreshPromise) return refreshPromise

  loading.value = true
  error.value = ''
  const requestId = ++refreshRequestId
  const request = (async () => {
    try {
      const response = await getQuickAccess()
      if (requestGeneration === generation && requestRevision === stateRevision) {
        items.value = response.data.items ?? []
      }
    } catch (cause) {
      if (requestGeneration === generation && requestRevision === stateRevision) {
        error.value = getApiErrorMessage(cause) || 'Unable to load Quick Access.'
      }
    } finally {
      if (requestGeneration === generation) loading.value = false
      if (requestId === refreshRequestId) {
        refreshPromise = null
      }
    }
  })()
  refreshPromise = request
  return request
}

function invalidateRefresh() {
  stateRevision += 1
  refreshRequestId += 1
  refreshPromise = null
}

async function pin(type: QuickAccessTargetType, id: number) {
  const requestGeneration = generation
  setBusy(type, id, true)
  try {
    const response = await pinQuickAccess(type, id)
    if (requestGeneration !== generation) return response.data

    invalidateRefresh()
    const index = items.value.findIndex((item) => item.type === type && item.id === id)
    if (response.status === 201 || index < 0) {
      items.value = [response.data, ...items.value.filter((item) => item.type !== type || item.id !== id)]
    } else {
      items.value = items.value.map((item, itemIndex) => itemIndex === index ? response.data : item)
    }
    error.value = ''
    return response.data
  } catch (cause) {
    if (requestGeneration === generation) {
      error.value = getApiErrorMessage(cause) || 'Unable to pin this item.'
    }
    throw cause
  } finally {
    if (requestGeneration === generation) setBusy(type, id, false)
  }
}

async function unpin(type: QuickAccessTargetType, id: number) {
  const requestGeneration = generation
  setBusy(type, id, true)
  try {
    await unpinQuickAccess(type, id)
    if (requestGeneration === generation) {
      invalidateRefresh()
      items.value = items.value.filter((item) => item.type !== type || item.id !== id)
      error.value = ''
    }
  } catch (cause) {
    if (requestGeneration === generation) {
      error.value = getApiErrorMessage(cause) || 'Unable to unpin this item.'
    }
    throw cause
  } finally {
    if (requestGeneration === generation) setBusy(type, id, false)
  }
}

function forget(type: QuickAccessTargetType, id: number) {
  invalidateRefresh()
  items.value = items.value.filter((item) => item.type !== type || item.id !== id)
}

function clear() {
  generation += 1
  invalidateRefresh()
  items.value = []
  loading.value = false
  error.value = ''
  busyKeys.value = new Set()
}

function isPinned(type: QuickAccessTargetType, id: number) {
  return computed(() => items.value.some((item) => item.type === type && item.id === id))
}

function isBusy(type: QuickAccessTargetType, id: number) {
  return computed(() => busyKeys.value.has(key(type, id)))
}

export function useQuickAccess() {
  return {
    items,
    loading,
    error,
    refresh,
    pin,
    unpin,
    forget,
    isPinned,
    isBusy,
    clear,
  }
}
