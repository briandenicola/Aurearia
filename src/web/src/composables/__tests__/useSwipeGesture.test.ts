/**
 * Tests for the useSwipeGesture shared primitive.
 *
 * Design authority: .squad/decisions/inbox/maximus-pwa-swipe-reliability-review.md §§9-13
 *
 * Covers: touchAction apply/restore, axis lock, arced swipe, exclusions,
 * cancel semantics, enabled reactive gate, button/anchor starts, stale
 * text selection irrelevant, multi-touch abort, pointer type filter.
 */

import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { useSwipeGesture } from '@/composables/useSwipeGesture'

// ---------------------------------------------------------------------------
// Pointer event synthesis
// ---------------------------------------------------------------------------

function firePointer(
  el: Element,
  type: string,
  opts: {
    clientX?: number
    clientY?: number
    pointerId?: number
    pointerType?: string
    isPrimary?: boolean
    target?: Element
  } = {},
) {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperties(event, {
    pointerId: { value: opts.pointerId ?? 1 },
    pointerType: { value: opts.pointerType ?? 'touch' },
    isPrimary: { value: opts.isPrimary ?? true },
    clientX: { value: opts.clientX ?? 0 },
    clientY: { value: opts.clientY ?? 0 },
  })
  ;(opts.target ?? el).dispatchEvent(event)
}

// Canonical qualifying left swipe (dx=-80, dy=0)
function doLeftSwipe(el: HTMLElement, sx = 300, sy = 400) {
  firePointer(el, 'pointerdown', { clientX: sx, clientY: sy })
  firePointer(el, 'pointermove', { clientX: sx - 12, clientY: sy + 1 })
  firePointer(el, 'pointermove', { clientX: sx - 80, clientY: sy + 2 })
  firePointer(el, 'pointerup', { clientX: sx - 80, clientY: sy + 2 })
}

// ---------------------------------------------------------------------------
// Harness factories
// ---------------------------------------------------------------------------

function mountWithGesture(overrides: Parameters<typeof useSwipeGesture>[1] = {}) {
  const onCommit = vi.fn()
  const onCancel = vi.fn()
  const onMove = vi.fn()
  const harness = defineComponent({
    setup() {
      const el = ref<HTMLElement | null>(null)
      useSwipeGesture(el, { threshold: 64, lockSlop: 10, ...overrides, onCommit, onCancel, onMove })
      return { el }
    },
    template: '<div ref="el"></div>',
  })
  const wrapper = mount(harness)
  return { wrapper, onCommit, onCancel, onMove, el: wrapper.element as HTMLElement }
}

// ---------------------------------------------------------------------------
// touchAction apply/restore (proxy for R1)
// ---------------------------------------------------------------------------

describe('touchAction apply and restore', () => {
  it('sets touch-action on the container element while attached (pan-y)', () => {
    const { el } = mountWithGesture({ touchAction: 'pan-y' })
    expect(el.style.touchAction).toBe('pan-y')
  })

  it('sets touch-action none on the container while attached', () => {
    const { el } = mountWithGesture({ touchAction: 'none' })
    expect(el.style.touchAction).toBe('none')
  })

  it('restores prior inline touch-action on unmount', () => {
    const harness = defineComponent({
      setup() {
        const el = ref<HTMLElement | null>(null)
        useSwipeGesture(el, { touchAction: 'pan-y' })
        return { el }
      },
      template: '<div ref="el" style="touch-action: auto"></div>',
    })
    const wrapper = mount(harness)
    const el = wrapper.element as HTMLElement
    expect(el.style.touchAction).toBe('pan-y')
    wrapper.unmount()
    expect(el.style.touchAction).toBe('auto')
  })

  it('restores empty touch-action when no prior inline value existed', () => {
    const { wrapper, el } = mountWithGesture({ touchAction: 'pan-y' })
    expect(el.style.touchAction).toBe('pan-y')
    wrapper.unmount()
    expect(el.style.touchAction).toBe('')
  })

  it('restores touch-action when enabled gate is turned off', async () => {
    const enabled = ref(true)
    const { el } = mountWithGesture({ touchAction: 'pan-y', enabled })
    expect(el.style.touchAction).toBe('pan-y')
    enabled.value = false
    await nextTick()
    expect(el.style.touchAction).toBe('')
  })

  it('does not touch the element touch-action when touchAction is null', () => {
    const { el } = mountWithGesture({ touchAction: null })
    expect(el.style.touchAction).toBe('')
  })
})

// ---------------------------------------------------------------------------
// Arced swipe: horizontal axis locked, large vertical drift still commits (R2)
// ---------------------------------------------------------------------------

describe('arced swipe -- horizontal lock with vertical drift navigates', () => {
  it('dx=80 dy=50 navigates when horizontal axis locked first (no 2:1 dominance gate)', () => {
    const { el, onCommit } = mountWithGesture()
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    // dx=12 dy=3 -- horizontal axis locked
    firePointer(el, 'pointermove', { clientX: 288, clientY: 403 })
    // Release at dx=80, dy=50 -- large vertical drift, but axis is already locked
    firePointer(el, 'pointerup', { clientX: 220, clientY: 450 })
    expect(onCommit).toHaveBeenCalledTimes(1)
    expect(onCommit).toHaveBeenCalledWith(-1)
  })

  it('direction is -1 for leftward drag', () => {
    const { el, onCommit } = mountWithGesture()
    doLeftSwipe(el)
    expect(onCommit).toHaveBeenCalledWith(-1)
  })

  it('direction is 1 for rightward drag', () => {
    const { el, onCommit } = mountWithGesture()
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 312, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 380, clientY: 402 })
    expect(onCommit).toHaveBeenCalledWith(1)
  })
})

// ---------------------------------------------------------------------------
// Button and anchor starts -- should NOT suppress (R3)
// ---------------------------------------------------------------------------

describe('button and anchor starts do not suppress navigation', () => {
  const allowedTags = ['button', 'a'] as const

  for (const tag of allowedTags) {
    it(`gesture starting on <${tag}> navigates (no button/anchor exclusion by default)`, () => {
      const { wrapper, onCommit } = mountWithGesture()
      const el = wrapper.element as HTMLElement
      const child = document.createElement(tag)
      el.appendChild(child)

      firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, target: child })
      firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
      firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })

      expect(onCommit).toHaveBeenCalledTimes(1)
      wrapper.unmount()
    })
  }
})

// ---------------------------------------------------------------------------
// Exclusion selector (R4)
// ---------------------------------------------------------------------------

describe('exclusion selector', () => {
  const EXCLUDE = 'input, textarea, select, [contenteditable="true"], [data-swipe-ignore]'

  const suppressedTags = ['input', 'textarea', 'select'] as const
  for (const tag of suppressedTags) {
    it(`gesture starting on <${tag}> is suppressed when in exclude`, () => {
      const { wrapper, onCommit } = mountWithGesture({ exclude: EXCLUDE })
      const el = wrapper.element as HTMLElement
      const child = document.createElement(tag)
      el.appendChild(child)

      firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, target: child })
      firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
      firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })

      expect(onCommit).not.toHaveBeenCalled()
      wrapper.unmount()
    })
  }

  it('gesture starting on [contenteditable="true"] is suppressed', () => {
    const { wrapper, onCommit } = mountWithGesture({ exclude: EXCLUDE })
    const el = wrapper.element as HTMLElement
    const child = document.createElement('div')
    child.setAttribute('contenteditable', 'true')
    el.appendChild(child)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, target: child })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })

    expect(onCommit).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('gesture starting on [data-swipe-ignore] is suppressed', () => {
    const { wrapper, onCommit } = mountWithGesture({ exclude: EXCLUDE })
    const el = wrapper.element as HTMLElement
    const child = document.createElement('div')
    child.setAttribute('data-swipe-ignore', '')
    el.appendChild(child)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, target: child })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })

    expect(onCommit).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('nested [data-swipe-ignore] inside parent suppresses via closest()', () => {
    const { wrapper, onCommit } = mountWithGesture({ exclude: EXCLUDE })
    const el = wrapper.element as HTMLElement
    const outer = document.createElement('div')
    const inner = document.createElement('div')
    inner.setAttribute('data-swipe-ignore', '')
    outer.appendChild(inner)
    el.appendChild(outer)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, target: inner })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })

    expect(onCommit).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('gesture starting on plain div (no exclude match) navigates', () => {
    const { wrapper, onCommit } = mountWithGesture({ exclude: EXCLUDE })
    const el = wrapper.element as HTMLElement
    const child = document.createElement('div')
    el.appendChild(child)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, target: child })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })

    expect(onCommit).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ---------------------------------------------------------------------------
// Text selection irrelevant (R6 -- stale selection no longer a gate)
// ---------------------------------------------------------------------------

describe('text selection does not gate commits', () => {
  it('commit succeeds even when window.getSelection returns non-empty text', () => {
    vi.spyOn(window, 'getSelection').mockReturnValue({ toString: () => 'selected text' } as Selection)
    const { el, onCommit } = mountWithGesture()
    doLeftSwipe(el)
    expect(onCommit).toHaveBeenCalledTimes(1)
    vi.restoreAllMocks()
  })
})

// ---------------------------------------------------------------------------
// Cancel semantics (R7)
// ---------------------------------------------------------------------------

describe('cancel semantics', () => {
  it('pointercancel aborts the gesture and calls onCancel', () => {
    const { el, onCommit, onCancel } = mountWithGesture()
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointercancel', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })
    expect(onCommit).not.toHaveBeenCalled()
    expect(onCancel).toHaveBeenCalled()
  })

  it('fresh gesture after pointercancel navigates -- no stuck state (R7)', () => {
    const { el, onCommit } = mountWithGesture()
    // First gesture -- cancelled
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointercancel', { clientX: 288, clientY: 401 })
    // Second gesture -- clean and qualifying
    doLeftSwipe(el)
    expect(onCommit).toHaveBeenCalledTimes(1)
  })

  it('lostpointercapture aborts the gesture', () => {
    const { el, onCommit, onCancel } = mountWithGesture()
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'lostpointercapture', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })
    expect(onCommit).not.toHaveBeenCalled()
    expect(onCancel).toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Reactive enabled gate
// ---------------------------------------------------------------------------

describe('reactive enabled gate', () => {
  it('enabled=false suppresses gesture start', () => {
    const enabled = ref(false)
    const { el, onCommit } = mountWithGesture({ enabled })
    doLeftSwipe(el)
    expect(onCommit).not.toHaveBeenCalled()
  })

  it('enabled=true allows commit', () => {
    const enabled = ref(true)
    const { el, onCommit } = mountWithGesture({ enabled })
    doLeftSwipe(el)
    expect(onCommit).toHaveBeenCalledTimes(1)
  })

  it('enabled gate evaluated at commit: disabling mid-gesture blocks navigation', async () => {
    const enabled = ref(true)
    const { el, onCommit } = mountWithGesture({ enabled })
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    enabled.value = false
    await nextTick()
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })
    expect(onCommit).not.toHaveBeenCalled()
  })

  it('toggling off detaches (touch-action restored), toggling on re-attaches', async () => {
    const enabled = ref(true)
    const { el } = mountWithGesture({ enabled, touchAction: 'pan-y' })
    expect(el.style.touchAction).toBe('pan-y')
    enabled.value = false
    await nextTick()
    expect(el.style.touchAction).toBe('')
    enabled.value = true
    await nextTick()
    expect(el.style.touchAction).toBe('pan-y')
  })
})

// ---------------------------------------------------------------------------
// Pointer type filter
// ---------------------------------------------------------------------------

describe('pointer type filter', () => {
  it('mouse pointer ignored when pointerTypes is touch-only', () => {
    const { el, onCommit } = mountWithGesture({ pointerTypes: ['touch'] })
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, pointerType: 'mouse' })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401, pointerType: 'mouse' })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402, pointerType: 'mouse' })
    expect(onCommit).not.toHaveBeenCalled()
  })

  it('touch pointer accepted when pointerTypes is touch-only', () => {
    const { el, onCommit } = mountWithGesture({ pointerTypes: ['touch'] })
    doLeftSwipe(el)
    expect(onCommit).toHaveBeenCalledTimes(1)
  })
})

// ---------------------------------------------------------------------------
// Multi-touch abort
// ---------------------------------------------------------------------------

describe('multi-touch abort', () => {
  it('second non-primary pointer aborts active gesture', () => {
    const { el, onCommit, onCancel } = mountWithGesture()
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400, pointerId: 1 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401, pointerId: 1 })
    // Second finger arrives
    firePointer(el, 'pointerdown', { clientX: 200, clientY: 400, pointerId: 2, isPrimary: false })
    // First finger releases at qualifying distance -- gesture was cancelled
    firePointer(el, 'pointerup', { clientX: 220, clientY: 402, pointerId: 1 })
    expect(onCommit).not.toHaveBeenCalled()
    expect(onCancel).toHaveBeenCalled()
  })
})

// ---------------------------------------------------------------------------
// Edge guard
// ---------------------------------------------------------------------------

describe('edge guard', () => {
  beforeEach(() => {
    Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true, writable: true })
  })

  it('ignores gesture starting inside the left edge guard', () => {
    const { el, onCommit } = mountWithGesture({ edgeGuard: 24 })
    firePointer(el, 'pointerdown', { clientX: 10, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: -2, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: -74, clientY: 402 })
    expect(onCommit).not.toHaveBeenCalled()
  })

  it('accepts gesture starting at the edge guard boundary', () => {
    const { el, onCommit } = mountWithGesture({ edgeGuard: 24 })
    firePointer(el, 'pointerdown', { clientX: 24, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 12, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: -40, clientY: 402 })
    expect(onCommit).toHaveBeenCalledTimes(1)
  })
})

// ---------------------------------------------------------------------------
// Late-attach: target ref that becomes non-null after onMounted (e.g. v-else-if)
// ---------------------------------------------------------------------------

describe('late-attach -- target element inside v-if becomes available after data loads', () => {
  it('attaches and accepts gestures when target ref is set after onMounted', async () => {
    const onCommit = vi.fn()
    const target = ref<HTMLElement | null>(null)
    // Mount a harness where the target ref starts null
    const harness = defineComponent({
      setup() {
        useSwipeGesture(target, { threshold: 64, lockSlop: null, onCommit })
        return {}
      },
      template: '<div></div>',
    })
    const wrapper = mount(harness)

    // Simulate target appearing after a conditional render (v-else-if data load)
    const el = document.createElement('div')
    el.setPointerCapture = vi.fn()
    el.releasePointerCapture = vi.fn()
    document.body.appendChild(el)
    target.value = el
    await nextTick()  // flush post-watchers (attach fires)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })

    expect(onCommit).toHaveBeenCalledTimes(1)

    document.body.removeChild(el)
    wrapper.unmount()
  })

  it('detaches cleanly when target ref is set back to null (element → null cycle)', async () => {
    const onCommit = vi.fn()
    const target = ref<HTMLElement | null>(null)
    const harness = defineComponent({
      setup() { useSwipeGesture(target, { threshold: 64, lockSlop: null, onCommit }); return {} },
      template: '<div></div>',
    })
    const wrapper = mount(harness)

    const el = document.createElement('div')
    el.setPointerCapture = vi.fn()
    el.releasePointerCapture = vi.fn()
    document.body.appendChild(el)

    target.value = el
    await nextTick()
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })
    expect(onCommit).toHaveBeenCalledTimes(1)

    // Detach by nulling the ref (simulates component unmounting v-else-if block)
    target.value = null
    await nextTick()
    onCommit.mockClear()
    // Gestures on the old element fire no callbacks after detach
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 })
    expect(onCommit).not.toHaveBeenCalled()

    document.body.removeChild(el)
    wrapper.unmount()
  })

  it('direct A-to-B target replacement migrates gesture to B: commit fires from B, listeners removed from A, touch-action restored on A', async () => {
    // B4 regression: the watcher now handles el !== attachedElement, so a direct A->B swap
    // detaches from A (restores touchAction, removes listeners) and attaches to B.
    const onCommit = vi.fn()
    const target = ref<HTMLElement | null>(null)
    const harness = defineComponent({
      setup() { useSwipeGesture(target, { threshold: 64, lockSlop: null, touchAction: 'pan-y', onCommit }); return {} },
      template: '<div></div>',
    })
    const wrapper = mount(harness)

    const elA = document.createElement('div')
    elA.setPointerCapture = vi.fn()
    elA.releasePointerCapture = vi.fn()
    document.body.appendChild(elA)
    const elB = document.createElement('div')
    elB.setPointerCapture = vi.fn()
    elB.releasePointerCapture = vi.fn()
    document.body.appendChild(elB)

    // Attach to A; gesture commits from A
    target.value = elA
    await nextTick()
    firePointer(elA, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(elA, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(elA, 'pointerup',   { clientX: 236, clientY: 402 })
    expect(onCommit).toHaveBeenCalledTimes(1)
    onCommit.mockClear()

    // Direct A-to-B: watcher detaches from A and attaches to B
    target.value = elB
    await nextTick()

    // Gesture on B commits (migration successful)
    firePointer(elB, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(elB, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(elB, 'pointerup',   { clientX: 236, clientY: 402 })
    expect(onCommit).toHaveBeenCalledTimes(1)

    // No listeners remain on A: gestures on A do not fire callbacks
    onCommit.mockClear()
    firePointer(elA, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(elA, 'pointermove', { clientX: 288, clientY: 401 })
    firePointer(elA, 'pointerup',   { clientX: 236, clientY: 402 })
    expect(onCommit).not.toHaveBeenCalled()

    // touch-action was restored on A when it was detached
    expect(elA.style.touchAction).toBe('')

    document.body.removeChild(elA)
    document.body.removeChild(elB)
    wrapper.unmount()
  })
})
// ---------------------------------------------------------------------------
// Threshold semantics — >= boundary, configurable via consumers
// ---------------------------------------------------------------------------

describe('threshold semantics -- commits at >= threshold, not at threshold-1', () => {
  it('commits when displacement equals threshold exactly (>= semantics; threshold: 64)', () => {
    const { el, onCommit } = mountWithGesture({ threshold: 64 })
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 }) // lock axis
    firePointer(el, 'pointerup', { clientX: 236, clientY: 402 }) // dx = 64
    expect(onCommit).toHaveBeenCalledTimes(1)
  })

  it('cancels when displacement is threshold-1 (63px for threshold: 64)', () => {
    const { el, onCommit, onCancel } = mountWithGesture({ threshold: 64 })
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 }) // lock axis
    firePointer(el, 'pointerup', { clientX: 237, clientY: 402 }) // dx = 63
    expect(onCommit).not.toHaveBeenCalled()
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  it('commits at exactly 101px when threshold: 101 (Gallery/Tray strict >100 semantics)', () => {
    // Gallery and Tray pass threshold: 101 to get "commit only when |dx| > 100" semantics.
    // Because primitive commits at >= threshold, threshold:101 means commits at 101+, never at 100.
    const { el, onCommit } = mountWithGesture({ threshold: 101 })
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 }) // lock axis
    firePointer(el, 'pointerup', { clientX: 199, clientY: 402 }) // dx = 101
    expect(onCommit).toHaveBeenCalledTimes(1)
  })

  it('cancels at exactly 100px when threshold: 101 (strict >100 semantics)', () => {
    const { el, onCommit, onCancel } = mountWithGesture({ threshold: 101 })
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(el, 'pointermove', { clientX: 288, clientY: 401 }) // lock axis
    firePointer(el, 'pointerup', { clientX: 200, clientY: 402 }) // dx = 100
    expect(onCommit).not.toHaveBeenCalled()
    expect(onCancel).toHaveBeenCalledTimes(1)
  })
})
// ---------------------------------------------------------------------------
// Source invariants
// ---------------------------------------------------------------------------

describe('source invariants', () => {
  it('all addEventListener calls in useSwipeGesture are registered with { passive: true }', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    const addCount = (source.match(/\.addEventListener\(/g) ?? []).length
    const passiveCount = (source.match(/passive:\s*true/g) ?? []).length
    expect(addCount).toBeGreaterThan(0)
    expect(passiveCount).toBe(addCount)
  })

  it('useSwipeGesture source contains no preventDefault call', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    expect(source).not.toContain('preventDefault')
  })

  it('useSwipeGesture source does not attach listeners to window or document', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    expect(source).not.toMatch(/window\.addEventListener/)
    expect(source).not.toMatch(/document\.addEventListener/)
  })

  it('useSwipeGesture has no module-level mutable state', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    // Bare let/var at column 0 -- not inside a function body.
    const bareLetOrVar = source.match(/^(?!\/\/)(?:let|var)\s+/m)
    expect(bareLetOrVar).toBeNull()
  })
})