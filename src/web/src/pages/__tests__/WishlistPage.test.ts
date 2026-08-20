import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import type { Coin, PurchaseReminder } from '@/types'
import WishlistPage from '../WishlistPage.vue'
import CoinCard from '@/components/CoinCard.vue'

const mockStore = {
  loading: false,
  coins: [] as Coin[],
  total: 0,
  fetchCoins: vi.fn(),
}
let mockIsPwa = false

const mockListPurchaseReminders = vi.fn()

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
  listPurchaseReminders: (...args: unknown[]) => mockListPurchaseReminders(...args),
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

function buildReminder(coinId: number, remindDate = '2026-09-25'): PurchaseReminder {
  return {
    id: 10 + coinId,
    coinId,
    remindDate,
    timezone: 'America/Chicago',
    status: 'pending',
    createdAt: '2026-09-01T10:00:00Z',
    updatedAt: '2026-09-01T10:00:00Z',
  }
}

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to" :title="$attrs.title"><slot /></a>',
}

function mountPage() {
  return shallowMount(WishlistPage, {
    global: {
      stubs: {
        RouterLink: routerLinkStub,
      },
    },
  })
}

describe('WishlistPage', () => {
  beforeEach(() => {
    mockStore.loading = false
    mockStore.coins = []
    mockStore.total = 0
    mockStore.fetchCoins.mockReset()
    mockIsPwa = false
    mockListPurchaseReminders.mockReset()
    mockListPurchaseReminders.mockResolvedValue({ data: { reminders: [] } })
  })

  it('does not show the empty state when wishlist coins are present on a single page', () => {
    mockStore.coins = [createCoin(1)]
    mockStore.total = 1

    const wrapper = mountPage()

    expect(mockStore.fetchCoins).toHaveBeenCalledWith({ wishlist: 'true', sort: 'updated_at', order: 'desc', page: 1 })
    expect(wrapper.find('.coins-grid').exists()).toBe(true)
    expect(wrapper.find('.empty-state').exists()).toBe(false)
    expect(wrapper.find('.pagination').exists()).toBe(false)
  })

  it('continues to fetch only wishlist coins and never quick-capture drafts', () => {
    mountPage()

    expect(mockStore.fetchCoins).toHaveBeenCalledWith({ wishlist: 'true', sort: 'updated_at', order: 'desc', page: 1 })
    expect(mockStore.fetchCoins).not.toHaveBeenCalledWith(expect.objectContaining({ sold: 'true' }))
  })

  it('shows the empty state when no wishlist coins are present', () => {
    const wrapper = mountPage()

    expect(wrapper.find('.coins-grid').exists()).toBe(false)
    expect(wrapper.find('.empty-state').exists()).toBe(true)
    const finderLink = wrapper.find('a[title="Add Wish List Finder Agent"]')
    expect(finderLink.exists()).toBe(true)
    expect(finderLink.attributes('href')).toBe('/wishlist/search-alerts')
    expect(finderLink.text()).toContain('Add Wish List Finder Agent')
  })

  it('routes the desktop add action to the Identify Coin workflow', () => {
    const wrapper = mountPage()

    const links = wrapper.findAll('a')
    expect(links.filter(link => link.attributes('href') === '/lookup')).toHaveLength(1)
    expect(links.some(link => link.attributes('href') === '/lookup' && link.text().includes('Identify Coin'))).toBe(true)
    expect(links.some(link => link.attributes('href') === '/add?wishlist=true')).toBe(false)
  })

  it('shows the desktop search alerts action with text when wishlist coins are present', () => {
    mockStore.coins = [createCoin(1)]
    mockStore.total = 1

    const wrapper = mountPage()

    const finderLink = wrapper.find('.header-actions a[href="/wishlist/search-alerts"]')
    expect(finderLink.exists()).toBe(true)
    expect(finderLink.attributes('href')).toBe('/wishlist/search-alerts')
    expect(finderLink.text()).toContain('Search Alerts')
  })

  it('routes the PWA plus icon to the Identify Coin workflow', () => {
    mockIsPwa = true

    const wrapper = mountPage()

    const lookupLink = wrapper.find('a[title="Identify Coin"]')
    expect(lookupLink.exists()).toBe(true)
    expect(lookupLink.attributes('href')).toBe('/lookup')
    expect(wrapper.find('a[href="/add?wishlist=true"]').exists()).toBe(false)
  })

  it('shows the finder agent icon in PWA mode when wishlist coins are present', () => {
    mockIsPwa = true
    mockStore.coins = [createCoin(1)]
    mockStore.total = 1

    const wrapper = mountPage()

    const finderLink = wrapper.find('.pwa-actions a[href="/wishlist/search-alerts"]')
    expect(finderLink.exists()).toBe(true)
    expect(finderLink.attributes('href')).toBe('/wishlist/search-alerts')
    expect(finderLink.text()).not.toContain('Search Alerts')
  })

  // US5 AC1 — badge propagation
  describe('reminder badge propagation (US5 AC1)', () => {
    it('passes the matching activeReminder to the CoinCard when listPurchaseReminders returns a reminder', async () => {
      const coin = createCoin(42)
      mockStore.coins = [coin]
      mockStore.total = 1
      const reminder = buildReminder(42, '2026-09-25')
      mockListPurchaseReminders.mockResolvedValue({ data: { reminders: [reminder] } })

      const wrapper = mountPage()
      await flushPromises()

      const card = wrapper.findComponent(CoinCard)
      expect(card.props('activeReminder')).toEqual(reminder)
    })

    it('passes null activeReminder when no reminder exists for the coin', async () => {
      const coin = createCoin(7)
      mockStore.coins = [coin]
      mockStore.total = 1
      mockListPurchaseReminders.mockResolvedValue({ data: { reminders: [] } })

      const wrapper = mountPage()
      await flushPromises()

      const card = wrapper.findComponent(CoinCard)
      expect(card.props('activeReminder')).toBeNull()
    })

    it('matches reminders to the correct coin when multiple coins are present', async () => {
      const coins = [createCoin(1), createCoin(2), createCoin(3)]
      mockStore.coins = coins
      mockStore.total = 3
      const reminder2 = buildReminder(2, '2026-10-01')
      mockListPurchaseReminders.mockResolvedValue({ data: { reminders: [reminder2] } })

      const wrapper = mountPage()
      await flushPromises()

      const cards = wrapper.findAllComponents(CoinCard)
      expect(cards[0]?.props('activeReminder')).toBeNull()
      expect(cards[1]?.props('activeReminder')).toEqual(reminder2)
      expect(cards[2]?.props('activeReminder')).toBeNull()
    })

    it('calls listPurchaseReminders on every loadCoins invocation', () => {
      mockStore.coins = [createCoin(1)]
      mountPage()

      expect(mockListPurchaseReminders).toHaveBeenCalledTimes(1)
    })

    it('tolerates a listPurchaseReminders failure without crashing (best-effort)', async () => {
      const coin = createCoin(5)
      mockStore.coins = [coin]
      mockStore.total = 1
      mockListPurchaseReminders.mockRejectedValue(new Error('network error'))

      const wrapper = mountPage()
      await flushPromises()

      // page still renders normally with null activeReminder
      expect(wrapper.find('.coins-grid').exists()).toBe(true)
      const card = wrapper.findComponent(CoinCard)
      expect(card.props('activeReminder')).toBeNull()
    })
  })
})
