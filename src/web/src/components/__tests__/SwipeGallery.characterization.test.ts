/**
 * Behaviour-level characterization tests for SwipeGallery's inline swipe engine.
 *
 * Design authority: .squad/decisions/inbox/maximus-pwa-swipe-reliability-review.md §§9–13
 * Purpose: pin the current intended contract of the Gallery drag engine BEFORE production
 * migration to the shared useSwipeGesture primitive (Phase 2 / PR 3 of the consolidation plan).
 * PR 3 must pass these tests unmodified after migration.
 *
 * DIRECTION (Phase 2 — aligned with iOS convention; user directive 2026-08-21T13:20)
 * All three surfaces now share the same direction semantics via useSwipeGesture:
 *   left drag (negative dx)  → direction -1 → flyAway(-1) → next coin
 *   right drag (positive dx) → direction  1 → flyAway( 1) → previous coin
 * G1/G2 direction expectations updated accordingly.
 *
 * THRESHOLD NOTE
 * Strict-greater-than preserved: primitive threshold: 101 → commits at >= 101, i.e., > 100.
 * 100 px springs back; 101 px is the minimum commit travel.
 *
 * G4 REGRESSION TEST (was it.fails — defect fixed by shared primitive):
 *   useSwipeGesture routes pointercancel/lostpointercapture → onCancel → spring-back.
 *   The prior Gallery inline engine wired cancel to onPointerUp which committed at threshold. */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import SwipeGallery from '../SwipeGallery.vue'
import { useCoinsStore } from '@/stores/coins'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'
import type { Coin } from '@/types'

// ---------------------------------------------------------------------------
// Hoisted mocks
// ---------------------------------------------------------------------------

const mockPush = vi.hoisted(() => vi.fn())

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

// ---------------------------------------------------------------------------
// Constants mirrored from SwipeGallery.vue (not exported)
// ---------------------------------------------------------------------------

/** Must match the component's SWIPE_THRESHOLD constant. */
const SWIPE_THRESHOLD = 100
/** Minimum px required to commit: Gallery uses strict >, so threshold+1. */
const MIN_COMMIT_DIST = SWIPE_THRESHOLD + 1
/** dragX must exceed this before hasDragged is set to true. */
const HASDRAGGED_SLOP = 5

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

function buildTestCoins(count: number, images: Coin['images'] = []): Coin[] {
  return Array.from({ length: count }, (_, i) =>
    buildRomanDenariusCore({ id: i + 1, name: `Coin ${i + 1}`, images }),
  )
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

function mountGallery(opts: {
  coins?: Coin[]
  total?: number
  page?: number
  perPage?: number
  startIndex?: number
} = {}) {
  const coins = opts.coins ?? buildTestCoins(5)
  const total = opts.total ?? coins.length
  const page = opts.page ?? 1
  const perPage = opts.perPage ?? 50
  const startIndex = opts.startIndex ?? 0

  const store = useCoinsStore()
  store.galleryIndex = startIndex

  return mount(SwipeGallery, {
    props: { coins, total, page, perPage },
  })
}

// ---------------------------------------------------------------------------
// Pointer event synthesis (same pattern as useCoinDetailSwipeNav.test.ts)
// ---------------------------------------------------------------------------

function firePointer(
  element: Element,
  type: string,
  opts: {
    clientX?: number
    clientY?: number
    pointerId?: number
    pointerType?: string
    isPrimary?: boolean
  } = {},
) {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperties(event, {
    pointerId: { value: opts.pointerId ?? 1 },
    pointerType: { value: opts.pointerType ?? 'touch' },
    isPrimary: { value: opts.isPrimary ?? true },
    clientX: { value: opts.clientX ?? 200 },
    clientY: { value: opts.clientY ?? 200 },
  })
  element.dispatchEvent(event)
}

/**
 * Return .active-card for event dispatch.
 * After Phase 2 migration useSwipeGesture attaches to .card-stack, so we stub
 * capture methods there (jsdom does not implement setPointerCapture natively).
 */
function prepCard(wrapper: ReturnType<typeof mountGallery>): HTMLElement {
  const stack = wrapper.find('.card-stack').element as HTMLElement
  stack.setPointerCapture = vi.fn()
  stack.releasePointerCapture = vi.fn()
  return wrapper.find('.active-card').element as HTMLElement
}

/** Return .card-stack element to assert capture calls (primitive target after Phase 2). */
function getStack(wrapper: ReturnType<typeof mountGallery>): HTMLElement {
  return wrapper.find('.card-stack').element as HTMLElement
}

/**
 * Right drag: positive dx → direction=1 → flyAway(1) → previous coin (Phase 2 / iOS convention).
 */
function doRightDrag(card: HTMLElement, distance = MIN_COMMIT_DIST + 19, startX = 300, startY = 300) {
  firePointer(card, 'pointerdown', { clientX: startX, clientY: startY })
  firePointer(card, 'pointermove', { clientX: startX + Math.floor(distance / 2), clientY: startY + 1 })
  firePointer(card, 'pointermove', { clientX: startX + distance, clientY: startY + 1 })
  firePointer(card, 'pointerup', { clientX: startX + distance, clientY: startY + 1 })
}

/**
 * Left drag: negative dx → direction=-1 → flyAway(-1) → next coin (Phase 2 / iOS convention).
 */
function doLeftDrag(card: HTMLElement, distance = MIN_COMMIT_DIST + 19, startX = 300, startY = 300) {
  firePointer(card, 'pointerdown', { clientX: startX, clientY: startY })
  firePointer(card, 'pointermove', { clientX: startX - Math.floor(distance / 2), clientY: startY + 1 })
  firePointer(card, 'pointermove', { clientX: startX - distance, clientY: startY + 1 })
  firePointer(card, 'pointerup', { clientX: startX - distance, clientY: startY + 1 })
}

// ---------------------------------------------------------------------------
// Suite setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  setActivePinia(createPinia())
  vi.useFakeTimers()
  mockPush.mockClear()
})

afterEach(() => {
  vi.useRealTimers()
})

// ===========================================================================
// Pointer type policy — Gallery accepts ALL pointer types (touch, mouse, pen)
// ===========================================================================

describe('pointer type policy — all pointer types accepted', () => {
  for (const pointerType of ['touch', 'mouse', 'pen'] as const) {
    it(`${pointerType} pointer starts a drag and commits at threshold+1`, async () => {
      const wrapper = mountGallery({ startIndex: 0 })
      const store = useCoinsStore()
      const card = prepCard(wrapper)

      firePointer(card, 'pointerdown', { clientX: 300, clientY: 300, pointerType })
      firePointer(card, 'pointermove', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301, pointerType })
      firePointer(card, 'pointerup', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301, pointerType })
      await vi.advanceTimersByTimeAsync(300)

      expect(store.galleryIndex).toBe(1)
      wrapper.unmount()
    })
  }
})

// ===========================================================================
// Commit threshold: 100 px springs back (strict >), 101 px commits
// ===========================================================================

describe('commit threshold', () => {
  it('exactly 100 px horizontal travel does NOT commit — springs back (threshold is strict >)', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 300 + SWIPE_THRESHOLD, clientY: 301 })
    firePointer(card, 'pointerup', { clientX: 300 + SWIPE_THRESHOLD, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(0)
    wrapper.unmount()
  })

  it('99 px horizontal travel does not commit — spring-back, index unchanged', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 300 + (SWIPE_THRESHOLD - 1), clientY: 301 })
    firePointer(card, 'pointerup', { clientX: 300 + (SWIPE_THRESHOLD - 1), clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(0)
    wrapper.unmount()
  })

  it('101 px horizontal travel commits — flyAway fires, index advances', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301 })
    firePointer(card, 'pointerup', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// G1 — positive dragX (right drag): index advances to next coin
// G2 — negative dragX (left drag): index retreats to previous coin
// ===========================================================================

describe('G1 — left drag (negative dx): index advances to next coin (Phase 2 / iOS convention)', () => {
  it('left drag from index 0 → index 1 (swipe left = next)', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    doLeftDrag(card)
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(1)
    wrapper.unmount()
  })

  it('left drag from index 1 → index 2 (swipe left = next)', async () => {
    const wrapper = mountGallery({ startIndex: 1 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    doLeftDrag(card)
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(2)
    wrapper.unmount()
  })
})

describe('G2 — right drag (positive dx): index retreats to previous coin (Phase 2 / iOS convention)', () => {
  it('left drag from index 2 → index 1', async () => {
    const wrapper = mountGallery({ startIndex: 2 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    doRightDrag(card)
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(1)
    wrapper.unmount()
  })

  it('left drag from index 1 → index 0', async () => {
    const wrapper = mountGallery({ startIndex: 1 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    doRightDrag(card)
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(0)
    wrapper.unmount()
  })
})

// ===========================================================================
// G3 — sub-threshold drag springs back; index unchanged
// ===========================================================================

describe('G3 — sub-threshold drag (40 px) springs back', () => {
  it('40 px right drag does not commit; index stays at 0', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 340, clientY: 301 })
    firePointer(card, 'pointerup', { clientX: 340, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(0)
    wrapper.unmount()
  })

  it('spring-back applies transition style to active card during the 300 ms window', async () => {
    const wrapper = mountGallery()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 340, clientY: 301 })
    firePointer(card, 'pointerup', { clientX: 340, clientY: 301 })
    await nextTick()

    const style = wrapper.find('.active-card').attributes('style') ?? ''
    expect(style).toContain('transition: transform 0.3s ease')
    wrapper.unmount()
  })
})

// ===========================================================================
// G4 — KNOWN DEFECT: pointercancel after threshold commits instead of spring-back
// ===========================================================================

describe('G4 — pointercancel behavior', () => {
  it(
    'G4 REGRESSION — pointercancel after 120 px springs back (index unchanged); '
    + 'fixed: useSwipeGesture routes cancel → onCancel → springBack, never commits',
    async () => {
      const wrapper = mountGallery({ startIndex: 0 })
      const store = useCoinsStore()
      const card = prepCard(wrapper)

      firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
      firePointer(card, 'pointermove', { clientX: 300 + 120, clientY: 301 })
      firePointer(card, 'pointercancel', { clientX: 300 + 120, clientY: 301 })
      await vi.advanceTimersByTimeAsync(300)

      expect(store.galleryIndex).toBe(0)
      wrapper.unmount()
    },
  )

  it('pointercancel below threshold springs back correctly (typical device scenario with touch-action: none)', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    // 30 px — below the 100 px threshold that cancel also checks via shared onPointerUp
    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 330, clientY: 301 })
    firePointer(card, 'pointercancel', { clientX: 330, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(0)
    wrapper.unmount()
  })

  it('fresh qualifying gesture accepted after pointercancel resets state (no stuck-state)', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 320, clientY: 301 })
    firePointer(card, 'pointercancel', { clientX: 320, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)

    // Left drag = next (iOS convention); index 0 → 1
    doLeftDrag(card)
    await vi.advanceTimersByTimeAsync(300)

    expect(store.galleryIndex).toBe(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// G5 — tap with no travel routes to /coin/:id
// ===========================================================================

describe('G5 — tap routes to coin detail', () => {
  it('pointerdown + pointerup at same coords + click → router.push to coin detail', async () => {
    const coins = buildTestCoins(3)
    const wrapper = mountGallery({ coins, startIndex: 0 })
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointerup', { clientX: 300, clientY: 300 })
    await vi.advanceTimersByTimeAsync(300)
    card.dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(mockPush).toHaveBeenCalledWith(`/coin/${coins[0]!.id}`)
    wrapper.unmount()
  })
})

// ===========================================================================
// G6 — tap suppressed after drag (hasDragged flag)
// ===========================================================================

describe('G6 — hasDragged suppresses click navigation after a drag', () => {
  it('click after a >5 px sub-threshold drag does NOT navigate (hasDragged=true)', async () => {
    const coins = buildTestCoins(3)
    const wrapper = mountGallery({ coins, startIndex: 0 })
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 300 + HASDRAGGED_SLOP + 1, clientY: 301 })
    firePointer(card, 'pointerup', { clientX: 300 + HASDRAGGED_SLOP + 1, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    card.dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('hasDragged resets on next pointerdown — subsequent tap navigates', async () => {
    const coins = buildTestCoins(3)
    const wrapper = mountGallery({ coins, startIndex: 0 })
    const card = prepCard(wrapper)

    // Gesture 1: drag sets hasDragged
    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 310, clientY: 301 })
    firePointer(card, 'pointerup', { clientX: 310, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    card.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    expect(mockPush).not.toHaveBeenCalled()

    // Gesture 2: tap; hasDragged resets in onPointerDown
    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointerup', { clientX: 300, clientY: 300 })
    await vi.advanceTimersByTimeAsync(300)
    card.dispatchEvent(new MouseEvent('click', { bubbles: true }))

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// G7 — flip button @pointerdown.stop prevents drag start
// ===========================================================================

describe('G7 — flip button @pointerdown.stop prevents drag start', () => {
  it('pointerdown on .flip-btn does not start a gesture (setPointerCapture not called on card-stack)', () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const card = prepCard(wrapper)
    const stack = getStack(wrapper)
    const flipBtn = wrapper.find('.flip-btn').element

    firePointer(flipBtn, 'pointerdown', { clientX: 300, clientY: 300 })

    expect(stack.setPointerCapture as ReturnType<typeof vi.fn>).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('pointermove after flip-button pointerdown does not apply transform (no listener registered)', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const card = prepCard(wrapper)
    const flipBtn = wrapper.find('.flip-btn').element

    firePointer(flipBtn, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(card, 'pointermove', { clientX: 300 + 120, clientY: 301 })
    await nextTick()

    const style = wrapper.find('.active-card').attributes('style') ?? ''
    expect(style).not.toContain('translate')
    wrapper.unmount()
  })
})

// ===========================================================================
// G8 — drag past last coin on a page emits page-change
// ===========================================================================

describe('G8 — page boundary: drag past last coin emits page-change', () => {
  it('left drag at last coin on page 1 (more pages exist) emits page-change to page 2', async () => {
    const coins = buildTestCoins(3)
    const wrapper = mountGallery({ coins, total: 6, page: 1, perPage: 3, startIndex: 2 })
    const card = prepCard(wrapper)

    doLeftDrag(card)
    await vi.advanceTimersByTimeAsync(300)

    expect(wrapper.emitted('page-change')).toEqual([[2]])
    wrapper.unmount()
  })

  it('right drag at first coin on page 2 emits page-change to page 1', async () => {
    const coins = buildTestCoins(3)
    const wrapper = mountGallery({ coins, total: 6, page: 2, perPage: 3, startIndex: 0 })
    const card = prepCard(wrapper)

    doRightDrag(card)
    await vi.advanceTimersByTimeAsync(300)

    expect(wrapper.emitted('page-change')).toEqual([[1]])
    wrapper.unmount()
  })
})

// ===========================================================================
// G9 — new pointerdown ignored during fly-away animation (isAnimating gate)
// ===========================================================================

describe('G9 — pointerdown ignored during fly-away animation', () => {
  it('second pointerdown mid-animation does not call setPointerCapture (isAnimating=true guard)', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)
    const stack = getStack(wrapper)

    doLeftDrag(card)  // left drag = advance; first commit
    // Animation in progress; re-stub stack spy to detect any NEW capture call
    const secondCaptureSpy = vi.fn()
    stack.setPointerCapture = secondCaptureSpy

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    expect(secondCaptureSpy).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(300)
    expect(store.galleryIndex).toBe(1)  // exactly one commit
    wrapper.unmount()
  })
})

// ===========================================================================
// G10 — .card-stack declares touch-action: none
// Source check is justified: jsdom does not process scoped CSS.
// ===========================================================================

describe('G10 — .card-stack touch-action: none', () => {
  it('.card-stack CSS block contains touch-action: none', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'components', 'SwipeGallery.vue'),
      'utf-8',
    )
    expect(source).toContain('.card-stack {')
    expect(source).toContain('touch-action: none')
  })
})

// ===========================================================================
// Pointer capture: acquired on pointerdown, released on pointerup
// ===========================================================================

describe('pointer capture — acquired and released', () => {
  it('setPointerCapture called with the pointer ID on pointerdown', () => {
    const wrapper = mountGallery()
    const card = prepCard(wrapper)
    const stack = getStack(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300, pointerId: 7 })

    expect(stack.setPointerCapture as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(7)
    wrapper.unmount()
  })

  it('releasePointerCapture called with the pointer ID on pointerup', () => {
    const wrapper = mountGallery()
    const card = prepCard(wrapper)
    const stack = getStack(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300, pointerId: 7 })
    firePointer(card, 'pointerup', { clientX: 300, clientY: 300, pointerId: 7 })

    expect(stack.releasePointerCapture as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(7)
    wrapper.unmount()
  })
})

// ===========================================================================
// Live transform feedback during drag
// ===========================================================================

describe('live transform feedback during drag', () => {
  it('active card has translate transform reflecting current dragX during pointermove', async () => {
    const wrapper = mountGallery()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 200, clientY: 200 })
    firePointer(card, 'pointermove', { clientX: 260, clientY: 201 })  // dragX = 60
    await nextTick()

    const style = wrapper.find('.active-card').attributes('style') ?? ''
    expect(style).toContain('translate(60px')
    wrapper.unmount()
  })

  it('transform cleared after spring-back timer completes', async () => {
    const wrapper = mountGallery()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 200, clientY: 200 })
    firePointer(card, 'pointermove', { clientX: 240, clientY: 201 })
    firePointer(card, 'pointerup', { clientX: 240, clientY: 201 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    const style = wrapper.find('.active-card').attributes('style') ?? ''
    expect(style).toBeFalsy()
    wrapper.unmount()
  })

  it('left-hint opacity=1 at full left drag (dragX=-100); right-hint opacity=0', async () => {
    const wrapper = mountGallery()
    const card = prepCard(wrapper)

    firePointer(card, 'pointerdown', { clientX: 300, clientY: 200 })
    firePointer(card, 'pointermove', { clientX: 200, clientY: 201 })  // dragX = -100
    await nextTick()

    const leftStyle = wrapper.find('.left-hint').attributes('style') ?? ''
    const rightStyle = wrapper.find('.right-hint').attributes('style') ?? ''
    expect(leftStyle).toContain('opacity: 1')
    expect(rightStyle).toContain('opacity: 0')
    wrapper.unmount()
  })
})

// ===========================================================================
// Cleanup — animation timers cleared on unmount
// ===========================================================================

describe('cleanup — animation timers cleared on unmount', () => {
  it('unmounting during animation does not throw (clearTimeout was called for all timers)', async () => {
    const wrapper = mountGallery()
    const card = prepCard(wrapper)

    doLeftDrag(card)
    wrapper.unmount()  // mid-animation

    await expect(vi.advanceTimersByTimeAsync(300)).resolves.not.toThrow()
  })
})

// ===========================================================================
// No double commit on rapid gestures
// ===========================================================================

describe('no double commit on rapid gestures', () => {
  it('isAnimating gate prevents a second commit before the first animation ends', async () => {
    const wrapper = mountGallery({ startIndex: 0 })
    const store = useCoinsStore()
    const card = prepCard(wrapper)
    const stack = getStack(wrapper)

    doLeftDrag(card)  // first commit (left = advance) → isAnimating = true

    const secondCaptureSpy = vi.fn()
    stack.setPointerCapture = secondCaptureSpy
    firePointer(card, 'pointerdown', { clientX: 300, clientY: 300 })
    expect(secondCaptureSpy).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(300)
    expect(store.galleryIndex).toBe(1)  // exactly one change

    // Third gesture after animation succeeds
    doLeftDrag(card)
    await vi.advanceTimersByTimeAsync(300)
    expect(store.galleryIndex).toBe(2)

    wrapper.unmount()
  })
})
