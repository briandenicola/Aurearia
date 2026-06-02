import { ref, onMounted, onUnmounted, type Ref } from 'vue'

const THRESHOLD = 80
const MAX_PULL = 130
const RESISTANCE = 0.45
// Minimum downward movement before a touch is treated as a pull gesture.
// Below this slop, taps and small drifts are left untouched so their click
// events are never suppressed by preventDefault().
const ENGAGE_SLOP = 10

export function usePullToRefresh(
  containerRef: Ref<HTMLElement | null>,
  onRefresh: () => Promise<void>,
) {
  const pullDistance = ref(0)
  const refreshing = ref(false)
  let startY = 0
  let pulling = false
  let engaged = false

  function isAtTop(): boolean {
    return window.scrollY <= 0
  }

  function reset() {
    pulling = false
    engaged = false
    pullDistance.value = 0
  }

  function onTouchStart(e: TouchEvent) {
    pulling = false
    engaged = false
    if (refreshing.value || !isAtTop()) return
    const touch = e.touches[0]
    if (!touch) return
    startY = touch.clientY
    pulling = true
  }

  function onTouchMove(e: TouchEvent) {
    if (!pulling || refreshing.value) return
    if (!isAtTop()) {
      engaged = false
      pullDistance.value = 0
      return
    }

    const touch = e.touches[0]
    if (!touch) return
    const dy = touch.clientY - startY
    if (dy < 0) {
      engaged = false
      pullDistance.value = 0
      return
    }

    // Only treat as a pull (and suppress native scroll) once the finger has
    // moved past the slop threshold. This keeps taps fully clickable.
    if (!engaged && dy < ENGAGE_SLOP) return
    engaged = true

    // Prevent native scroll while pulling
    e.preventDefault()
    pullDistance.value = Math.min(dy * RESISTANCE, MAX_PULL)
  }

  async function onTouchEnd() {
    if (!pulling) return
    pulling = false
    engaged = false

    if (pullDistance.value >= THRESHOLD) {
      refreshing.value = true
      pullDistance.value = THRESHOLD * 0.6
      try {
        await onRefresh()
      } finally {
        refreshing.value = false
        pullDistance.value = 0
      }
    } else {
      pullDistance.value = 0
    }
  }

  function onTouchCancel() {
    // iOS/Android can hijack a gesture (notifications, multitouch, system
    // back-swipe) and fire touchcancel instead of touchend. Without this,
    // `pulling` would stay true and every later tap at scroll-top would get
    // preventDefault()'d, silently killing clicks across the app.
    reset()
  }

  onMounted(() => {
    const el = containerRef.value
    if (!el) return
    el.addEventListener('touchstart', onTouchStart, { passive: true })
    el.addEventListener('touchmove', onTouchMove, { passive: false })
    el.addEventListener('touchend', onTouchEnd, { passive: true })
    el.addEventListener('touchcancel', onTouchCancel, { passive: true })
  })

  onUnmounted(() => {
    const el = containerRef.value
    if (!el) return
    el.removeEventListener('touchstart', onTouchStart)
    el.removeEventListener('touchmove', onTouchMove)
    el.removeEventListener('touchend', onTouchEnd)
    el.removeEventListener('touchcancel', onTouchCancel)
  })

  return { pullDistance, refreshing }
}
