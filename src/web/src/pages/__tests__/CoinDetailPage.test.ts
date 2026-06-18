import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
import CoinDetailPage from '../CoinDetailPage.vue'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'

const coin = buildRomanDenariusCore()
const fetchCoin = vi.fn()
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
  useRouter: () => ({ push: vi.fn() }),
}))

vi.mock('@/api/client', () => ({
  deleteCoin: vi.fn(),
  purchaseCoin: vi.fn(),
  sellCoin: vi.fn(),
}))

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
    shareCoinCard.mockReset()
    shareCoinCard.mockResolvedValue({ mode: 'downloaded' })
    sharing.value = false
  })

  it('keeps the beta/main two-image hero display and shows the share action', () => {
    const wrapper = mount(CoinDetailPage, {
      global: {
        stubs: pageStubs(),
      },
    })

    expect(wrapper.find('.hero-media-grid').exists()).toBe(true)
    expect(wrapper.text()).toContain('Share')
    expect(fetchCoin).toHaveBeenCalledWith(coin.id)
  })

  it('shares the currently loaded coin when the Share action is clicked', async () => {
    const wrapper = mount(CoinDetailPage, {
      global: {
        stubs: pageStubs(),
      },
    })

    await wrapper.findAll('button').find((button) => button.text().includes('Share'))!.trigger('click')
    await flushPromises()

    expect(shareCoinCard).toHaveBeenCalledWith(coin)
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
    ArrowLeft: true,
    Share2: true,
  }
}
