/**
 * Mounted call-site integration test for CoinDetailSectionPageShell.vue.
 *
 * Verifies that the ref="containerRef" binding on the root <div> is wired to
 * the useCoinDetailSwipeNav composable. If the binding is removed from the
 * template, capturedRef.value will be null after mount and this test fails.
 *
 * This is the mounted complement to the source-grep tests in
 * useCoinDetailSwipeNav.test.ts §20 and closes the Maximus pre-main gate
 * requirement for mounted call-site assertions.
 */
import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { Ref } from 'vue'
import CoinDetailSectionPageShell from '../CoinDetailSectionPageShell.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const coin = buildRomanDenariusCore()

const swipeCapture = vi.hoisted(() => ({
  ref: undefined as Ref<HTMLElement | null> | undefined,
}))

vi.mock('@/composables/useCoinDetailSwipeNav', () => ({
  SWIPE_THRESHOLD: 64,
  AXIS_SLOP: 10,
  EDGE_GUARD: 24,
  AXIS_DOMINANCE: 2,
  useCoinDetailSwipeNav: (containerRef: Ref<HTMLElement | null>) => {
    swipeCapture.ref = containerRef
  },
}))

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => ({
    loading: false,
    currentCoin: coin,
    fetchCoin: vi.fn(),
  }),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({ params: { id: String(coin.id) } }),
    useRouter: () => ({ push: vi.fn() }),
  }
})

vi.mock('@/api/client', () => ({
  duplicateCoin: vi.fn(),
  sellCoin: vi.fn(),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showConfirm: vi.fn(), showAlert: vi.fn() }),
}))

describe('CoinDetailSectionPageShell -- useCoinDetailSwipeNav containerRef binding', () => {
  it('containerRef is bound to an HTMLElement at mount (fails if ref="containerRef" is removed)', () => {
    swipeCapture.ref = undefined

    const wrapper = mount(CoinDetailSectionPageShell, {
      props: { sectionTitle: 'Activity Journal' },
      global: {
        stubs: {
          CoinDetailOverflowMenu: true,
          AuthenticatedImage: true,
          SellModal: true,
          ChevronLeft: true,
        },
      },
    })

    expect(swipeCapture.ref, 'useCoinDetailSwipeNav was not called -- composable may have been removed from CoinDetailSectionPageShell').toBeDefined()
    expect(
      swipeCapture.ref!.value,
      'containerRef.value is null after mount -- ref="containerRef" may have been removed from the template in CoinDetailSectionPageShell',
    ).toBeInstanceOf(HTMLElement)

    wrapper.unmount()
  })
})
