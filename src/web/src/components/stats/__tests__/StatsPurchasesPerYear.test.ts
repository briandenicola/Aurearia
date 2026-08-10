import { flushPromises, mount, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StatsPurchasesPerYear from '@/components/stats/StatsPurchasesPerYear.vue'
import type { Coin } from '@/types'

const mockGetCoins = vi.fn()
vi.mock('@/api/client', () => ({
  getCoins: (params?: Record<string, unknown>) => mockGetCoins(params),
}))

function makeCoin(overrides: Partial<Coin>): Coin {
  return {
    id: 1,
    name: 'Test Coin',
    category: 'Roman',
    denomination: 'Denarius',
    ruler: 'Augustus',
    romanImperialFigureId: null,
    era: 'ancient',
    mint: 'Rome',
    mintLocationId: null,
    mintLocation: null,
    material: 'Silver',
    weightGrams: null,
    diameterMm: null,
    grade: 'VF',
    obverseInscription: '',
    reverseInscription: '',
    obverseDescription: '',
    reverseDescription: '',
    rarityRating: '',
    purchasePrice: null,
    currentValue: null,
    purchaseDate: '2021-06-15',
    purchaseLocation: '',
    vendorSku: '',
    vendorInvoice: '',
    storageLocationId: null,
    storageLocation: null,
    notes: '',
    aiAnalysis: '',
    obverseAnalysis: '',
    reverseAnalysis: '',
    referenceUrl: '',
    referenceText: '',
    isWishlist: false,
    isSold: false,
    soldPrice: null,
    soldDate: null,
    soldTo: '',
    isPrivate: false,
    listingStatus: '',
    listingCheckedAt: null,
    listingCheckReason: '',
    userId: 1,
    images: [],
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('StatsPurchasesPerYear', () => {
  beforeEach(() => {
    mockGetCoins.mockReset()
    mockGetCoins.mockResolvedValue({
      data: {
        coins: [
          makeCoin({ id: 1, purchaseDate: '2020-01-10' }),
          makeCoin({ id: 2, purchaseDate: '2021-03-02' }),
          makeCoin({ id: 3, purchaseDate: '2021-06-11' }),
        ],
        total: 3,
        page: 1,
        limit: 100,
      },
    })
  })

  it('shows a loading spinner while fetching', () => {
    mockGetCoins.mockReturnValue(new Promise(() => {}))
    const wrapper = shallowMount(StatsPurchasesPerYear)
    expect(wrapper.find('.spinner').exists()).toBe(true)
  })

  it('fetches active non-wishlist, non-sold coins sorted by purchase_date', async () => {
    shallowMount(StatsPurchasesPerYear)
    await flushPromises()
    expect(mockGetCoins).toHaveBeenCalledWith({
      wishlist: 'false',
      sold: 'false',
      page: 1,
      limit: 100,
      sort: 'purchase_date',
      order: 'asc',
    })
  })

  it('renders yearly labels and total purchased count when data exists', async () => {
    const wrapper = mount(StatsPurchasesPerYear)
    await flushPromises()
    expect(wrapper.text()).toContain('Coins Purchased Per Year')
    expect(wrapper.text()).toContain('2020')
    expect(wrapper.text()).toContain('2021')
    expect(wrapper.text()).toContain('3 purchased coins')
    expect(wrapper.find('svg').exists()).toBe(true)
  })

  it('shows empty state when no coins include purchaseDate', async () => {
    mockGetCoins.mockResolvedValue({
      data: {
        coins: [makeCoin({ id: 1, purchaseDate: null }), makeCoin({ id: 2, purchaseDate: null })],
        total: 2,
        page: 1,
        limit: 100,
      },
    })
    const wrapper = shallowMount(StatsPurchasesPerYear)
    await flushPromises()
    expect(wrapper.text()).toContain('Add purchase dates to your coins')
    expect(wrapper.find('svg').exists()).toBe(false)
  })

  it('uses a horizontally scrollable container with dynamic chart min width', async () => {
    const wrapper = mount(StatsPurchasesPerYear)
    await flushPromises()
    const scrollRegion = wrapper.find('.overflow-x-auto')
    expect(scrollRegion.exists()).toBe(true)
    const chartHost = scrollRegion.find('div')
    expect(chartHost.attributes('style')).toContain('min-width')
  })

  it('paginates until all coins are loaded', async () => {
    const page1 = Array.from({ length: 100 }, (_, index) =>
      makeCoin({
        id: index + 1,
        purchaseDate: `2021-01-${String((index % 28) + 1).padStart(2, '0')}`,
      }),
    )
    const page2 = [makeCoin({ id: 101, purchaseDate: '2022-02-02' })]
    mockGetCoins
      .mockResolvedValueOnce({ data: { coins: page1, total: 101, page: 1, limit: 100 } })
      .mockResolvedValueOnce({ data: { coins: page2, total: 101, page: 2, limit: 100 } })

    const wrapper = mount(StatsPurchasesPerYear)
    await flushPromises()

    expect(mockGetCoins).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('101 purchased coins')
  })
})
