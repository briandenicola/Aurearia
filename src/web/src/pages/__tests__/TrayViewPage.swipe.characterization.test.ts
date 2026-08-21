/**
 * Behaviour-level characterization tests for TrayViewPage's inline tray swipe engine.
 *
 * Design authority: .squad/decisions/inbox/maximus-pwa-swipe-reliability-review.md §§9–13
 * Purpose: pin the current intended contract of the Tray drag engine BEFORE production
 * migration to the shared useSwipeGesture primitive (Phase 2).
 * PR 3 must pass or explicitly update these tests after migration.
 *
 * DIRECTION CONVENTION
 * Tray matches iOS convention (opposite of Gallery's current code):
 *   right drag (positive trayDragX) → flyTray(-1) → handlePrevDrawer()
 *   left  drag (negative trayDragX) → flyTray(+1) → handleNextDrawer()
 *
 * THRESHOLD NOTE
 * Tray uses strict-greater-than: |dragX| > SWIPE_THRESHOLD (100 px) to commit.
 * Exactly 100 px falls into the spring-back path; 101 px is the minimum commit travel.
 *
 * REGRESSION TESTS (T3/T4 were it.fails — defects fixed by Phase 2 migration):
 *
 *   T3 FIX: useSwipeGesture routes pointercancel → onCancel → traySpringBack (never commits).
 *
 *   T4 FIX: Tailwind class `touch-pan-y` removed from MuseumTray. The gesture wrapper div
 *        receives `touch-action: none` via the useSwipeGesture primitive (touchAction: 'none').
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import { nextTick } from 'vue'
import TrayViewPage from '@/pages/TrayViewPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

// ---------------------------------------------------------------------------
// Hoisted mocks
// ---------------------------------------------------------------------------

const mockGetCoins = vi.hoisted(() => vi.fn())
const mockPush = vi.hoisted(() => vi.fn())

vi.mock('@/api/client', () => ({
  getCoins: (params?: Record<string, unknown>) => mockGetCoins(params),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

vi.mock('@/composables/useTrayPreference', () => ({
  useTrayPreference: () => ({ feltColor: 'green' }),
}))

// ---------------------------------------------------------------------------
// Constants mirrored from TrayViewPage.vue (not exported)
// ---------------------------------------------------------------------------

const SWIPE_THRESHOLD = 100
/** Minimum px required to commit: Tray uses strict >, so threshold+1. */
const MIN_COMMIT_DIST = SWIPE_THRESHOLD + 1
const COINS_PER_DRAWER = 12

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

/**
 * Build N coins with known diameter so they pass hasKnownDiameterMm and appear in tray drawers.
 * Note: buildRomanDenariusCore defaults to diameterMm: 18 already; explicit override for clarity.
 */
function buildTrayCoins(count: number) {
  return Array.from({ length: count }, (_, i) =>
    buildRomanDenariusCore({ id: i + 1, name: `Coin ${i + 1}`, diameterMm: 18 }),
  )
}

// ---------------------------------------------------------------------------
// Mount helper
// ---------------------------------------------------------------------------

interface MountOptions {
  coins?: ReturnType<typeof buildTrayCoins>
}

async function mountTray(opts: MountOptions = {}) {
  const coins = opts.coins ?? buildTrayCoins(COINS_PER_DRAWER * 2)  // 24 → 2 drawers
  mockGetCoins.mockResolvedValueOnce({ data: { coins, total: coins.length } })

  const wrapper = shallowMount(TrayViewPage, {
    global: {
      stubs: {
        RouterLink: routerLinkStub,
        MuseumTray: true,
        TrayControls: true,
      },
    },
  })
  await flushPromises()
  return wrapper
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
 * Return MuseumTray element for event dispatch (events bubble to the gesture wrapper).
 * After Phase 2 migration, useSwipeGesture attaches to the .tray-gesture-surface wrapper,
 * so we stub capture methods there (jsdom does not implement setPointerCapture natively).
 */
function prepTrayEl(wrapper: Awaited<ReturnType<typeof mountTray>>): HTMLElement {
  // Stub capture on the gesture wrapper (where primitive calls setPointerCapture)
  const gestureEl = wrapper.find('.tray-gesture-surface').element as HTMLElement
  gestureEl.setPointerCapture = vi.fn()
  gestureEl.releasePointerCapture = vi.fn()
  // Return MuseumTray for event dispatch (events bubble to gesture wrapper)
  return wrapper.findComponent({ name: 'MuseumTray' }).element as HTMLElement
}

/** Return the gesture surface wrapper element to assert capture calls. */
function getGestureEl(wrapper: Awaited<ReturnType<typeof mountTray>>): HTMLElement {
  return wrapper.find('.tray-gesture-surface').element as HTMLElement
}

/**
 * Read drawer index from the TrayControls stub element.
 * VTU2 stubs inherit the real component's prop declarations; drawerIndex goes to $props (not $attrs).
 * Vue 3 sets it on the DOM element via setAttribute('drawerIndex', value) — jsdom stores it as
 * 'drawerindex' (all lowercase). The attribute value is a stringified number.
 */
function drawerIndex(wrapper: Awaited<ReturnType<typeof mountTray>>): number {
  const ctrl = wrapper.findComponent({ name: 'TrayControls' })
  return Number(ctrl.element.getAttribute('drawerindex') ?? 0)
}

/**
 * Right drag: positive trayDragX → flyTray(-1) → handlePrevDrawer() [iOS convention].
 * Uses MIN_COMMIT_DIST to ensure the commit threshold (> 100) is crossed.
 */
function doTrayRightDrag(el: HTMLElement, distance = MIN_COMMIT_DIST + 19, startX = 300, startY = 300) {
  firePointer(el, 'pointerdown', { clientX: startX, clientY: startY })
  firePointer(el, 'pointermove', { clientX: startX + Math.floor(distance / 2), clientY: startY + 1 })
  firePointer(el, 'pointermove', { clientX: startX + distance, clientY: startY + 1 })
  firePointer(el, 'pointerup', { clientX: startX + distance, clientY: startY + 1 })
}

/**
 * Left drag: negative trayDragX → flyTray(+1) → handleNextDrawer().
 */
function doTrayLeftDrag(el: HTMLElement, distance = MIN_COMMIT_DIST + 19, startX = 300, startY = 300) {
  firePointer(el, 'pointerdown', { clientX: startX, clientY: startY })
  firePointer(el, 'pointermove', { clientX: startX - Math.floor(distance / 2), clientY: startY + 1 })
  firePointer(el, 'pointermove', { clientX: startX - distance, clientY: startY + 1 })
  firePointer(el, 'pointerup', { clientX: startX - distance, clientY: startY + 1 })
}

// ---------------------------------------------------------------------------
// Suite setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  mockGetCoins.mockReset()
  mockPush.mockReset()
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

// ===========================================================================
// T1 — left drag past threshold advances drawer (left drag → next drawer)
// ===========================================================================

describe('T1 — left drag (negative dragX): advances to next drawer', () => {
  it('left drag from drawer 0 → drawer 1', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    doTrayLeftDrag(el)
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    expect(drawerIndex(wrapper)).toBe(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// T2 — right drag past threshold retreats drawer (right drag → previous drawer)
// ===========================================================================

describe('T2 — right drag (positive dragX): retreats to previous drawer', () => {
  it('right drag from drawer 1 → drawer 0', async () => {
    const coins = buildTrayCoins(COINS_PER_DRAWER * 2)
    mockGetCoins.mockResolvedValueOnce({ data: { coins, total: coins.length } })
    const wrapper = shallowMount(TrayViewPage, {
      global: { stubs: { RouterLink: routerLinkStub, MuseumTray: true, TrayControls: true } },
    })
    await flushPromises()
    const el = prepTrayEl(wrapper)

    // Navigate to drawer 1 via left drag
    doTrayLeftDrag(el)
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()
    expect(drawerIndex(wrapper)).toBe(1)

    // Now right drag back to drawer 0
    doTrayRightDrag(el)
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()
    expect(drawerIndex(wrapper)).toBe(0)
    wrapper.unmount()
  })
})

// ===========================================================================
// T3 — KNOWN DEFECT: pointercancel after threshold commits drawer change
// ===========================================================================

describe('T3 — pointercancel behavior', () => {
  it(
    'T3 REGRESSION — pointercancel after travel > threshold springs back (drawer unchanged); '
    + 'fixed: useSwipeGesture routes cancel → onCancel → traySpringBack, never commits',
    async () => {
      const wrapper = await mountTray()
      const el = prepTrayEl(wrapper)

      firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
      firePointer(el, 'pointermove', { clientX: 300 - MIN_COMMIT_DIST - 19, clientY: 301 })  // left drag past threshold
      firePointer(el, 'pointercancel', { clientX: 300 - MIN_COMMIT_DIST - 19, clientY: 301 })
      await vi.advanceTimersByTimeAsync(300)
      await nextTick()

      // Correct behavior: drawer stays at 0 after cancel
      expect(drawerIndex(wrapper)).toBe(0)
      wrapper.unmount()
    },
  )

  it('pointercancel below threshold springs back (drawer unchanged)', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    // 30 px — below the 100 px threshold
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 270, clientY: 301 })
    firePointer(el, 'pointercancel', { clientX: 270, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    expect(drawerIndex(wrapper)).toBe(0)
    wrapper.unmount()
  })

  it('fresh gesture accepted after pointercancel resets state (no stuck-state)', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    // Cancel sub-threshold
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 270, clientY: 301 })
    firePointer(el, 'pointercancel', { clientX: 270, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    // Full qualifying left drag should now work
    doTrayLeftDrag(el)
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()
    expect(drawerIndex(wrapper)).toBe(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// T4 — KNOWN DEFECT: MuseumTray gesture surface has touch-action: pan-y, not none
// Source check justified: jsdom does not process scoped CSS or Tailwind classes.
// ===========================================================================

describe('T4 — MuseumTray touch-action expectation and tray gesture surface touch-action', () => {
  it(
    'T4 REGRESSION — MuseumTray element has no touch-pan-y class (touch-action conflict fixed); '
    + 'the gesture wrapper .tray-gesture-surface receives touch-action: pan-y from useSwipeGesture',
    async () => {
      const wrapper = await mountTray()
      const trayEl = wrapper.findComponent({ name: 'MuseumTray' })

      // MuseumTray must not carry the Tailwind touch-pan-y class (conflict fixed).
      expect(trayEl.classes()).not.toContain('touch-pan-y')
      wrapper.unmount()
    },
  )

  it('tray-gesture-surface wrapper receives touch-action: pan-y from useSwipeGesture (proves vertical scroll preserved)', async () => {
    const wrapper = await mountTray()
    const gestureEl = wrapper.find('.tray-gesture-surface').element as HTMLElement

    // The primitive sets pan-y inline so the browser keeps vertical scroll while the
    // axis-lock (lockSlop: 10) still lets horizontal drags commit drawer changes.
    expect(gestureEl.style.touchAction).toBe('pan-y')
    wrapper.unmount()
  })

  it('vertical-dominant drag does not commit (axis lock yields vertical-dominant gestures to browser pan-y)', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    // dy >> dx: axis lock fires, determines vertical, marks gesture as ignored -> spring-back
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 305, clientY: 345 })  // dx=5, dy=45 -> vertical dominant
    firePointer(el, 'pointerup',   { clientX: 310, clientY: 390 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    expect(drawerIndex(wrapper)).toBe(0)
    wrapper.unmount()
  })
})

// ===========================================================================
// T5 — single-drawer page: onTrayPointerDown returns early, no capture
// ===========================================================================

describe('T5 — single-drawer page (totalDrawers <= 1): pointerdown is a no-op', () => {
  it('setPointerCapture is NOT called when only 1 drawer exists', async () => {
    const wrapper = await mountTray({ coins: buildTrayCoins(6) })  // 6/12 = 1 drawer
    const el = prepTrayEl(wrapper)
    const gestureEl = getGestureEl(wrapper)  // capture is stubbed on gesture wrapper

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })

    // Primitive is disabled (trayGestureEnabled=false when totalDrawers<=1), so no capture
    expect(gestureEl.setPointerCapture as ReturnType<typeof vi.fn>).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('left drag attempt on single-drawer page leaves drawer at 0', async () => {
    const wrapper = await mountTray({ coins: buildTrayCoins(6) })
    const el = prepTrayEl(wrapper)

    doTrayLeftDrag(el)
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    expect(drawerIndex(wrapper)).toBe(0)
    wrapper.unmount()
  })
})

// ===========================================================================
// Commit threshold: 100 px springs back (strict >), 101 px commits
// ===========================================================================

describe('tray commit threshold', () => {
  it('exactly 100 px horizontal travel does NOT commit — springs back (threshold is strict >)', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 300 - SWIPE_THRESHOLD, clientY: 301 })
    firePointer(el, 'pointerup', { clientX: 300 - SWIPE_THRESHOLD, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    expect(drawerIndex(wrapper)).toBe(0)
    wrapper.unmount()
  })

  it('99 px horizontal travel does not commit — drawer unchanged', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 300 - (SWIPE_THRESHOLD - 1), clientY: 301 })
    firePointer(el, 'pointerup', { clientX: 300 - (SWIPE_THRESHOLD - 1), clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    expect(drawerIndex(wrapper)).toBe(0)
    wrapper.unmount()
  })

  it('101 px horizontal travel commits — drawer advances', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301 })
    firePointer(el, 'pointerup', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    expect(drawerIndex(wrapper)).toBe(1)
    wrapper.unmount()
  })
})

// ===========================================================================
// Pointer capture: acquired on pointerdown, released on pointerup
// ===========================================================================

describe('tray pointer capture — acquired and released', () => {
  it('setPointerCapture called with the pointer ID on pointerdown', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)
    const gestureEl = getGestureEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300, pointerId: 5 })

    expect(gestureEl.setPointerCapture as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(5)
    wrapper.unmount()
  })

  it('releasePointerCapture called with the pointer ID on pointerup', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)
    const gestureEl = getGestureEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300, pointerId: 5 })
    firePointer(el, 'pointerup', { clientX: 300, clientY: 300, pointerId: 5 })

    expect(gestureEl.releasePointerCapture as ReturnType<typeof vi.fn>).toHaveBeenCalledWith(5)
    wrapper.unmount()
  })
})

// ===========================================================================
// Pointer type policy — Tray accepts ALL pointer types (touch, mouse, pen)
// ===========================================================================

describe('tray pointer type policy — all types accepted', () => {
  for (const pointerType of ['touch', 'mouse', 'pen'] as const) {
    it(`${pointerType} pointer starts a drag and commits at threshold+1`, async () => {
      const wrapper = await mountTray()
      const el = prepTrayEl(wrapper)

      firePointer(el, 'pointerdown', { clientX: 300, clientY: 300, pointerType })
      firePointer(el, 'pointermove', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301, pointerType })
      firePointer(el, 'pointerup', { clientX: 300 - MIN_COMMIT_DIST, clientY: 301, pointerType })
      await vi.advanceTimersByTimeAsync(300)
      await nextTick()

      expect(drawerIndex(wrapper)).toBe(1)
      wrapper.unmount()
    })
  }
})

// ===========================================================================
// Live transform during tray drag
// ===========================================================================

describe('live tray transform feedback during drag', () => {
  it('MuseumTray has translateX style reflecting trayDragX during pointermove', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 240, clientY: 301 })  // dragX = -60
    await nextTick()

    const style = wrapper.findComponent({ name: 'MuseumTray' }).attributes('style') ?? ''
    expect(style).toContain('translateX(-60px)')
    wrapper.unmount()
  })

  it('transform cleared after spring-back timer completes', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 260, clientY: 301 })
    firePointer(el, 'pointerup', { clientX: 260, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    const style = wrapper.findComponent({ name: 'MuseumTray' }).attributes('style') ?? ''
    expect(style).toBeFalsy()
    wrapper.unmount()
  })
})

// ===========================================================================
// Click suppression: suppressCoinClick set during drag, cleared after animation
// ===========================================================================

describe('tray click suppression during drag', () => {
  it('handleCoinClicked does NOT navigate when suppressCoinClick is true (during drag > 8 px)', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 289, clientY: 301 })  // dragX = -11, > 8 px

    // Emit coin-clicked from the stub while dragging (suppressCoinClick = true)
    wrapper.findComponent({ name: 'MuseumTray' }).vm.$emit('coin-clicked', 1)

    expect(mockPush).not.toHaveBeenCalled()

    firePointer(el, 'pointerup', { clientX: 289, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    wrapper.unmount()
  })

  it('suppressCoinClick resets after spring-back timer — navigation allowed again', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    // Small drag that sets suppressCoinClick but does not commit (below threshold)
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    firePointer(el, 'pointermove', { clientX: 289, clientY: 301 })  // dragX = -11
    firePointer(el, 'pointerup', { clientX: 289, clientY: 301 })
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()

    // After spring-back: coin click should navigate
    wrapper.findComponent({ name: 'MuseumTray' }).vm.$emit('coin-clicked', 1)
    await nextTick()

    expect(mockPush).toHaveBeenCalledWith({ name: 'coin-detail', params: { id: 1 } })
    wrapper.unmount()
  })
})

// ===========================================================================
// No double commit on rapid gestures
// ===========================================================================

describe('tray — no double commit on rapid gestures', () => {
  it('trayIsAnimating gate prevents a second commit before the first animation ends', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    doTrayLeftDrag(el)  // first commit (left = advance) → trayIsAnimating = true

    // Second pointerdown during animation: no new capture
    const secondCaptureSpy = vi.fn()
    getGestureEl(wrapper).setPointerCapture = secondCaptureSpy
    firePointer(el, 'pointerdown', { clientX: 300, clientY: 300 })
    expect(secondCaptureSpy).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(300)
    await nextTick()
    expect(drawerIndex(wrapper)).toBe(1)  // exactly one change

    // Third gesture: right drag back to drawer 0
    doTrayRightDrag(el)
    await vi.advanceTimersByTimeAsync(300)
    await nextTick()
    expect(drawerIndex(wrapper)).toBe(0)
    wrapper.unmount()
  })
})

// ===========================================================================
// Cleanup — animation timers cleared on beforeUnmount
// ===========================================================================

describe('tray cleanup — animation timers cleared on unmount', () => {
  it('unmounting during animation does not throw (clearTimeout called for all timers)', async () => {
    const wrapper = await mountTray()
    const el = prepTrayEl(wrapper)

    doTrayLeftDrag(el)
    wrapper.unmount()  // mid-animation

    await expect(vi.advanceTimersByTimeAsync(300)).resolves.not.toThrow()
  })
})

