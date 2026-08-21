// Experimental PWA-only swipe navigation across coin-detail menu pages.
// Only active when running as an installed PWA (isPwa) and the user has
// enabled the account-wide pwaSwipeNavEnabled preference. Desktop and
// in-browser mobile are byte-for-byte unaffected regardless of preference.
//
// Exported constants are named so that Brutus's test file can reference them
// without duplicating the values.

import { onMounted, onUnmounted, watch, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { COIN_DETAIL_SECTIONS, SECTION_ORDER } from '@/constants/coinDetailSections'
import { usePwa } from '@/composables/usePwa'
import { useAuthStore } from '@/stores/auth'

/** Minimum horizontal travel (px) required to commit a swipe. */
export const SWIPE_THRESHOLD = 64
/** Distance (px) a finger must travel before the axis is decided. */
export const AXIS_SLOP = 10
/** Viewport edge width (px) inside which gesture starts are ignored. */
export const EDGE_GUARD = 24
/**
 * Required ratio: abs(dx) must be >= AXIS_DOMINANCE * abs(dy) at release.
 * A value of 2 means the horizontal travel must be at least 2x the vertical.
 */
export const AXIS_DOMINANCE = 2

// Elements/regions that legitimately own horizontal gestures or that should
// never trigger navigation (interactive controls, text editors, explicit opt-out).
const SUPPRESS_SELECTOR =
  'input, textarea, select, button, a, [role="button"], [contenteditable="true"], [data-swipe-ignore]'

export function useCoinDetailSwipeNav(
  containerRef: Ref<HTMLElement | null>,
  options?: { enabled?: Ref<boolean> },
): void {
  const { isPwa } = usePwa()
  // Exit immediately for non-PWA; no listeners, no overhead.
  if (!isPwa) return

  const auth = useAuthStore()
  const route = useRoute()
  const router = useRouter()

  // Build the ordered list of paths for the current coin at gesture time so
  // that it always reflects the coin ID in the URL.
  function buildRoutes(coinId: number): string[] {
    return [
      `/coin/${coinId}`,
      ...SECTION_ORDER.map((id) => COIN_DETAIL_SECTIONS[id]!.route(coinId)),
    ]
  }

  // Gesture state — plain mutable variables; no reactivity needed here.
  let activePointerId: number | null = null
  let startX = 0
  let startY = 0
  let axisLocked = false
  let axisIsHorizontal = false
  let ignoring = false // true when a vertical axis was locked or region was suppressed

  function reset() {
    activePointerId = null
    axisLocked = false
    axisIsHorizontal = false
    ignoring = false
  }

  function onPointerDown(e: PointerEvent) {
    if (e.pointerType !== 'touch') return
    // A second finger while a gesture is in flight cancels it (pinch/two-finger scroll).
    if (!e.isPrimary) {
      if (activePointerId !== null) reset()
      return
    }
    // Edge guard: OS/browser back-swipe zones on either side.
    if (e.clientX < EDGE_GUARD || e.clientX > window.innerWidth - EDGE_GUARD) return
    // Suppression zone: interactive controls and explicit opt-out elements.
    const target = e.target as Element | null
    if (target?.closest(SUPPRESS_SELECTOR)) return

    activePointerId = e.pointerId
    startX = e.clientX
    startY = e.clientY
    axisLocked = false
    axisIsHorizontal = false
    ignoring = false
  }

  function onPointerMove(e: PointerEvent) {
    if (e.pointerId !== activePointerId || ignoring) return
    if (axisLocked) return
    const absDx = Math.abs(e.clientX - startX)
    const absDy = Math.abs(e.clientY - startY)
    if (absDx > AXIS_SLOP || absDy > AXIS_SLOP) {
      axisLocked = true
      axisIsHorizontal = absDx >= absDy
      // If vertical won the axis race, abandon for this entire pointer stream.
      if (!axisIsHorizontal) ignoring = true
    }
  }

  function onPointerUp(e: PointerEvent) {
    if (e.pointerId !== activePointerId) return
    const dx = e.clientX - startX
    const dy = e.clientY - startY
    const wasIgnoring = ignoring
    const wasAxisLocked = axisLocked
    const wasHorizontal = axisIsHorizontal
    reset()

    if (wasIgnoring || !wasAxisLocked || !wasHorizontal) return
    if (Math.abs(dx) < SWIPE_THRESHOLD) return
    if (Math.abs(dx) < AXIS_DOMINANCE * Math.abs(dy)) return
    // Discard if the user has selected text (e.g., dragged over a paragraph).
    if (window.getSelection()?.toString()) return
    // Explicit enabled gate (e.g., a modal is open inside this container).
    if (options?.enabled && !options.enabled.value) return

    const coinId = Number(route.params.id)
    const routes = buildRoutes(coinId)
    const currentIndex = routes.findIndex((r) => r === route.path)
    if (currentIndex === -1) return

    const nextIndex = dx < 0 ? currentIndex + 1 : currentIndex - 1
    if (nextIndex < 0 || nextIndex >= routes.length) return // no wrap at boundaries

    router.push({ path: routes[nextIndex]!, query: route.query, hash: route.hash })
  }

  function onPointerCancel(e: PointerEvent) {
    if (e.pointerId === activePointerId) reset()
  }

  // Capture the element at attach time so the same node is used for removal
  // even if the ref is cleared before detach is called.
  let attachedElement: HTMLElement | null = null

  function attach() {
    // No-op when already attached: prevents duplicate listeners on repeated enable/disable cycles.
    if (attachedElement) return
    const el = containerRef.value
    if (!el) return
    attachedElement = el
    el.addEventListener('pointerdown', onPointerDown, { passive: true })
    el.addEventListener('pointermove', onPointerMove, { passive: true })
    el.addEventListener('pointerup', onPointerUp, { passive: true })
    el.addEventListener('pointercancel', onPointerCancel, { passive: true })
    el.addEventListener('lostpointercapture', onPointerCancel, { passive: true })
  }

  function detach() {
    const el = attachedElement
    if (!el) return
    el.removeEventListener('pointerdown', onPointerDown)
    el.removeEventListener('pointermove', onPointerMove)
    el.removeEventListener('pointerup', onPointerUp)
    el.removeEventListener('pointercancel', onPointerCancel)
    el.removeEventListener('lostpointercapture', onPointerCancel)
    attachedElement = null
  }

  onMounted(() => {
    // Attach only when the account preference is enabled at mount time.
    // Subsequent preference changes are handled by the watcher below.
    if (auth.user?.pwaSwipeNavEnabled === true) attach()
  })

  onUnmounted(() => {
    detach()
  })

  // Reactively attach or detach when the user enables/disables the preference,
  // logs out (auth.user becomes null), or switches accounts (new user value).
  // attach() no-ops if already attached; detach() no-ops if not attached.
  watch(
    () => auth.user?.pwaSwipeNavEnabled === true,
    (enabled) => {
      if (enabled) attach()
      else detach()
    },
  )
}
