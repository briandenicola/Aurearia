import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AuctionsPage from '../AuctionsPage.vue'
import { getAuctionLotCounts, getAuctionLots, syncNumisBidsWatchlist } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

vi.mock('@/api/client', () => ({
  getAuctionLots: vi.fn(),
  getAuctionLotCounts: vi.fn(),
  syncNumisBidsWatchlist: vi.fn(),
  listCalendarEvents: vi.fn(),
  bulkLinkAuctionLotEvent: vi.fn(),
  onTokenRefreshed: vi.fn(),
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

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
    vi.mocked(getAuctionLots).mockReset()
    vi.mocked(getAuctionLotCounts).mockReset()
    vi.mocked(syncNumisBidsWatchlist).mockReset()
    vi.mocked(getAuctionLots).mockResolvedValue({ data: { lots: [], total: 0, page: 1, limit: 50 } } as Awaited<ReturnType<typeof getAuctionLots>>)
    vi.mocked(getAuctionLotCounts).mockResolvedValue({ data: { counts: {} } } as Awaited<ReturnType<typeof getAuctionLotCounts>>)
    vi.mocked(syncNumisBidsWatchlist).mockResolvedValue({ data: { synced: 3, lots: [] } } as Awaited<ReturnType<typeof syncNumisBidsWatchlist>>)
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
    expect(wrapper.text()).toContain('Synced 3 lots from CNG Auctions')
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
    expect(wrapper.text()).toContain('Configure auction provider credentials in Settings before syncing')
  })
})
