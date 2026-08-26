import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AuctionsPage from '../AuctionsPage.vue'
import { getAuctionLot, getAuctionLotCounts, getAuctionLots, listAlerts, listReminders, syncNumisBidsWatchlist } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import type { AuctionLot } from '@/types'

function makeLot(overrides: Partial<AuctionLot> = {}): AuctionLot {
  return {
    id: 1,
    numisBidsUrl: '',
    source: 'numisbids',
    sourceUrl: '',
    saleId: 'sale-1',
    lotNumber: 1,
    auctionHouse: 'CNG',
    saleName: 'Triton XXVI',
    saleDate: null,
    auctionEndTime: null,
    title: 'Denarius',
    description: '',
    notes: '',
    category: 'roman',
    estimate: null,
    initialBid: null,
    currentBid: null,
    maxBid: null,
    winningBid: null,
    currency: 'USD',
    status: 'watching',
    imageUrl: '',
    coinId: null,
    eventId: null,
    userId: 1,
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

vi.mock('@/api/client', () => ({
  getAuctionLot: vi.fn(),
  getAuctionLots: vi.fn(),
  getAuctionLotCounts: vi.fn(),
  syncNumisBidsWatchlist: vi.fn(),
  listAlerts: vi.fn(),
  listReminders: vi.fn(),
  listCalendarEvents: vi.fn(),
  bulkLinkAuctionLotEvent: vi.fn(),
  onTokenRefreshed: vi.fn(),
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

// Mutable so a test can arrive on /auctions?lot=<id>, the deep link auction notifications use.
const route = { query: {} as Record<string, string> }
const routerReplace = vi.fn()

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => route,
    useRouter: () => ({ replace: routerReplace }),
  }
})

function mountPage() {
  return mount(AuctionsPage, {
    global: {
      stubs: {
        AuctionBulkActionBar: true,
        AuctionLotCard: true,
        AuctionLotDetailModal: true,
        AuctionStatusFilter: true,
        CheckSquare: true,
        CirclePlus: true,
        ExternalLink: true,
        ImportLotModal: true,
        Plus: true,
        PullToRefresh: { template: '<div><slot /></div>' },
        RefreshCw: true,
        SafeExternalLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('AuctionsPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    route.query = {}
    routerReplace.mockReset()
    vi.mocked(getAuctionLot).mockReset()
    vi.mocked(getAuctionLots).mockReset()
    vi.mocked(getAuctionLotCounts).mockReset()
    vi.mocked(syncNumisBidsWatchlist).mockReset()
    vi.mocked(listAlerts).mockReset()
    vi.mocked(listReminders).mockReset()
    vi.mocked(getAuctionLots).mockResolvedValue({ data: { lots: [], total: 0, page: 1, limit: 50 } } as Awaited<ReturnType<typeof getAuctionLots>>)
    vi.mocked(getAuctionLotCounts).mockResolvedValue({ data: { counts: {} } } as Awaited<ReturnType<typeof getAuctionLotCounts>>)
    vi.mocked(syncNumisBidsWatchlist).mockResolvedValue({ data: { synced: 3, lots: [] } } as Awaited<ReturnType<typeof syncNumisBidsWatchlist>>)
    vi.mocked(listAlerts).mockResolvedValue({ data: { alerts: [] } } as Awaited<ReturnType<typeof listAlerts>>)
    vi.mocked(listReminders).mockResolvedValue({ data: { reminders: [] } } as Awaited<ReturnType<typeof listReminders>>)
    vi.mocked(getAuctionLot).mockResolvedValue({ data: makeLot({ id: 42, title: 'Julia Domna AR Denarius' }) } as Awaited<ReturnType<typeof getAuctionLot>>)
  })

  // Auction sync notifications (in-app and Pushover) link to /auctions?lot=<id>; the linked
  // lot is usually filtered out of the default Bidding list, so it has to be fetched directly.
  describe('lot deep link', () => {
    it('opens the linked lot even when it is not in the filtered list', async () => {
      route.query = { lot: '42' }

      const wrapper = mountPage()
      await flushPromises()

      expect(getAuctionLot).toHaveBeenCalledWith(42)
      const modal = wrapper.findComponent({ name: 'AuctionLotDetailModal' })
      expect(modal.exists()).toBe(true)
      expect(modal.props('lot')).toMatchObject({ id: 42, title: 'Julia Domna AR Denarius' })
    })

    it('leaves the page on the list when the linked lot can no longer be loaded', async () => {
      vi.mocked(getAuctionLot).mockRejectedValue(new Error('not found'))
      route.query = { lot: '42' }

      const wrapper = mountPage()
      await flushPromises()

      expect(wrapper.findComponent({ name: 'AuctionLotDetailModal' }).exists()).toBe(false)
    })

    it('ignores a malformed lot parameter', async () => {
      route.query = { lot: 'not-a-lot' }

      const wrapper = mountPage()
      await flushPromises()

      expect(getAuctionLot).not.toHaveBeenCalled()
      expect(wrapper.findComponent({ name: 'AuctionLotDetailModal' }).exists()).toBe(false)
    })

    it('clears the deep link when the lot modal is closed', async () => {
      route.query = { lot: '42' }

      const wrapper = mountPage()
      await flushPromises()

      wrapper.findComponent({ name: 'AuctionLotDetailModal' }).vm.$emit('close')
      await flushPromises()

      expect(routerReplace).toHaveBeenCalledWith({ query: {} })
      expect(wrapper.findComponent({ name: 'AuctionLotDetailModal' }).exists()).toBe(false)
    })
  })

  it('syncs only CNG when only CNG credentials are configured', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 1,
      username: 'collector',
      role: 'user',
      email: 'collector@example.com',
      avatarPath: '',
      isPublic: false,
      bio: '',
      zipCode: '',
      numisBidsConfigured: false,
      cngConfigured: true,
    }

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('.header-actions .btn-secondary').trigger('click')
    await flushPromises()

    expect(syncNumisBidsWatchlist).toHaveBeenCalledTimes(1)
    expect(syncNumisBidsWatchlist).toHaveBeenCalledWith('cng')
    expect(wrapper.text()).toContain('Synced 3 lots from CNG Auctions with hosted outcome detection where available')
  })

  it('does not sync any provider when no auction credentials are configured', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 1,
      username: 'collector',
      role: 'user',
      email: 'collector@example.com',
      avatarPath: '',
      isPublic: false,
      bio: '',
      zipCode: '',
      numisBidsConfigured: false,
      cngConfigured: false,
    }

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('.header-actions .btn-secondary').trigger('click')
    await flushPromises()

    expect(syncNumisBidsWatchlist).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Configure auction provider credentials in Settings before syncing watchlists')
  })

  it('explains manual NumisBids outcomes in the empty state', async () => {
    const auth = useAuthStore()
    auth.user = {
      id: 1,
      username: 'collector',
      role: 'user',
      email: 'collector@example.com',
      avatarPath: '',
      isPublic: false,
      bio: '',
      zipCode: '',
      numisBidsConfigured: true,
      cngConfigured: false,
    }

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('NumisBids outcomes are updated manually')
  })

  describe('grouping by Auction House -> Sale', () => {
    beforeEach(() => {
      vi.mocked(getAuctionLots).mockResolvedValue({
        data: {
          lots: [
            makeLot({ id: 1, auctionHouse: 'CNG', saleName: 'Triton XXVI', lotNumber: 101 }),
            makeLot({ id: 2, auctionHouse: 'CNG', saleName: 'Triton XXVI', lotNumber: 102 }),
            makeLot({ id: 3, auctionHouse: 'CNG', saleName: 'Electronic Auction 550', lotNumber: 5 }),
            makeLot({ id: 4, auctionHouse: 'NumisBids', saleName: 'Spring Sale', lotNumber: 12 }),
          ],
          total: 4,
          page: 1,
          limit: 50,
        },
      } as Awaited<ReturnType<typeof getAuctionLots>>)
    })

    it('defaults to grouped by Auction House then Sale on the default (bidding) view', async () => {
      const wrapper = mountPage()
      await flushPromises()

      const text = wrapper.text()
      expect(text).toContain('CNG')
      expect(text).toContain('NumisBids')
      expect(text).toContain('Triton XXVI')
      expect(text).toContain('Electronic Auction 550')
      expect(text).toContain('Spring Sale')
      expect(wrapper.findAllComponents({ name: 'AuctionLotCard' }).length).toBe(4)
      const toggle = wrapper.find('[aria-label="Auction grouping"] button')
      expect(toggle.attributes('aria-pressed')).toBe('true')
    })

    it('preserves an explicit user choice to turn grouping off for the rest of the visit', async () => {
      const wrapper = mountPage()
      await flushPromises()

      const toggle = wrapper.find('[aria-label="Auction grouping"] button')
      await toggle.trigger('click')
      await flushPromises()

      expect(toggle.attributes('aria-pressed')).toBe('false')
      // Flat grid: still all 4 lots rendered, just no group headings container.
      expect(wrapper.findAllComponents({ name: 'AuctionLotCard' }).length).toBe(4)

      // Refetch (e.g. from a pull-to-refresh) must not silently re-enable grouping.
      await wrapper.vm.$nextTick()
      expect(wrapper.find('[aria-label="Auction grouping"] button').attributes('aria-pressed')).toBe('false')
    })

    it('does not show the grouping toggle for statuses other than watching/bidding', async () => {
      vi.mocked(getAuctionLotCounts).mockResolvedValue({ data: { counts: {} } } as Awaited<ReturnType<typeof getAuctionLotCounts>>)
      const wrapper = mountPage()
      await flushPromises()

      // Simulate the status filter emitting a non-watching/bidding status, as AuctionStatusFilter is stubbed.
      await wrapper.findComponent({ name: 'AuctionStatusFilter' }).vm.$emit('update:modelValue', 'won')
      await flushPromises()

      expect(wrapper.find('[aria-label="Auction grouping"]').exists()).toBe(false)
    })
  })
})
