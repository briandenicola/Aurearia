import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import CoinDetailPage from '../CoinDetailPage.vue'
import router from '@/router'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const coin = buildRomanDenariusCore()
const fetchCoin = vi.fn()
const routerPush = vi.fn()
const shareCoinCard = vi.fn()
const sharing = ref(false)
const quickPinned = ref(false)
const quickBusy = ref(false)
const quickError = ref('')
const quickRefresh = vi.fn()
const quickForget = vi.fn()
const showConfirm = vi.fn()

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => ({
    loading: false,
    currentCoin: coin,
    fetchCoin,
  }),
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({ params: { id: String(coin.id) } }),
    useRouter: () => ({ push: routerPush }),
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

import { deleteCoin, duplicateCoin, purchaseCoin, sellCoin } from '@/api/client'

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showConfirm,
    showAlert: vi.fn(),
  }),
}))

vi.mock('@/composables/useQuickAccess', () => ({
  useQuickAccess: () => ({
    error: quickError,
    refresh: quickRefresh,
    forget: quickForget,
    pin: vi.fn(),
    unpin: vi.fn(),
    isPinned: () => quickPinned,
    isBusy: () => quickBusy,
  }),
}))

vi.mock('@/composables/useCoinShareCard', () => ({
  useCoinShareCard: () => ({
    sharing,
    shareCoinCard,
  }),
}))

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('CoinDetailPage', () => {
  beforeEach(() => {
    fetchCoin.mockReset()
    routerPush.mockReset()
    shareCoinCard.mockReset()
    shareCoinCard.mockResolvedValue({ mode: 'downloaded' })
    vi.mocked(duplicateCoin).mockReset()
    vi.mocked(deleteCoin).mockReset()
    vi.mocked(purchaseCoin).mockReset()
    vi.mocked(sellCoin).mockReset()
    vi.mocked(duplicateCoin).mockResolvedValue({ data: { ...coin, id: 314 } })
    vi.mocked(deleteCoin).mockResolvedValue({ data: {} })
    vi.mocked(purchaseCoin).mockResolvedValue({ data: coin })
    vi.mocked(sellCoin).mockResolvedValue({ data: coin })
    quickRefresh.mockReset()
    quickRefresh.mockResolvedValue(undefined)
    quickForget.mockReset()
    showConfirm.mockReset()
    showConfirm.mockResolvedValue(true)
    coin.isWishlist = false
    coin.isSold = false
    sharing.value = false
  })

  it('keeps the beta/main two-image hero display and shows the share action', () => {
    const wrapper = mount(CoinDetailPage, {
      global: {
        stubs: pageStubs(),
      },
    })

    // Two-column obverse/reverse hero grid — was `.hero-media-grid`, now a plain
    // Tailwind `grid grid-cols-2` container (only the hero media uses grid-cols-2
    // without a responsive prefix; the metadata section below uses `md:grid-cols-2`).
    expect(wrapper.find('.grid-cols-2').exists()).toBe(true)
    expect(wrapper.find('button[aria-label="Share"]').exists()).toBe(true)
    expect(fetchCoin).toHaveBeenCalledWith(coin.id)
  })

  it('shares the currently loaded coin when the Share action is clicked', async () => {
    const wrapper = mount(CoinDetailPage, {
      global: {
        stubs: pageStubs(),
      },
    })

    await wrapper.find('button[aria-label="Share"]').trigger('click')
    await flushPromises()

    expect(shareCoinCard).toHaveBeenCalledWith(coin)
  })

  it('duplicates the loaded coin and navigates to the new detail page', async () => {
    const wrapper = mount(CoinDetailPage, {
      global: {
        stubs: pageStubs(),
      },
    })

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    await wrapper.find('button[aria-label="Copy Coin"]').trigger('click')
    await flushPromises()

    expect(duplicateCoin).toHaveBeenCalledWith(coin.id)
    expect(routerPush).toHaveBeenCalledWith('/coin/314')
  })

  it('anchors Catalog References on the existing coin detail route without a standalone Numista destination', () => {
    const stubs = pageStubs()
    delete stubs.CoinReferencesSection
    const wrapper = mount(CoinDetailPage, {
      global: { stubs },
    })

    const heading = wrapper.findAll('h3').find(item => item.text() === 'Catalog References')
    expect(heading).toBeDefined()
    expect(heading!.element.closest('section')?.id).toBe('catalog-references')

    const numistaRoutes = router.getRoutes().filter(route =>
      /numista/i.test(`${route.path} ${String(route.name ?? '')}`),
    )
    expect(numistaRoutes).toHaveLength(0)
  })

  it('refreshes Quick Access after purchasing a pinned wishlist coin', async () => {
    coin.isWishlist = true
    const stubs = pageStubs()
    stubs.PurchaseModal = {
      template: '<button aria-label="Confirm purchase" @click="$emit(\'confirm\', {})">Confirm</button>',
    }
    const wrapper = mount(CoinDetailPage, { global: { stubs } })
    await wrapper.findAll('button').find((button) => button.text().includes('Mark as Purchased'))!.trigger('click')
    await wrapper.get('button[aria-label="Confirm purchase"]').trigger('click')
    await flushPromises()

    expect(purchaseCoin).toHaveBeenCalledWith(coin.id, {})
    expect(quickRefresh).toHaveBeenCalled()
  })

  it('removes Quick Access state after successful delete and sell workflows', async () => {
    const stubs = pageStubs()
    stubs.SellModal = {
      template: '<button aria-label="Confirm sale" @click="$emit(\'confirm\', 100, \'Buyer\')">Confirm</button>',
    }
    const wrapper = mount(CoinDetailPage, { global: { stubs } })

    await wrapper.get('button[aria-label="Delete"]').trigger('click')
    await flushPromises()
    expect(quickForget).toHaveBeenCalledWith('coin', coin.id)

    await wrapper.find('button[aria-label="Open overflow actions"]').trigger('click')
    await wrapper.get('button[aria-label="Sell Coin"]').trigger('click')
    await wrapper.get('button[aria-label="Confirm sale"]').trigger('click')
    await flushPromises()
    expect(sellCoin).toHaveBeenCalledWith(coin.id, 100, 'Buyer')
    expect(quickForget).toHaveBeenCalledWith('coin', coin.id)
  })
})

function pageStubs() {
  return {
    RouterLink: routerLinkStub,
    SellModal: true,
    PurchaseModal: true,
    ImageLightbox: true,
    CoinTagsSection: true,
    CoinDetailMetadataTable: true,
    CoinListingStatus: true,
    CoinReferencesSection: true,
    AuthenticatedImage: true,
    ArrowLeft: true,
    CircleDollarSign: true,
    Copy: true,
    Pencil: true,
    Share2: true,
    Trash2: true,
  }
}
