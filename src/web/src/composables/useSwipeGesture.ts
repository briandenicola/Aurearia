/**
 * Shared pointer-based horizontal swipe gesture primitive.
 *
 * Owns the pointer lifecycle, touch-action management, axis lock, edge guard,
 * exclusion filtering, and commit evaluation. Consumers own navigation, animation,
 * boundary logic, and feature gating.
 *
 * Design authority: .squad/decisions/inbox/maximus-pwa-swipe-reliability-review.md §§9-13
 */

import { onMounted, onUnmounted, ref, watch, type Ref } from 'vue'

/** -1 = leftward drag (advance), 1 = rightward drag (go back) */
export type SwipeDirection = -1 | 1

export interface SwipeGestureOptions {
  /** Horizontal travel (px) required to commit. Default: 64. */
  threshold?: number
  /**
   * Travel (px) before the axis is decided. null disables axis locking.
   * For surfaces that own both axes (touchAction 'none'), set null.
   * Default: 10.
   */
  lockSlop?: number | null
  /** Viewport edge band (px) where gesture starts are ignored. Default: 0. */
  edgeGuard?: number
  /**
   * Applied to the attach element while attached; prior value restored on detach.
   * 'none'  — surface owns both axes (Gallery, Tray)
   * 'pan-y' — vertical scroll stays with the browser (detail pages)
   * null    — leave the element's touch-action unchanged
   * Default: null.
   */
  touchAction?: 'none' | 'pan-y' | null
  /** Pointer types that may start a gesture. Default: all types. */
  pointerTypes?: readonly string[]
  /** closest() selector; a start inside a match is ignored. Default: '' (none). */
  exclude?: string
  /**
   * Call setPointerCapture on the attach element at gesture start.
   * Safe for card-surface elements (Gallery, Tray). Avoid on full-page
   * containers -- Chromium retargets click to the capture element, which
   * would break child button interactions.
   * Default: false.
   */
  capture?: boolean
  /**
   * Reactive suppression gate. Checked at gesture start and at commit.
   * When false the gesture engine stays attached but ignores all input.
   */
  enabled?: Ref<boolean>

  onStart?: (event: PointerEvent) => void
  onMove?: (dx: number, dy: number) => void
  onCommit?: (direction: SwipeDirection) => void
  onCancel?: () => void
}

export interface SwipeGestureState {
  dx: Ref<number>
  dy: Ref<number>
  isDragging: Ref<boolean>
  /** True once travel has exceeded the move slop -- for tap suppression. */
  hasMoved: Ref<boolean>
}

export function useSwipeGesture(
  target: Ref<HTMLElement | null>,
  options: SwipeGestureOptions = {},
): SwipeGestureState {
  const {
    threshold = 64,
    lockSlop = 10,
    edgeGuard = 0,
    touchAction = null,
    pointerTypes,
    exclude = '',
    capture = false,
    enabled,
    onStart,
    onMove,
    onCommit,
    onCancel,
  } = options

  const dx = ref(0)
  const dy = ref(0)
  const isDragging = ref(false)
  const hasMoved = ref(false)

  let activePointerId: number | null = null
  let startX = 0
  let startY = 0
  let axisLocked = false
  let axisIsHorizontal = false
  let ignoring = false
  let attachedElement: HTMLElement | null = null
  let savedTouchAction = ''
  // The move-slop for hasMoved: matches lockSlop when set, else 5px fallback.
  const moveSlop = lockSlop !== null ? lockSlop : 5

  function resetGesture() {
    activePointerId = null
    axisLocked = false
    axisIsHorizontal = false
    ignoring = false
    dx.value = 0
    dy.value = 0
    isDragging.value = false
    hasMoved.value = false
  }

  function onPointerDown(e: PointerEvent) {
    if (pointerTypes && !pointerTypes.includes(e.pointerType)) return
    // Second finger while a gesture is active -- cancel and abort.
    if (!e.isPrimary) {
      if (activePointerId !== null) { resetGesture(); onCancel?.() }
      return
    }
    if (enabled && !enabled.value) return
    if (edgeGuard > 0 && (e.clientX < edgeGuard || e.clientX > window.innerWidth - edgeGuard)) return
    if (exclude) {
      const evTarget = e.target as Element | null
      if (evTarget?.closest(exclude)) return
    }
    if (capture) (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)

    activePointerId = e.pointerId
    startX = e.clientX
    startY = e.clientY
    axisLocked = false
    axisIsHorizontal = false
    ignoring = false
    dx.value = 0
    dy.value = 0
    isDragging.value = true
    hasMoved.value = false
    onStart?.(e)
  }

  function onPointerMove(e: PointerEvent) {
    if (e.pointerId !== activePointerId || ignoring) return
    const curDx = e.clientX - startX
    const curDy = e.clientY - startY
    dx.value = curDx
    dy.value = curDy
    if (!hasMoved.value && (Math.abs(curDx) > moveSlop || Math.abs(curDy) > moveSlop)) hasMoved.value = true

    if (lockSlop !== null && !axisLocked) {
      if (Math.abs(curDx) > lockSlop || Math.abs(curDy) > lockSlop) {
        axisLocked = true
        axisIsHorizontal = Math.abs(curDx) >= Math.abs(curDy)
        if (!axisIsHorizontal) {
          ignoring = true
          resetGesture()
          onCancel?.()
          return
        }
      }
    }
    onMove?.(curDx, curDy)
  }

  function onPointerUp(e: PointerEvent) {
    if (e.pointerId !== activePointerId) return
    const finalDx = e.clientX - startX
    const wasDragging = isDragging.value
    const wasIgnoring = ignoring
    const wasAxisLocked = axisLocked
    const wasHorizontal = axisIsHorizontal

    if (capture && attachedElement) {
      try {
        attachedElement.releasePointerCapture(e.pointerId)
      } catch {
        // releasePointerCapture throws if the pointer is already gone (e.g. browser released it
        // for a touch-action transition); that outcome is harmless — capture has already ended.
      }
    }
    resetGesture()

    if (!wasDragging || wasIgnoring) { onCancel?.(); return }
    if (lockSlop !== null && (!wasAxisLocked || !wasHorizontal)) { onCancel?.(); return }
    if (Math.abs(finalDx) < threshold) { onCancel?.(); return }
    if (enabled && !enabled.value) { onCancel?.(); return }

    onCommit?.(finalDx < 0 ? -1 : 1)
  }

  function onPointerCancelOrLost(e: PointerEvent) {
    if (e.pointerId === activePointerId) { resetGesture(); onCancel?.() }
  }

  function attach() {
    if (attachedElement) return
    const el = target.value
    if (!el) return
    attachedElement = el
    if (touchAction !== null) { savedTouchAction = el.style.touchAction ?? ''; el.style.touchAction = touchAction }
    el.addEventListener('pointerdown', onPointerDown, { passive: true })
    el.addEventListener('pointermove', onPointerMove, { passive: true })
    el.addEventListener('pointerup', onPointerUp, { passive: true })
    el.addEventListener('pointercancel', onPointerCancelOrLost, { passive: true })
    el.addEventListener('lostpointercapture', onPointerCancelOrLost, { passive: true })
  }

  function detach() {
    const el = attachedElement
    if (!el) return
    if (touchAction !== null) el.style.touchAction = savedTouchAction
    el.removeEventListener('pointerdown', onPointerDown)
    el.removeEventListener('pointermove', onPointerMove)
    el.removeEventListener('pointerup', onPointerUp)
    el.removeEventListener('pointercancel', onPointerCancelOrLost)
    el.removeEventListener('lostpointercapture', onPointerCancelOrLost)
    attachedElement = null
    resetGesture()
  }

  onMounted(() => {
    if (enabled === undefined || enabled.value) attach()
  })

  onUnmounted(() => {
    detach()
  })

  // Sync attach/detach whenever the target ref changes.
  // Handles null->el (late conditional render), el->null (v-if removal), and el-A->el-B
  // (direct swap — detaches listeners from A and restores its touch-action before attaching B).
  watch(target, (el) => {
    if (el !== attachedElement) {
      detach()
      if (el && (enabled === undefined || enabled.value)) attach()
    }
  }, { flush: 'post' })

  if (enabled !== undefined) {
    watch(enabled, (isEnabled) => { if (isEnabled) attach(); else detach() }, { flush: 'sync' })
  }

  return { dx, dy, isDragging, hasMoved }
}