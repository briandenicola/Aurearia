import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import QuickAccessPage from '../QuickAccessPage.vue'
import type { QuickAccessItem } from '@/types'

const push = vi.fn()
const refresh = vi.fn()
const unpin = vi.fn()
const items = ref<QuickAccessItem[]>([])
const loading = ref(false)
const error = ref('')
const busy = ref(false)

vi.mock('vue-router', () => ({
  useRouter: () => ({ push }),
}))

vi.mock('@/composables/useQuickAccess', () => ({
  useQuickAccess: () => ({
    items,
    loading,
    error,
    refresh,
    unpin,
    isBusy: () => busy,
  }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ showToast: vi.fn() }),
}))

const mixedItems: QuickAccessItem[] = [
  { type: 'coin', id: 1, pinnedAt: '2026-09-10T04:00:00Z', coin: { name: 'Coin One', classification: 'owned', primaryImageUrl: '' } },
  { type: 'coin_set', id: 2, pinnedAt: '2026-09-09T04:00:00Z', coinSet: { name: 'Set Two', setType: 'goal', color: '#6b7280', icon: 'Crown' } },
  { type: 'auction_lot', id: 3, pinnedAt: '2026-09-08T04:00:00Z', auctionLot: { title: 'Lot Three', status: 'bidding', auctionHouse: 'CNG', saleDate: null, auctionEndTime: null, imageUrl: '' } },
  { type: 'calendar_event', id: 4, pinnedAt: '2026-09-07T04:00:00Z', calendarEvent: { title: 'Event Four', auctionHouse: '', startDate: null, endDate: null } },
]

function mountPage() {
  return mount(QuickAccessPage, {
    global: {
      stubs: {
        PullToRefresh: { template: '<div><slot /></div>' },
        AuthenticatedImage: true,
      },
    },
  })
}

describe('QuickAccessPage', () => {
  beforeEach(() => {
    items.value = []
    loading.value = false
    error.value = ''
    busy.value = false
    push.mockReset()
    refresh.mockReset()
    unpin.mockReset()
    refresh.mockResolvedValue(undefined)
    unpin.mockResolvedValue(undefined)
  })

  it('renders all variants in server order with null metadata safely', () => {
    items.value = mixedItems
    const wrapper = mountPage()
    const text = wrapper.text()

    expect(text.indexOf('Coin One')).toBeLessThan(text.indexOf('Set Two'))
    expect(text.indexOf('Set Two')).toBeLessThan(text.indexOf('Lot Three'))
    expect(text.indexOf('Lot Three')).toBeLessThan(text.indexOf('Event Four'))
    expect(text).toContain('Date not set')
    const setMedia = wrapper.findAll('article')[1]!.find('[style]')
    expect(setMedia.attributes('style')).toContain('rgb(107, 114, 128)')
    expect(setMedia.find('svg').classes()).toContain('lucide-crown')
  })

  it('uses the fallback set icon and ignores invalid set colors', () => {
    items.value = [{
      type: 'coin_set',
      id: 5,
      pinnedAt: '2026-09-09T04:00:00Z',
      coinSet: { name: 'Fallback Set', setType: 'goal', color: 'javascript:red', icon: '<script>' },
    }]
    const wrapper = mountPage()
    const setMedia = wrapper.get('article > div')

    expect(setMedia.find('svg').classes()).toContain('lucide-layers')
    expect(setMedia.attributes('style')).toBeUndefined()
  })

  it('invokes Retry and keeps the card layout mobile-safe', async () => {
    items.value = mixedItems
    error.value = 'Unable to refresh'
    const wrapper = mountPage()

    await wrapper.get('button').trigger('click')

    expect(refresh).toHaveBeenCalledTimes(1)
    expect(wrapper.get('article').classes()).toEqual(expect.arrayContaining([
      'grid-cols-[4rem_minmax(0,1fr)_auto]',
      'max-sm:grid-cols-[3.25rem_minmax(0,1fr)_auto]',
    ]))
    expect(wrapper.get('article .min-w-0').exists()).toBe(true)
  })

  it('navigates each row to its canonical destination', async () => {
    items.value = mixedItems
    const wrapper = mountPage()
    const rows = wrapper.findAll('article')

    for (const row of rows) await row.trigger('click')

    expect(push.mock.calls).toEqual([
      ['/coin/1'],
      ['/sets/2'],
      [{ path: '/auctions', query: { lot: '3' } }],
      [{ path: '/calendar', query: { event: '4' } }],
    ])
  })

  it('shows empty, loading, and retained-list error states', async () => {
    let wrapper = mountPage()
    expect(wrapper.text()).toContain('Nothing pinned yet')
    wrapper.unmount()

    loading.value = true
    wrapper = mountPage()
    const loadingStatus = wrapper.find('[aria-label="Loading Quick Access"]')
    expect(loadingStatus.attributes('role')).toBe('status')
    expect(wrapper.get('button[aria-label="Refreshing Quick Access"]').attributes('aria-busy')).toBe('true')
    wrapper.unmount()

    loading.value = false
    items.value = mixedItems
    error.value = 'Unable to refresh'
    wrapper = mountPage()
    expect(wrapper.text()).toContain('Unable to refresh')
    expect(wrapper.text()).toContain('Coin One')
  })

  it('uses an explicit unpin control without triggering row navigation', async () => {
    items.value = [mixedItems[0]!]
    const wrapper = mountPage()
    await wrapper.find('button[aria-label="Unpin Coin One from Quick Access"]').trigger('click')
    await flushPromises()

    expect(unpin).toHaveBeenCalledWith('coin', 1)
    expect(push).not.toHaveBeenCalled()
  })

  it('announces and disables a pending unpin without duplicating an error message', () => {
    items.value = [mixedItems[0]!]
    busy.value = true
    const wrapper = mountPage()

    const unpinButton = wrapper.get('button[aria-label="Removing Coin One from Quick Access"]')
    expect(unpinButton.attributes('disabled')).toBeDefined()
    expect(unpinButton.attributes('aria-busy')).toBe('true')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })
})
