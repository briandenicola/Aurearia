/**
 * Contract-based regression tests for the PWA-only coin-detail swipe navigation composable.
 *
 * Design authority: .squad/decisions/inbox/maximus-pwa-detail-swipe-review.md
 * Covers criteria 1-16 from the acceptance checklist plus the USER CONTRACT in the spawn prompt.
 *
 * Tests are authored against the PUBLIC CONTRACT of useCoinDetailSwipeNav.
 * Adjust to the composable exported API if Aurelia's implementation differs in
 * non-observable private details, but never loosen the behavioral assertions.
 *
 * NOTE: All tests will fail until Aurelia delivers src/composables/useCoinDetailSwipeNav.ts
 *       -- that is the expected BLOCK state.
 */

import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import type { Ref } from 'vue'
import { COIN_DETAIL_SECTIONS, SECTION_ORDER } from '@/constants/coinDetailSections'
import {
  useCoinDetailSwipeNav,
  SWIPE_THRESHOLD,
  EDGE_GUARD,
  AXIS_SLOP,
} from '@/composables/useCoinDetailSwipeNav'

// ---------------------------------------------------------------------------
// Hoisted mocks (evaluated before imports are resolved)
// ---------------------------------------------------------------------------

const mockIsPwa = vi.hoisted(() => ({ value: true }))

const mockRouteState = vi.hoisted(() => ({
  path: '/coin/42',
  params: { id: '42' } as Record<string, string>,
  query: {} as Record<string, string>,
  hash: '',
}))

const mockPush = vi.hoisted(() => vi.fn())

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: mockIsPwa.value }),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => mockRouteState,
    useRouter: () => ({ push: mockPush }),
  }
})

// Auth store mock — reactive so that watch() inside the composable tracks user changes.
// Default: pwaSwipeNavEnabled=true so all existing tests keep their current behavior.
const authStore = vi.hoisted(() => ({
  proxy: null as unknown as { user: { pwaSwipeNavEnabled?: boolean } | null },
}))

vi.mock('@/stores/auth', async () => {
  const { reactive } = await import('vue')
  authStore.proxy = reactive({ user: { pwaSwipeNavEnabled: true } as { pwaSwipeNavEnabled?: boolean } | null })
  return { useAuthStore: () => authStore.proxy }
})

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const COIN_ID = 42

// Canonical 8-stop swipe order derived from the authoritative menu constants.
// Overview is prefixed; SECTION_ORDER drives the remaining seven stops.
// This is the SINGLE SOURCE OF TRUTH the composable must also use at runtime.
const SWIPE_STOPS = [
  `/coin/${COIN_ID}`,
  ...SECTION_ORDER.map(id => COIN_DETAIL_SECTIONS[id].route(COIN_ID)),
]

// ---------------------------------------------------------------------------
// Pointer event synthesis (follows ZoomableSurface.test.ts pattern)
// ---------------------------------------------------------------------------

function firePointer(
  container: Element,
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
  ;(opts.target ?? container).dispatchEvent(event)
}

// Canonical left swipe: net dx = -SWIPE_MIN_DISTANCE, clean horizontal axis lock.
function doLeftSwipe(container: HTMLElement, startX = 300, startY = 400, opts: { pointerId?: number } = {}) {
  const pid = opts.pointerId ?? 1
  firePointer(container, 'pointerdown', { clientX: startX, clientY: startY, pointerId: pid })
  // First move exceeds slop in horizontal axis only -> horizontal lock
  firePointer(container, 'pointermove', { clientX: startX - (AXIS_SLOP + 2), clientY: startY + 1, pointerId: pid })
  // Second move reaches minimum distance (dy << dx/2)
  firePointer(container, 'pointermove', { clientX: startX - SWIPE_THRESHOLD, clientY: startY + 2, pointerId: pid })
  firePointer(container, 'pointerup', { clientX: startX - SWIPE_THRESHOLD, clientY: startY + 2, pointerId: pid })
}

// Canonical right swipe: net dx = +SWIPE_MIN_DISTANCE.
function doRightSwipe(container: HTMLElement, startX = 300, startY = 400, opts: { pointerId?: number } = {}) {
  const pid = opts.pointerId ?? 1
  firePointer(container, 'pointerdown', { clientX: startX, clientY: startY, pointerId: pid })
  firePointer(container, 'pointermove', { clientX: startX + (AXIS_SLOP + 2), clientY: startY + 1, pointerId: pid })
  firePointer(container, 'pointermove', { clientX: startX + SWIPE_THRESHOLD, clientY: startY + 2, pointerId: pid })
  firePointer(container, 'pointerup', { clientX: startX + SWIPE_THRESHOLD, clientY: startY + 2, pointerId: pid })
}

// ---------------------------------------------------------------------------
// Harness factory
// ---------------------------------------------------------------------------

function mountHarness(options?: {
  enabled?: Ref<boolean>
  routePath?: string
  coinId?: number
  query?: Record<string, string>
  hash?: string
}) {
  const id = options?.coinId ?? COIN_ID
  mockRouteState.path = options?.routePath ?? `/coin/${id}`
  mockRouteState.params = { id: String(id) }
  mockRouteState.query = options?.query ?? {}
  mockRouteState.hash = options?.hash ?? ''

  const enabled = options?.enabled

  const harness = defineComponent({
    setup() {
      const containerRef = ref<HTMLElement | null>(null)
      useCoinDetailSwipeNav(containerRef, enabled !== undefined ? { enabled } : undefined)
      return { containerRef }
    },
    template: '<div ref="containerRef"></div>',
  })

  return mount(harness)
}

// ---------------------------------------------------------------------------
// Reset shared mock state before each test
// ---------------------------------------------------------------------------

beforeEach(() => {
  mockPush.mockClear()
  mockIsPwa.value = true
  mockRouteState.path = `/coin/${COIN_ID}`
  mockRouteState.params = { id: String(COIN_ID) }
  mockRouteState.query = {}
  mockRouteState.hash = ''
  Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true, writable: true })
  if (authStore.proxy) { authStore.proxy.user = { pwaSwipeNavEnabled: true } }
})

// ===========================================================================
// 1. Exported constants -- thresholds must be named exports (design review §6)
// ===========================================================================

describe('exported threshold constants', () => {
  it('SWIPE_THRESHOLD is 64', () => {
    expect(SWIPE_THRESHOLD).toBe(64)
  })

  it('EDGE_GUARD is 24', () => {
    expect(EDGE_GUARD).toBe(24)
  })

  it('AXIS_SLOP is 10', () => {
    expect(AXIS_SLOP).toBe(10)
  })
})

// ===========================================================================
// 2. Route ordering derived from authoritative menu constants (criterion 11)
// ===========================================================================

describe('route ordering from menu constants', () => {
  it('canonical stop list has exactly 8 entries', () => {
    expect(SWIPE_STOPS).toHaveLength(8)
  })

  it('first stop is the overview path /coin/:id', () => {
    expect(SWIPE_STOPS[0]).toBe(`/coin/${COIN_ID}`)
  })

  it('stops 1-7 match SECTION_ORDER in the declared menu order', () => {
    const expectedSectionPaths = SECTION_ORDER.map(id => COIN_DETAIL_SECTIONS[id].route(COIN_ID))
    expect(SWIPE_STOPS.slice(1)).toEqual(expectedSectionPaths)
  })

  it('declares shipment, journal, health, notes, actions, analysis, valuation in that order', () => {
    expect(SWIPE_STOPS).toEqual([
      `/coin/${COIN_ID}`,
      `/coin/${COIN_ID}/shipment`,
      `/coin/${COIN_ID}/journal`,
      `/coin/${COIN_ID}/health`,
      `/coin/${COIN_ID}/notes`,
      `/coin/${COIN_ID}/actions`,
      `/coin/${COIN_ID}/analysis`,
      `/coin/${COIN_ID}/valuation`,
    ])
  })

  it('contains no sell or copy path segment', () => {
    for (const stop of SWIPE_STOPS) {
      expect(stop).not.toMatch(/\/(sell|copy)/i)
    }
  })

  it('SECTION_ORDER has exactly 7 entries', () => {
    expect(SECTION_ORDER).toHaveLength(7)
  })
})

// ===========================================================================
// 3. Left swipe advances (criteria 1, coin ID preserved)
// ===========================================================================

describe('left swipe -- advance one stop', () => {
  it('advances from each non-terminal stop to the next in order', () => {
    for (let i = 0; i < SWIPE_STOPS.length - 1; i++) {
      const wrapper = mountHarness({ routePath: SWIPE_STOPS[i] })
      const container = wrapper.element as HTMLElement

      doLeftSwipe(container)

      expect(mockPush).toHaveBeenCalledTimes(1)
      expect(mockPush).toHaveBeenCalledWith(
        expect.objectContaining({ path: SWIPE_STOPS[i + 1] }),
      )

      mockPush.mockClear()
      wrapper.unmount()
    }
  })

  it('preserves the coin ID in the advance navigation path', () => {
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}` })
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)

    const pushed = mockPush.mock.calls[0]?.[0] as { path: string }
    expect(pushed.path).toContain(`/coin/${COIN_ID}`)
    wrapper.unmount()
  })
})

// ===========================================================================
// 4. Right swipe goes back (criterion 1)
// ===========================================================================

describe('right swipe -- go back one stop', () => {
  it('goes back from each non-initial stop to the previous in order', () => {
    for (let i = SWIPE_STOPS.length - 1; i > 0; i--) {
      const wrapper = mountHarness({ routePath: SWIPE_STOPS[i] })
      const container = wrapper.element as HTMLElement

      doRightSwipe(container)

      expect(mockPush).toHaveBeenCalledTimes(1)
      expect(mockPush).toHaveBeenCalledWith(
        expect.objectContaining({ path: SWIPE_STOPS[i - 1] }),
      )

      mockPush.mockClear()
      wrapper.unmount()
    }
  })
})

// ===========================================================================
// 5. No wrap at boundaries (criterion 2)
// ===========================================================================

describe('boundary -- no wrap', () => {
  it('right swipe on Overview (index 0) does not navigate', () => {
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}` })
    doRightSwipe(wrapper.element as HTMLElement)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('left swipe on Value Trend (index 7) does not navigate', () => {
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}/valuation` })
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('boundary swipe does not mutate any DOM state', async () => {
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}` })
    const container = wrapper.element as HTMLElement
    const classListBefore = container.className

    doRightSwipe(container)
    await nextTick()

    expect(container.className).toBe(classListBefore)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

// ===========================================================================
// 6. Minimum distance threshold: 63 px rejected, 64 px accepted (criterion 3)
// ===========================================================================

describe('minimum distance threshold', () => {
  it('63 px horizontal travel does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    firePointer(container, 'pointermove', { clientX: startX - (AXIS_SLOP + 2), clientY: 401 })
    firePointer(container, 'pointermove', { clientX: startX - (SWIPE_THRESHOLD - 1), clientY: 402 })
    firePointer(container, 'pointerup', { clientX: startX - (SWIPE_THRESHOLD - 1), clientY: 402 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('exactly 64 px horizontal travel does navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// 7. Axis dominance (criterion 4)
// ===========================================================================

describe('axis dominance', () => {
  it('vertical-dominant drag (dy > dx at slop point) does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    // dy=12 > dx=2 at first move past slop -- vertical axis locked, gesture abandoned
    firePointer(container, 'pointermove', { clientX: startX + 2, clientY: 412 })
    // Large subsequent horizontal movement cannot rescue the abandoned gesture
    firePointer(container, 'pointermove', { clientX: startX + 100, clientY: 415 })
    firePointer(container, 'pointerup', { clientX: startX + 100, clientY: 415 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('arced swipe (axis locked horizontal, vertical drift at release) navigates -- 2:1 dominance gate removed', () => {
    // dx=64 meets distance; dy=40; axis locked horizontal at first move (dx=12 > dy=3).
    // Old 2:1 release check would have rejected this; new implementation accepts it.
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    // First move: dx=12, dy=3 -- horizontal axis locked (dx > dy)
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 403 })
    // Release at dx=64, dy=40: vertical drift at release is now ignored -- navigates
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 440 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('diagonal satisfying abs(dx) >= 2*abs(dy) does navigate', () => {
    // dx=80, dy=10: distance passes; 80 >= 2*10=20 -- dominance passes
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 402 })
    firePointer(container, 'pointerup', { clientX: startX - 80, clientY: 410 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// 8. Axis lock (criterion 5)
// ===========================================================================

describe('axis lock', () => {
  it('vertical axis locked early -- later horizontal travel does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    // dy=15 triggers vertical lock at first move past slop
    firePointer(container, 'pointermove', { clientX: startX - 3, clientY: 415 })
    // Large horizontal sweep cannot rescue gesture once vertical is locked
    firePointer(container, 'pointermove', { clientX: startX - 80, clientY: 418 })
    firePointer(container, 'pointerup', { clientX: startX - 80, clientY: 418 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('horizontal axis locked -- gesture completes and navigates', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    // dx=12, dy=1 -- horizontal lock
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    // dx=80, dy=10 at release: 80 >= 2*10 and 80 >= 64
    firePointer(container, 'pointerup', { clientX: startX - 80, clientY: 410 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// 9. Pointer cancel and state reset (criterion 5)
// ===========================================================================

describe('pointer cancel resets state', () => {
  it('pointercancel mid-gesture -- subsequent pointerup does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    // iOS/Android hijacks the gesture
    firePointer(container, 'pointercancel', { clientX: startX - 12, clientY: 401 })
    // Stale pointerup must not navigate
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('lostpointercapture mid-gesture resets state', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'lostpointercapture', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('second concurrent pointer cancels the active gesture', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    // Pointer 1 starts a qualifying gesture
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, pointerId: 1 })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401, pointerId: 1 })
    // Pointer 2 arrives (pinch-zoom scenario)
    firePointer(container, 'pointerdown', { clientX: 200, clientY: 400, pointerId: 2 })
    // Pointer 1 releases at qualifying distance -- gesture was cancelled by second touch
    firePointer(container, 'pointerup', { clientX: startX - 80, clientY: 402, pointerId: 1 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

// ===========================================================================
// 10. Non-touch and non-primary pointer rejection (criterion 5)
// ===========================================================================

describe('non-touch and non-primary pointer rejection', () => {
  it('mouse pointer does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, pointerType: 'mouse', isPrimary: true })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401, pointerType: 'mouse', isPrimary: true })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402, pointerType: 'mouse', isPrimary: true })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('pen pointer does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, pointerType: 'pen', isPrimary: true })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401, pointerType: 'pen', isPrimary: true })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402, pointerType: 'pen', isPrimary: true })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('non-primary touch does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, pointerType: 'touch', isPrimary: false })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401, pointerType: 'touch', isPrimary: false })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402, pointerType: 'touch', isPrimary: false })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

// ===========================================================================
// 11. Edge guard -- 24 px from either viewport edge (criterion 6)
// ===========================================================================

describe('edge guard', () => {
  const VIEWPORT_WIDTH = 1024

  beforeEach(() => {
    Object.defineProperty(window, 'innerWidth', { value: VIEWPORT_WIDTH, configurable: true, writable: true })
  })

  afterEach(() => {
    Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true, writable: true })
  })

  it('gesture starting at clientX=23 (inside left 24 px zone) does not navigate', () => {
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement, EDGE_GUARD - 1)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('gesture starting at clientX=24 (outside left zone) does navigate', () => {
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement, EDGE_GUARD)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('gesture starting inside right 24 px zone does not navigate', () => {
    // startX = VIEWPORT_WIDTH - 10 = 1014 is inside right zone (>1024-24=1000)
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}/shipment` })
    doRightSwipe(wrapper.element as HTMLElement, VIEWPORT_WIDTH - 10)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('gesture starting outside the right 24 px zone does navigate', () => {
    // startX = VIEWPORT_WIDTH - SWIPE_EDGE_GUARD - 1 = 999, safely inside the valid zone
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}/shipment` })
    doRightSwipe(wrapper.element as HTMLElement, VIEWPORT_WIDTH - EDGE_GUARD - 1)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// 12. Suppression zones -- descendant start rejected (criterion 7)
// ===========================================================================

describe('suppression zones -- interactive descendants', () => {
  const suppressedTags = ['input', 'textarea', 'select'] as const

  for (const tag of suppressedTags) {
    it(`gesture starting on <${tag}> child does not navigate`, () => {
      const wrapper = mountHarness()
      const container = wrapper.element as HTMLElement
      const child = document.createElement(tag)
      container.appendChild(child)

      const startX = 300
      // pointerdown dispatched on child -- bubbles to container listener
      firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: child })
      firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
      firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

      expect(mockPush).not.toHaveBeenCalled()
      container.removeChild(child)
      wrapper.unmount()
    })
  }

  it('gesture starting on <button> child navigates -- button no longer suppresses (R3)', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const child = document.createElement('button')
    container.appendChild(child)

    const startX = 300
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: child })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    container.removeChild(child)
    wrapper.unmount()
  })

  it('gesture starting on <a> child navigates -- anchor no longer suppresses (R3)', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const child = document.createElement('a')
    container.appendChild(child)

    const startX = 300
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: child })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    container.removeChild(child)
    wrapper.unmount()
  })

  it('tap (no travel) on a child button fires click, does not navigate (R5)', () => {
    // Guards the regression risk: removing button from exclusion list must not break button taps.
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const btn = document.createElement('button')
    const clickSpy = vi.fn()
    btn.addEventListener('click', clickSpy)
    container.appendChild(btn)

    firePointer(container, 'pointerdown', { clientX: 300, clientY: 400, target: btn })
    firePointer(container, 'pointerup', { clientX: 300, clientY: 400, target: btn })
    btn.click()

    expect(mockPush).not.toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalledTimes(1)
    container.removeChild(btn)
    wrapper.unmount()
  })

  it('gesture starting on [role="button"] element navigates -- removed from exclusion list (R3)', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const el = document.createElement('div')
    el.setAttribute('role', 'button')
    container.appendChild(el)

    const startX = 300
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: el })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('gesture starting on [contenteditable="true"] element does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const el = document.createElement('div')
    el.setAttribute('contenteditable', 'true')
    container.appendChild(el)

    const startX = 300
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: el })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('gesture starting on [data-swipe-ignore] element does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const el = document.createElement('div')
    el.setAttribute('data-swipe-ignore', '')
    container.appendChild(el)

    const startX = 300
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: el })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('[data-swipe-ignore] nested inside parent still suppresses (ZoomableSurface/horizontal-scroller convention)', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const outer = document.createElement('div')
    const inner = document.createElement('div')
    inner.setAttribute('data-swipe-ignore', '')
    outer.appendChild(inner)
    container.appendChild(outer)

    const startX = 300
    // Touch starts on inner (the actual touch target), bubbles to container
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: inner })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('gesture starting on a plain div child (non-interactive) does navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const plain = document.createElement('div')
    container.appendChild(plain)

    const startX = 300
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400, target: plain })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointerup', { clientX: startX - 64, clientY: 402 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// 13. Enabled gate / modal suppression (criterion 8)
// ===========================================================================

describe('enabled gate -- modal suppression', () => {
  it('enabled=false suppresses navigation', () => {
    const enabled = ref(false)
    const wrapper = mountHarness({ enabled })
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('enabled=true allows navigation', () => {
    const enabled = ref(true)
    const wrapper = mountHarness({ enabled })
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('navigation recovers after toggling enabled from false to true between gestures', () => {
    const enabled = ref(false)
    const wrapper = mountHarness({ enabled })
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()

    enabled.value = true
    mockPush.mockClear()

    doLeftSwipe(container)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('enabled gate evaluates at release: disabling mid-gesture blocks navigation', async () => {
    // The enabled gate is checked when the gesture is committed (pointerup).
    // A modal that opens after gesture start but before release must still block navigation.
    const enabled = ref(true)
    const wrapper = mountHarness({ enabled })
    const container = wrapper.element as HTMLElement

    firePointer(container, 'pointerdown', { clientX: 300, clientY: 400 })
    firePointer(container, 'pointermove', { clientX: 300 - (AXIS_SLOP + 2), clientY: 401 })

    enabled.value = false
    await nextTick()

    firePointer(container, 'pointerup', { clientX: 300 - SWIPE_THRESHOLD, clientY: 402 })
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

// ===========================================================================
// 14. PWA gate -- isPwa false registers no listeners (criterion 9)
// ===========================================================================

describe('PWA gate -- isPwa false', () => {
  beforeEach(() => {
    mockIsPwa.value = false
  })

  afterEach(() => {
    mockIsPwa.value = true
  })

  it('does not navigate on left or right swipe when isPwa is false', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    doLeftSwipe(container)
    doRightSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('no listeners registered on the container when isPwa is false (behavioral)', () => {
    // Behavioral proxy: when isPwa is false the composable is a no-op.
    // A qualifying swipe on the container does not navigate.
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    doLeftSwipe(container)
    doRightSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

// ===========================================================================
// 15. Query and hash forwarding (criterion 12)
// ===========================================================================

describe('query and hash forwarding', () => {
  it('forwards route.query verbatim on advance', () => {
    const wrapper = mountHarness({
      routePath: `/coin/${COIN_ID}`,
      query: { view: 'timeline', page: '2' },
    })
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).toHaveBeenCalledWith(
      expect.objectContaining({ query: { view: 'timeline', page: '2' } }),
    )
    wrapper.unmount()
  })

  it('forwards route.hash verbatim on advance', () => {
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}`, hash: '#catalog-references' })
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).toHaveBeenCalledWith(
      expect.objectContaining({ hash: '#catalog-references' }),
    )
    wrapper.unmount()
  })

  it('passes empty query and hash when the current route has none', () => {
    const wrapper = mountHarness({ routePath: `/coin/${COIN_ID}`, query: {}, hash: '' })
    doLeftSwipe(wrapper.element as HTMLElement)
    const pushed = mockPush.mock.calls[0]?.[0] as Record<string, unknown>
    expect(pushed.query).toEqual({})
    expect(pushed.hash).toBe('')
    wrapper.unmount()
  })

  it('uses router.push (not replace) so the back-button undoes each swipe individually', () => {
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement)
    // push was called; if replace were used a separate mockReplace spy would be called
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// 17. Passive listeners and no scroll interference (criterion 16)
// ===========================================================================

describe('passive listeners -- no scroll interference', () => {
  it('all pointer event listeners are registered with { passive: true }', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    // Every addEventListener call in the primitive must carry { passive: true }
    // so that pointer events never block the browser's native scroll path.
    const addCount = (source.match(/\.addEventListener\(/g) ?? []).length
    const passiveCount = (source.match(/passive:\s*true/g) ?? []).length
    expect(addCount).toBeGreaterThan(0)
    expect(passiveCount).toBe(addCount)
  })

  it('composable source contains no preventDefault call', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    expect(source).not.toContain('preventDefault')
  })
})

// ===========================================================================
// 18. Unmount cleanup (criterion 10)
// ===========================================================================

describe('unmount cleanup', () => {
  it('after unmount a qualifying swipe on the detached container triggers no navigation', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    wrapper.unmount()
    doLeftSwipe(container)

    expect(mockPush).not.toHaveBeenCalled()
  })

  it('remounting adds exactly one listener set -- no duplicate navigation after remount', () => {
    const wrapper1 = mountHarness()
    wrapper1.unmount()

    const wrapper2 = mountHarness()
    const container2 = wrapper2.element as HTMLElement

    doLeftSwipe(container2)

    // Must be called exactly once, not twice (which would indicate a stale listener from wrapper1)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper2.unmount()
  })
})

// ===========================================================================
// 19. Source invariants (criteria 15 and 16)
// ===========================================================================

describe('source invariants', () => {
  it('composable source does not add aria attributes or manipulate focus', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    expect(source).not.toContain('setAttribute')
    expect(source).not.toContain('aria-')
    expect(source).not.toContain('tabindex')
    expect(source).not.toContain('tabIndex')
    expect(source).not.toContain('focus(')
  })

  it('composable source does not attach listeners to window or document (container-scoped only)', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    expect(source).not.toMatch(/window\.addEventListener/)
    expect(source).not.toMatch(/document\.addEventListener/)
  })

  it('composable source has no module-level mutable state that would persist across instances', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    // Match let/var starting at column 0 only — indented declarations are inside function bodies
    // and are per-instance state, not shared module-level mutable state.
    const bareLetOrVar = source.match(/^(?!\/\/)(?:let|var)\s+/m)
    expect(bareLetOrVar).toBeNull()
  })
})

// ===========================================================================
// 20. Component integration -- both call sites wired (criterion 14)
// ===========================================================================

describe('component integration -- call sites', () => {
  it('CoinDetailPage.vue imports and calls useCoinDetailSwipeNav', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'pages', 'CoinDetailPage.vue'),
      'utf-8',
    )
    expect(source).toContain('useCoinDetailSwipeNav')
  })

  it('CoinDetailSectionPageShell.vue imports and calls useCoinDetailSwipeNav', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'components', 'coin', 'CoinDetailSectionPageShell.vue'),
      'utf-8',
    )
    expect(source).toContain('useCoinDetailSwipeNav')
  })

  it('CoinDetailPage.vue passes a containerRef to scope the listener to the page root', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'pages', 'CoinDetailPage.vue'),
      'utf-8',
    )
    expect(source).toContain('containerRef')
  })

  it('each call site references the composable a bounded number of times (no accidental duplication)', () => {
    // import line + one call site = 2 occurrences; 3 is acceptable for a type import as well
    const overviewSource = readFileSync(
      join(process.cwd(), 'src', 'pages', 'CoinDetailPage.vue'),
      'utf-8',
    )
    const shellSource = readFileSync(
      join(process.cwd(), 'src', 'components', 'coin', 'CoinDetailSectionPageShell.vue'),
      'utf-8',
    )
    const overviewCount = (overviewSource.match(/useCoinDetailSwipeNav/g) ?? []).length
    const shellCount = (shellSource.match(/useCoinDetailSwipeNav/g) ?? []).length

    expect(overviewCount).toBeGreaterThanOrEqual(1)
    expect(overviewCount).toBeLessThanOrEqual(3)
    expect(shellCount).toBeGreaterThanOrEqual(1)
    expect(shellCount).toBeLessThanOrEqual(3)
  })
})

// ===========================================================================
// 21. Accessibility -- overflow menu structure unchanged (criterion 15)
// ===========================================================================

describe('overflow menu -- accessibility unchanged', () => {
  it('CoinDetailOverflowMenu renders section links via SECTION_ORDER (not a hardcoded list)', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'components', 'coin', 'CoinDetailOverflowMenu.vue'),
      'utf-8',
    )
    expect(source).toContain('SECTION_ORDER')
  })

  it('CoinDetailOverflowMenu still contains Sell Coin and Copy Coin actions', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'components', 'coin', 'CoinDetailOverflowMenu.vue'),
      'utf-8',
    )
    expect(source).toContain('Sell Coin')
    expect(source).toContain('Copy Coin')
  })

  it('useCoinDetailSwipeNav adds no aria-hidden, no focus trap, no tabindex', () => {
    const source = readFileSync(
      join(process.cwd(), 'src', 'composables', 'useSwipeGesture.ts'),
      'utf-8',
    )
    expect(source).not.toContain('aria-hidden')
    expect(source).not.toContain('tabIndex')
    expect(source).not.toContain('tabindex')
  })
})

// ===========================================================================
// 22. Preference gate -- account-wide pwaSwipeNavEnabled setting
//     Design authority: .squad/decisions/inbox/maximus-pwa-swipe-setting-review.md
// ===========================================================================

describe('preference gate -- pwaSwipeNavEnabled', () => {
  it('default disabled: pwaSwipeNavEnabled=false attaches no listeners and no gesture navigates', () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('pwaSwipeNavEnabled=true enables navigation in installed PWA', () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('fail closed: user object with pwaSwipeNavEnabled absent (undefined) behaves as disabled', () => {
    authStore.proxy.user = {}
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('fail closed: auth.user=null (logged out) attaches no listeners', () => {
    authStore.proxy.user = null
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('browser inactive even with pwaSwipeNavEnabled=true: isPwa=false, no navigation', () => {
    mockIsPwa.value = false
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('live enable: flipping pwaSwipeNavEnabled true on a mounted component attaches listeners', async () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()

    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    await nextTick()

    doLeftSwipe(container)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('live disable: flipping pwaSwipeNavEnabled false detaches listeners and navigation stops', async () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)
    expect(mockPush).toHaveBeenCalledTimes(1)
    mockPush.mockClear()

    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    await nextTick()

    doLeftSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('no duplicate listeners after repeated on/off/on cycles', async () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    await nextTick()
    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    await nextTick()
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    await nextTick()

    doLeftSwipe(container)
    // Must be called exactly once -- duplicate listeners would give 2 or more
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('logout detaches listeners: auth.user null after enable stops navigation', async () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)
    expect(mockPush).toHaveBeenCalledTimes(1)
    mockPush.mockClear()

    authStore.proxy.user = null
    await nextTick()

    doLeftSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('account switch: replacing user with pwaSwipeNavEnabled=false detaches listeners', async () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)
    expect(mockPush).toHaveBeenCalledTimes(1)
    mockPush.mockClear()

    // Account switch: new user has the setting off
    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    await nextTick()

    doLeftSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('account switch: replacing user with pwaSwipeNavEnabled=true attaches listeners', async () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()

    // Account switch: new user has the setting on
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    await nextTick()

    doLeftSwipe(container)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('preference gate and options.enabled gate are independent concerns', async () => {
    // options.enabled is the modal-suppression gate; pwaSwipeNavEnabled is the account preference.
    // Both must pass for navigation to occur.
    const enabled = ref(true)
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness({ enabled })
    const container = wrapper.element as HTMLElement

    // Both gates open: navigation works
    doLeftSwipe(container)
    expect(mockPush).toHaveBeenCalledTimes(1)
    mockPush.mockClear()

    // Modal gate closes: navigation suppressed even with preference on
    enabled.value = false
    await nextTick()
    doLeftSwipe(container)
    expect(mockPush).not.toHaveBeenCalled()

    wrapper.unmount()
  })
})


// ===========================================================================
// R1-R9. Regression matrix -- Phase 1 corrections
// ===========================================================================

describe('R1: touch-action applied and restored', () => {
  it('container touch-action is pan-y while feature is attached', () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    const el = wrapper.element as HTMLElement
    expect(el.style.touchAction).toBe('pan-y')
    wrapper.unmount()
  })

  it('touch-action is restored after unmount', () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    const el = wrapper.element as HTMLElement
    wrapper.unmount()
    // restored to the prior value (empty string -- no prior inline value was set)
    expect(el.style.touchAction).not.toBe('pan-y')
  })

  it('touch-action is restored when preference is turned off', async () => {
    authStore.proxy.user = { pwaSwipeNavEnabled: true }
    const wrapper = mountHarness()
    const el = wrapper.element as HTMLElement
    expect(el.style.touchAction).toBe('pan-y')

    authStore.proxy.user = { pwaSwipeNavEnabled: false }
    await nextTick()

    expect(el.style.touchAction).not.toBe('pan-y')
    wrapper.unmount()
  })
})

describe('R2: arced swipe navigates (no release-time 2:1 dominance gate)', () => {
  it('dx=-80 dy=+50 navigates (axis locked horizontal at slop, vertical drift at release ignored)', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    // horizontal axis locked (dx=12 > dy=3)
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 403 })
    // release with large vertical drift -- no 2:1 gate, navigates
    firePointer(container, 'pointerup', { clientX: startX - 80, clientY: 450 })

    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

describe('R6: stale document text selection does not block a qualifying swipe', () => {
  it('non-empty window.getSelection at release does not discard the gesture', () => {
    vi.spyOn(window, 'getSelection').mockReturnValue({ toString: () => 'selected text' } as Selection)
    const wrapper = mountHarness()
    doLeftSwipe(wrapper.element as HTMLElement)
    expect(mockPush).toHaveBeenCalledTimes(1)
    vi.restoreAllMocks()
    wrapper.unmount()
  })
})

describe('R7: fresh gesture after pointercancel navigates -- no stuck state', () => {
  it('first gesture cancelled, second qualifying gesture navigates', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    // First gesture: cancelled
    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    firePointer(container, 'pointermove', { clientX: startX - 12, clientY: 401 })
    firePointer(container, 'pointercancel', { clientX: startX - 12, clientY: 401 })

    // Second gesture: clean and qualifying
    doLeftSwipe(container, startX)
    expect(mockPush).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})

describe('R8: vertical-dominant drag still rejects (axis lock preserved)', () => {
  it('vertical-dominant drag at slop does not navigate', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement
    const startX = 300

    firePointer(container, 'pointerdown', { clientX: startX, clientY: 400 })
    // dy=15 > dx=3 at slop -- vertical axis locked, gesture abandoned
    firePointer(container, 'pointermove', { clientX: startX - 3, clientY: 415 })
    firePointer(container, 'pointermove', { clientX: startX - 80, clientY: 420 })
    firePointer(container, 'pointerup', { clientX: startX - 80, clientY: 420 })

    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

describe('R9: gesture within edge guard ignored', () => {
  it('gesture starting within 24 px of left viewport edge is ignored', () => {
    const wrapper = mountHarness()
    const container = wrapper.element as HTMLElement

    doLeftSwipe(container, EDGE_GUARD - 1) // startX = 23, inside guard
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('gesture starting within 24 px of right viewport edge is ignored', () => {
    const wrapper = mountHarness({ routePath: '/coin/42/shipment' })
    const container = wrapper.element as HTMLElement

    doRightSwipe(container, window.innerWidth - 10) // inside right guard
    expect(mockPush).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
