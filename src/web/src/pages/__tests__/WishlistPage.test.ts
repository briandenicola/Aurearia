import { beforeEach, describe, expect, it, vi } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import type { Coin } from '@/types'
import WishlistPage from '../WishlistPage.vue'

const mockStore = {
  loading: false,
  coins: [] as Coin[],
  total: 0,
  fetchCoins: vi.fn(),
}
let mockIsPwa = false

vi.mock('@/stores/coins', () => ({
  useCoinsStore: () => mockStore,
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({
    isPwa: mockIsPwa,
  }),
}))

vi.mock('@/api/client', () => ({
  purchaseCoin: vi.fn(),
  checkWishlistAvailability: vi.fn(),
  updateListingStatus: vi.fn(),
}))

function createCoin(id: number): Coin {
  return {
    id,
    name: `Coin ${id}`,
    category: 'Roman',
    denomination: 'Denarius',
    ruler: 'Philip I',
    era: 'Roman Empire',
    mint: 'Rome',
    material: 'Silver',
    weightGrams: null,
    diameterMm: null,
    grade: '',
    obverseInscription: '',
    reverseInscription: '',
    obverseDescription: '',
    reverseDescription: '',
    rarityRating: '',
    purchasePrice: 800,
    currentValue: null,
    purchaseDate: null,
    purchaseLocation: '',
    storageLocationId: null,
    storageLocation: null,
    notes: '',
    aiAnalysis: '',
    obverseAnalysis: '',
    reverseAnalysis: '',
    referenceUrl: '',
    referenceText: '',
    isWishlist: true,
    isSold: false,
    soldPrice: null,
    soldDate: null,
    soldTo: '',
    isPrivate: false,
    listingStatus: 'available',
    listingCheckedAt: null,
    listingCheckReason: '',
    userId: 1,
    images: [],
    createdAt: '',
    updatedAt: '',
  }
}

describe('WishlistPage', () => {
  beforeEach(() => {
    mockStore.loading = false
    mockStore.coins = []
    mockStore.total = 0
    mockStore.fetchCoins.mockReset()
    mockIsPwa = false
  })

  const routerLinkStub = {
    props: ['to'],
    template: '<a :href="to" :title="$attrs.title"><slot /></a>',
  }

  it('does not show the empty state when wishlist coins are present on a single page', () => {
    mockStore.coins = [createCoin(1)]
    mockStore.total = 1

    const wrapper = shallowMount(WishlistPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
        },
      },
    })

    expect(mockStore.fetchCoins).toHaveBeenCalledWith({ wishlist: 'true', sort: 'updated_at', order: 'desc', page: 1 })
    expect(wrapper.find('.coins-grid').exists()).toBe(true)
    expect(wrapper.find('.empty-state').exists()).toBe(false)
    expect(wrapper.find('.pagination').exists()).toBe(false)
  })

  it('continues to fetch only wishlist coins and never quick-capture drafts', () => {
    shallowMount(WishlistPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
        },
      },
    })

    expect(mockStore.fetchCoins).toHaveBeenCalledWith({ wishlist: 'true', sort: 'updated_at', order: 'desc', page: 1 })
    expect(mockStore.fetchCoins).not.toHaveBeenCalledWith(expect.objectContaining({ sold: 'true' }))
  })

  it('shows the empty state when no wishlist coins are present', () => {
    const wrapper = shallowMount(WishlistPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
        },
      },
    })

    expect(wrapper.find('.coins-grid').exists()).toBe(false)
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    const finderLink = wrapper.find('a[title="Add Wish List Finder Agent"]')
    expect(finderLink.exists()).toBe(true)
    expect(finderLink.attributes('href')).toBe('/wishlist/search-alerts')
    expect(finderLink.text()).toContain('Add Wish List Finder Agent')
  })

  it('routes the desktop add action to the Identify Coin workflow', () => {
    const wrapper = shallowMount(WishlistPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
        },
      },
    })

    const links = wrapper.findAll('a')
    expect(links.filter(link => link.attributes('href') === '/lookup')).toHaveLength(1)
    expect(links.some(link => link.attributes('href') === '/lookup' && link.text().includes('Identify Coin'))).toBe(true)
    expect(links.some(link => link.attributes('href') === '/add?wishlist=true')).toBe(false)
  })

  it('shows the desktop search alerts action with text when wishlist coins are present', () => {
    mockStore.coins = [createCoin(1)]
    mockStore.total = 1

    const wrapper = shallowMount(WishlistPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
        },
      },
    })

    const finderLink = wrapper.find('.header-actions a[href="/wishlist/search-alerts"]')
    expect(finderLink.exists()).toBe(true)
    expect(finderLink.attributes('href')).toBe('/wishlist/search-alerts')
    expect(finderLink.text()).toContain('Search Alerts')
  })

  it('routes the PWA plus icon to the Identify Coin workflow', () => {
    mockIsPwa = true

    const wrapper = shallowMount(WishlistPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
        },
      },
    })

    const lookupLink = wrapper.find('a[title="Identify Coin"]')
    expect(lookupLink.exists()).toBe(true)
    expect(lookupLink.attributes('href')).toBe('/lookup')
    expect(wrapper.find('a[href="/add?wishlist=true"]').exists()).toBe(false)
  })

  it('shows the finder agent icon in PWA mode when wishlist coins are present', () => {
    mockIsPwa = true
    mockStore.coins = [createCoin(1)]
    mockStore.total = 1

    const wrapper = shallowMount(WishlistPage, {
      global: {
        stubs: {
          RouterLink: routerLinkStub,
        },
      },
    })

    const finderLink = wrapper.find('.pwa-actions a[href="/wishlist/search-alerts"]')
    expect(finderLink.exists()).toBe(true)
    expect(finderLink.attributes('href')).toBe('/wishlist/search-alerts')
    expect(finderLink.text()).not.toContain('Search Alerts')
  })
})
