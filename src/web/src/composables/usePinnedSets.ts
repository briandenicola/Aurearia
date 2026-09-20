import { computed } from 'vue'
import { useQuickAccess } from './useQuickAccess'

export const PIN_LIMIT = 5

const quickAccess = useQuickAccess()
const pinnedSets = computed(() => quickAccess.items.value
  .filter(item => item.type === 'coin_set')
  .map(item => ({ id: item.id, pinnedAt: item.pinnedAt, ...item.coinSet }))
  .sort((a, b) => Date.parse(a.pinnedAt) - Date.parse(b.pinnedAt) || a.name.localeCompare(b.name)))
const pinLimitReached = computed(() => pinnedSets.value.length >= PIN_LIMIT)

// Compatibility surface; Quick Access owns all state and request invalidation.
export function usePinnedSets() {
  return {
    pinnedSets,
    pinLimitReached,
    error: quickAccess.error,
    refresh: quickAccess.refresh,
    setPinned: (id: number, pinned: boolean) => pinned
      ? quickAccess.pin('coin_set', id) : quickAccess.unpin('coin_set', id),
    clear: quickAccess.clear,
  }
}
