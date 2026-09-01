// Singleton composable modeled on useNotifications.ts (module-level ref, no
// Pinia store). Owns the pinned-sets sidebar state: the sidebar reads
// GET /sets, filters to pinned sets, and sorts them oldest-pinned-first.
import { computed, ref } from 'vue'
import { getSets, updateSet } from '@/api/client'
import type { CoinSetSummary } from '@/types'

export const PIN_LIMIT = 5

const pinnedSets = ref<CoinSetSummary[]>([])

function sortPinned(sets: CoinSetSummary[]): CoinSetSummary[] {
  return [...sets].sort((a, b) => {
    const aTime = a.pinnedAt ? Date.parse(a.pinnedAt) : 0
    const bTime = b.pinnedAt ? Date.parse(b.pinnedAt) : 0
    if (aTime !== bTime) return aTime - bTime
    return a.name.localeCompare(b.name)
  })
}

async function refresh() {
  try {
    const res = await getSets()
    pinnedSets.value = sortPinned(res.data.sets.filter((set) => set.pinned))
  } catch {
    // The sidebar must never break navigation — keep the last-known list.
  }
}

async function setPinned(id: number, pinned: boolean) {
  // Rethrows so callers (the set-detail pin button) can surface the
  // server's cap/ownership error as a toast.
  await updateSet(id, { pinned })
  await refresh()
}

function clear() {
  pinnedSets.value = []
}

const pinLimitReached = computed(() => pinnedSets.value.length >= PIN_LIMIT)

export function usePinnedSets() {
  return {
    pinnedSets,
    pinLimitReached,
    refresh,
    setPinned,
    clear,
  }
}
