/**
 * Mounted call-site integration test for CoinDetailPage.vue.
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
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import type { Ref } from 'vue'
import CoinDetailPage from '../CoinDetailPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const coin = buildRomanDenariusCore()

// Capture the containerRef argument passed to the composable.
// Hoisted so it is available inside the hoisted vi.mock factory.
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
  createCoinReference: vi.fn(),
  deleteCoin: vi.fn(),
  deleteCoinReference: vi.fn(),
  duplicateCoin: vi.fn(),
  listCatalogs: vi.fn().mockResolvedValue([]),
  purchaseCoin: vi.fn(),
  sellCoin: vi.fn(),
  updateCoinReference: vi.fn(),
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({ showConfirm: vi.fn(), showAlert: vi.fn() }),
}))

vi.mock('@/composables/useCoinShareCard', () => ({
  useCoinShareCard: () => ({ sharing: ref(false), shareCoinCard: vi.fn() }),
}))

describe('CoinDetailPage -- useCoinDetailSwipeNav containerRef binding', () => {
  it('containerRef is bound to an HTMLElement at mount (fails if ref="containerRef" is removed)', () => {
    swipeCapture.ref = undefined

    const wrapper = mount(CoinDetailPage, {
      global: {
        stubs: {
          RouterLink: { props: ['to'], template: '<a><slot /></a>' },
          SellModal: true,
          PurchaseModal: true,
          PurchaseReminderModal: true,
          ImageLightbox: true,
          CoinTagsSection: true,
          CoinDetailMetadataTable: true,
          CoinListingStatus: true,
          CoinReferencesSection: true,
          AuthenticatedImage: true,
          ArrowLeft: true,
          CoinDetailHeaderActions: true,
          CircleDollarSign: true,
          Copy: true,
          Pencil: true,
          Share2: true,
          Trash2: true,
        },
      },
    })

    expect(swipeCapture.ref, 'useCoinDetailSwipeNav was not called -- composable may have been removed from CoinDetailPage').toBeDefined()
    expect(
      swipeCapture.ref!.value,
      'containerRef.value is null after mount -- ref="containerRef" may have been removed from the template in CoinDetailPage',
    ).toBeInstanceOf(HTMLElement)

    wrapper.unmount()
  })
})

