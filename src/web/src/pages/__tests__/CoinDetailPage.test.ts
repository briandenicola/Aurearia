import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import CoinDetailPage from '../CoinDetailPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const coin = buildRomanDenariusCore()
const fetchCoin = vi.fn()
const routerPush = vi.fn()
const shareCoinCard = vi.fn()
const sharing = ref(false)

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => ({
    loading: false,
    currentCoin: coin,
    fetchCoin,
  }),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: String(coin.id) } }),
  useRouter: () => ({ push: routerPush }),
}))

vi.mock('@/api/client', () => ({
  deleteCoin: vi.fn(),
  duplicateCoin: vi.fn(),
  purchaseCoin: vi.fn(),
  sellCoin: vi.fn(),
}))

import { duplicateCoin } from '@/api/client'

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showConfirm: vi.fn(),
    showAlert: vi.fn(),
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
    vi.mocked(duplicateCoin).mockResolvedValue({ data: { ...coin, id: 314 } })
    sharing.value = false
  })

  it('keeps the beta/main two-image hero display and shows the share action', () => {
    const wrapper = mount(CoinDetailPage, {
      global: {
        stubs: pageStubs(),
      },
    })

    expect(wrapper.find('.hero-media-grid').exists()).toBe(true)
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

    await wrapper.find('button[aria-label="Duplicate"]').trigger('click')
    await flushPromises()

    expect(duplicateCoin).toHaveBeenCalledWith(coin.id)
    expect(routerPush).toHaveBeenCalledWith('/coin/314')
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
    CoinDetailSectionLinks: true,
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
