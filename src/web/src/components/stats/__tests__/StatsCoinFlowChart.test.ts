import { flushPromises, shallowMount, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StatsCoinFlowChart from '@/components/stats/StatsCoinFlowChart.vue'
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
    era: 'ancient',
    mint: 'Rome',
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

// Five coins with purchase dates across two periods, various rulers/eras/types
const sampleCoins: Coin[] = [
  makeCoin({ id: 1, ruler: 'Augustus',  era: 'ancient',  denomination: 'Denarius',    purchaseDate: '2020-01-15' }),
  makeCoin({ id: 2, ruler: 'Nero',      era: 'ancient',  denomination: 'Sestertius',  purchaseDate: '2021-03-10' }),
  makeCoin({ id: 3, ruler: 'Alexander', era: 'ancient',  denomination: 'Tetradrachm', purchaseDate: '2020-06-20', category: 'Greek' }),
  makeCoin({ id: 4, ruler: 'Justinian', era: 'medieval', denomination: 'Solidus',     purchaseDate: '2022-09-05', category: 'Byzantine' }),
  makeCoin({ id: 5, ruler: 'Trajan',    era: 'ancient',  denomination: 'Aureus',      purchaseDate: '2021-11-30' }),
]

describe('StatsCoinFlowChart', () => {
  beforeEach(() => {
    mockGetCoins.mockReset()
    mockGetCoins.mockResolvedValue({
      data: { coins: sampleCoins, total: sampleCoins.length, page: 1, limit: 100 },
    })
  })

  // ── Loading state ──────────────────────────────────────────────────────────

  it('shows a loading spinner while fetching coins', () => {
    mockGetCoins.mockReturnValue(new Promise(() => {}))
    const wrapper = shallowMount(StatsCoinFlowChart)
    expect(wrapper.find('.spinner').exists()).toBe(true)
  })

  // ── Fetch parameters ───────────────────────────────────────────────────────

  it('fetches active (non-wishlist, non-sold) coins sorted by purchase_date asc on mount', async () => {
    shallowMount(StatsCoinFlowChart)
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

  // ── Purchase-date filtering ────────────────────────────────────────────────

  it('only maps coins that have a purchaseDate', async () => {
    // 5 coins returned, but 2 lack purchaseDate — only 3 should appear in the chart
    const mixed = [
      makeCoin({ id: 1, purchaseDate: '2021-01-01' }),
      makeCoin({ id: 2, purchaseDate: '2021-04-01' }),
      makeCoin({ id: 3, purchaseDate: '2022-07-01' }),
      makeCoin({ id: 4, purchaseDate: null }),
      makeCoin({ id: 5, purchaseDate: null }),
    ]
    mockGetCoins.mockResolvedValue({ data: { coins: mixed, total: mixed.length, page: 1, limit: 100 } })
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.text()).toContain('3 coins with purchase date')
  })

  it('shows empty state when all loaded coins lack a purchaseDate', async () => {
    const noDates = [
      makeCoin({ id: 1, purchaseDate: null }),
      makeCoin({ id: 2, purchaseDate: null }),
      makeCoin({ id: 3, purchaseDate: null }),
    ]
    mockGetCoins.mockResolvedValue({ data: { coins: noDates, total: noDates.length, page: 1, limit: 100 } })
    const wrapper = shallowMount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.find('.py-8').exists()).toBe(true)
    expect(wrapper.find('svg').exists()).toBe(false)
  })

  // ── Chart anatomy ──────────────────────────────────────────────────────────

  it('renders the flow chart card with correct heading after coins load', async () => {
    const wrapper = shallowMount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.find('.stats-section').exists()).toBe(true)
    expect(wrapper.text()).toContain('Acquisition Flow')
    expect(wrapper.text()).toContain('Coins Bought by Period, Ruler, Era')
  })

  it('renders four column header labels: Purchase Period, Ruler, Era, Type', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    const labels = wrapper.findAll('.text-label').map((el) => el.text())
    expect(labels).toContain('Purchase Period')
    expect(labels).toContain('Ruler')
    expect(labels).toContain('Era')
    expect(labels).toContain('Type')
  })

  it('renders a Sankey/alluvial SVG chart when acquisition data is available', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.findAll('.sankey-node').length).toBeGreaterThan(0)
    expect(wrapper.findAll('.sankey-flow').length).toBeGreaterThan(0)
  })

  it('shows coin count with purchase date in the header', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.text()).toContain('5 coins with purchase date')
  })

  // ── Node breakdown correctness ─────────────────────────────────────────────

  it('renders sankey nodes for each purchase period, ruler, era, and type present', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    // sampleCoins: 3 periods (2020,2021,2022) + 5 rulers + 2 eras (ancient,medieval) + 5 types = 15 nodes
    expect(wrapper.findAll('.sankey-node').length).toBeGreaterThanOrEqual(15)
  })

  it('ruler nodes collectively account for all coins with purchase dates', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    // Each ruler label includes "(N)" count. Extract all counts from ruler node labels.
    const text = wrapper.text()
    // All 5 rulers appear (Augustus, Nero, Alexander, Justinian, Trajan)
    expect(text).toContain('Augustus')
    expect(text).toContain('Nero')
    expect(text).toContain('Alexander')
    expect(text).toContain('Justinian')
    expect(text).toContain('Trajan')
  })

  it('era nodes reflect the era distribution of purchased coins', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    const text = wrapper.text()
    // 4 ancient, 1 medieval in sampleCoins
    expect(text).toContain('ancient')
    expect(text).toContain('medieval')
  })

  it('type nodes reflect denomination distribution (denomination preferred over category)', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    const text = wrapper.text()
    expect(text).toContain('Denarius')
    expect(text).toContain('Tetradrachm')
    expect(text).toContain('Solidus')
  })

  // ── Empty state ────────────────────────────────────────────────────────────

  it('shows empty state when fewer than 3 coins have a purchaseDate', async () => {
    mockGetCoins.mockResolvedValue({
      data: {
        coins: [makeCoin({ id: 1 }), makeCoin({ id: 2 })],
        total: 2,
        page: 1,
        limit: 100,
      },
    })
    const wrapper = shallowMount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.find('.py-8').exists()).toBe(true)
    expect(wrapper.find('svg').exists()).toBe(false)
  })

  it('empty state message mentions purchase date', async () => {
    mockGetCoins.mockResolvedValue({
      data: { coins: [makeCoin({ id: 1, purchaseDate: null })], total: 1, page: 1, limit: 100 },
    })
    const wrapper = shallowMount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.find('.py-8').text()).toContain('purchase date')
  })

  // ── Top-N grouping ─────────────────────────────────────────────────────────

  it('groups rulers beyond top-8 into "Other Rulers"', async () => {
    // 9 distinct rulers → 8 top + 1 grouped
    const manyRulers = Array.from({ length: 9 }, (_, i) =>
      makeCoin({ id: i + 1, ruler: `Ruler${i}`, purchaseDate: `2021-0${(i % 9) + 1}-01` }),
    )
    mockGetCoins.mockResolvedValue({
      data: { coins: manyRulers, total: manyRulers.length, page: 1, limit: 100 },
    })
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.text()).toContain('Other Rulers')
  })

  it('groups types beyond top-8 into "Other Types"', async () => {
    const manyTypes = Array.from({ length: 9 }, (_, i) =>
      makeCoin({ id: i + 1, denomination: `Type${i}`, purchaseDate: `2021-0${(i % 9) + 1}-01` }),
    )
    mockGetCoins.mockResolvedValue({
      data: { coins: manyTypes, total: manyTypes.length, page: 1, limit: 100 },
    })
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    expect(wrapper.text()).toContain('Other Types')
  })

  it('shows exactly 5 distinct ruler nodes when there are 5 rulers (no "Other Rulers" node)', async () => {
    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()
    // sampleCoins has 5 distinct rulers; all fit within TOP_N=8 → no grouping node added
    // Ruler nodes sit at COL_X[1]; verify all 5 rulers are individually visible
    const text = wrapper.text()
    expect(text).toContain('Augustus')
    expect(text).toContain('Nero')
    expect(text).toContain('Alexander')
    expect(text).toContain('Justinian')
    expect(text).toContain('Trajan')
    // SVG nodes: 3 periods + 5 rulers + 2 eras + 5 types = 15 (no "Other Rulers" node added)
    expect(wrapper.findAll('.sankey-node')).toHaveLength(15)
  })

  // ── Pagination ─────────────────────────────────────────────────────────────

  it('paginates until all coins are loaded', async () => {
    const page1 = Array.from({ length: 100 }, (_, i) =>
      makeCoin({ id: i + 1, ruler: 'Augustus', purchaseDate: `2021-01-${String((i % 28) + 1).padStart(2, '0')}` }),
    )
    const page2 = [makeCoin({ id: 101, ruler: 'Nero', purchaseDate: '2022-03-15' })]
    mockGetCoins
      .mockResolvedValueOnce({ data: { coins: page1, total: 101, page: 1, limit: 100 } })
      .mockResolvedValueOnce({ data: { coins: page2, total: 101, page: 2, limit: 100 } })

    const wrapper = mount(StatsCoinFlowChart)
    await flushPromises()

    expect(mockGetCoins).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('101 coins with purchase date')
  })
})

