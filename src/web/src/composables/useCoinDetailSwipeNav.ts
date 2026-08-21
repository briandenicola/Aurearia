// Thin adapter: stop list, index arithmetic, PWA gate, pwaSwipeNavEnabled gate.
// The pointer state machine is owned by useSwipeGesture.
//
// Exported constants are kept so existing test imports remain valid.
// AXIS_DOMINANCE is removed -- the axis lock is the single direction decision.

import { computed, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { COIN_DETAIL_SECTIONS, SECTION_ORDER } from '@/constants/coinDetailSections'
import { usePwa } from '@/composables/usePwa'
import { useAuthStore } from '@/stores/auth'
import { useSwipeGesture } from '@/composables/useSwipeGesture'

/** Minimum horizontal travel (px) required to commit a swipe. */
export const SWIPE_THRESHOLD = 64
/** Distance (px) a finger must travel before the axis is decided. */
export const AXIS_SLOP = 10
/** Viewport edge width (px) inside which gesture starts are ignored. */
export const EDGE_GUARD = 24

// Narrowed exclusion: only elements that genuinely own horizontal input or
// are text-entry fields. Ordinary buttons and anchors do not prevent a
// deliberate 64 px swipe and must not suppress navigation.
const SUPPRESS_SELECTOR =
  'input, textarea, select, [contenteditable="true"], [data-swipe-ignore]'

export function useCoinDetailSwipeNav(
  containerRef: Ref<HTMLElement | null>,
  options?: { enabled?: Ref<boolean> },
): void {
  const { isPwa } = usePwa()
  // Exit immediately for non-PWA; no lifecycle hooks registered, no overhead.
  if (!isPwa) return

  const auth = useAuthStore()
  const route = useRoute()
  const router = useRouter()

  function buildRoutes(coinId: number): string[] {
    return [
      `/coin/${coinId}`,
      ...SECTION_ORDER.map((id) => COIN_DETAIL_SECTIONS[id]!.route(coinId)),
    ]
  }

  // Combined gate: account preference AND the external modal-suppression gate.
  const swipeEnabled = computed(() => {
    if (auth.user?.pwaSwipeNavEnabled !== true) return false
    if (options?.enabled !== undefined && !options.enabled.value) return false
    return true
  })

  useSwipeGesture(containerRef, {
    threshold: SWIPE_THRESHOLD,
    lockSlop: AXIS_SLOP,
    edgeGuard: EDGE_GUARD,
    touchAction: 'pan-y',
    pointerTypes: ['touch'],
    exclude: SUPPRESS_SELECTOR,
    capture: false,
    enabled: swipeEnabled,
    onCommit(direction) {
      const coinId = Number(route.params.id)
      const routes = buildRoutes(coinId)
      const currentIndex = routes.findIndex((r) => r === route.path)
      if (currentIndex === -1) return

      // direction: -1 = leftward drag (advance), 1 = rightward drag (go back)
      const nextIndex = direction === -1 ? currentIndex + 1 : currentIndex - 1
      if (nextIndex < 0 || nextIndex >= routes.length) return // no wrap at boundaries

      router.push({ path: routes[nextIndex]!, query: route.query, hash: route.hash })
    },
  })
}