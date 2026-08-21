import { mount, flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { ref } from 'vue'
import CoinDetailValuationPage from '@/pages/CoinDetailValuationPage.vue'
import { getCoinValueHistory } from '@/api/client'
import type { CoinValueHistory, Coin } from '@/types'

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '42' } }),
}))

vi.mock('@/api/client', () => ({
  getCoinValueHistory: vi.fn(),
}))

// Mutable so individual tests can swap the coin state
const coinRef = ref<Partial<Coin>>({
  id: 42,
  isWishlist: false,
  isSold: false,
  purchasePrice: 100,
  purchaseDate: '2024-01-01T00:00:00Z',
})

vi.mock('@/composables/useCoinDetailContext', () => ({
  useCoinDetailContext: () => ({ coin: coinRef }),
}))

// Stub the shell so it passes `coin` through the default scoped slot
const ShellStub = {
  name: 'CoinDetailSectionPageShell',
  props: ['sectionTitle'],
  computed: {
    coin() {
      return coinRef.value
    },
  },
  template: '<div><slot :coin="coin" /></div>',
}

const defaultMountOptions = {
  global: { stubs: { CoinDetailSectionPageShell: ShellStub } },
}

/** Build a minimal CoinValueHistory array from partial overrides. */
function makeHistory(overrides: Array<Partial<CoinValueHistory>>): CoinValueHistory[] {
  return overrides.map((o, i) => ({
    id: i + 1,
    coinId: 42,
    userId: 1,
    value: 100 + i * 10,
    confidence: 'high',
    recordedAt: `2024-0${i + 1}-01T00:00:00Z`,
    ...o,
  }))
}

describe('CoinDetailValuationPage', () => {
  beforeEach(() => {
    coinRef.value = {
      id: 42,
      isWishlist: false,
      isSold: false,
      purchasePrice: 100,
      purchaseDate: '2024-01-01T00:00:00Z',
    }
    vi.mocked(getCoinValueHistory).mockResolvedValue({ data: [] } as never)
  })

  // ── Access gate ──────────────────────────────────────────────────────────

  it('shows gate message for wishlist coins', async () => {
    coinRef.value = { ...coinRef.value, isWishlist: true }
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('Value tracking is only available for active coins')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  it('shows gate message for sold coins', async () => {
    coinRef.value = { ...coinRef.value, isSold: true }
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('Value tracking is only available for active coins')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  // ── No-data state ────────────────────────────────────────────────────────

  it('shows "not enough data" when there is no history and no purchase date', async () => {
    coinRef.value = { ...coinRef.value, purchasePrice: null, purchaseDate: null }
    vi.mocked(getCoinValueHistory).mockResolvedValue({ data: [] } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('Not enough data points to chart')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  // ── Table rendering ──────────────────────────────────────────────────────

  it('renders the history table when there is one entry', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ value: 150, recordedAt: '2024-06-01T00:00:00Z' }]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.text()).toContain('Value History')
  })

  it('renders table even with only one entry when chart gate would block it', async () => {
    coinRef.value = { ...coinRef.value, purchasePrice: null, purchaseDate: null }
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ value: 150 }]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.find('svg').exists()).toBe(false)
  })

  it('shows Date, Value, Change, Source column headers', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ value: 150 }]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    const headers = wrapper.findAll('th').map((th) => th.text())
    expect(headers).toContain('Date')
    expect(headers).toContain('Value')
    expect(headers).toContain('Change')
    expect(headers).toContain('Source')
  })

  it('orders table rows newest-first', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([
        { id: 1, value: 100, recordedAt: '2024-01-01T00:00:00Z' },
        { id: 2, value: 200, recordedAt: '2024-06-01T00:00:00Z' },
      ]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    const rows = wrapper.findAll('tbody tr')
    expect(rows[0]!.text()).toContain('$200.00')
    expect(rows[1]!.text()).toContain('$100.00')
  })

  it('shows — for the oldest entry change and signed delta for newer entries', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([
        { id: 1, value: 100, recordedAt: '2024-01-01T00:00:00Z' },
        { id: 2, value: 150, recordedAt: '2024-06-01T00:00:00Z' },
      ]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    // Newest-first: June row (change = +$50), January row (change = —)
    const rows = wrapper.findAll('tbody tr')
    expect(rows[0]!.text()).toContain('+')
    expect(rows[1]!.text()).toContain('—')
  })

  // ── Source label resolution ──────────────────────────────────────────────

  it('shows "AI Scheduled" for source ai_scheduled', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ source: 'ai_scheduled', confidence: 'high' }]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('AI Scheduled')
  })

  it('shows "AI Estimate" for source ai_estimate', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ source: 'ai_estimate', confidence: 'medium' }]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('AI Estimate')
  })

  it('shows "Manual" for source manual', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ source: 'manual', confidence: 'manual' }]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('Manual')
    expect(wrapper.text()).not.toContain('AI Scheduled')
  })

  it('infers Manual from confidence="manual" when source is absent (legacy row)', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: [{ id: 1, coinId: 42, userId: 1, value: 100, confidence: 'manual', recordedAt: '2024-01-01T00:00:00Z' }],
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('Manual')
    expect(wrapper.text()).not.toContain('AI Scheduled')
  })

  it('infers AI Scheduled from high confidence when source is absent (legacy row)', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: [{ id: 1, coinId: 42, userId: 1, value: 100, confidence: 'high', recordedAt: '2024-01-01T00:00:00Z' }],
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.text()).toContain('AI Scheduled')
  })

  // ── Chart gate ───────────────────────────────────────────────────────────

  it('renders chart when >= 2 data points (purchase date + 1 entry)', async () => {
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ value: 150, recordedAt: '2024-06-01T00:00:00Z' }]),
    } as never)
    // coinRef.value has purchasePrice + purchaseDate — 2 total chart points
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.find('table').exists()).toBe(true)
  })

  it('does not render chart when only 1 total data point', async () => {
    coinRef.value = { ...coinRef.value, purchasePrice: null, purchaseDate: null }
    vi.mocked(getCoinValueHistory).mockResolvedValue({
      data: makeHistory([{ value: 150 }]),
    } as never)
    const wrapper = mount(CoinDetailValuationPage, defaultMountOptions)
    await flushPromises()
    expect(wrapper.find('svg').exists()).toBe(false)
    expect(wrapper.find('table').exists()).toBe(true)
  })
})
