import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { reactive, ref } from 'vue'
import CalendarPage from '../CalendarPage.vue'

const mocks = vi.hoisted(() => ({
  getCalendar: vi.fn(),
  getCalendarEvent: vi.fn(),
  createCalendarEvent: vi.fn(),
  updateCalendarEvent: vi.fn(),
  deleteCalendarEvent: vi.fn(),
  proxyImage: vi.fn(),
  replace: vi.fn(),
  pin: vi.fn(),
  unpin: vi.fn(),
  refresh: vi.fn(),
  forget: vi.fn(),
  showToast: vi.fn(),
}))
const route = reactive({ query: {} as Record<string, string | undefined> })
const pinned = ref(false)
const busy = ref(false)
const error = ref('')
enableAutoUnmount(afterEach)

vi.mock('@/api/client', () => ({
  getCalendar: mocks.getCalendar,
  getCalendarEvent: mocks.getCalendarEvent,
  createCalendarEvent: mocks.createCalendarEvent,
  updateCalendarEvent: mocks.updateCalendarEvent,
  deleteCalendarEvent: mocks.deleteCalendarEvent,
  proxyImage: mocks.proxyImage,
}))

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ replace: mocks.replace }),
}))

vi.mock('@/composables/usePwa', () => ({
  usePwa: () => ({ isPwa: false }),
}))

vi.mock('@/composables/usePullToRefresh', () => ({
  usePullToRefresh: () => ({ pullDistance: ref(0), refreshing: ref(false) }),
}))

vi.mock('@/composables/useQuickAccess', () => ({
  useQuickAccess: () => ({
    error,
    pin: mocks.pin,
    unpin: mocks.unpin,
    refresh: mocks.refresh,
    forget: mocks.forget,
    isPinned: () => pinned,
    isBusy: () => busy,
  }),
}))

vi.mock('@/composables/useToast', () => ({
  useToast: () => ({ showToast: mocks.showToast }),
}))

function event(origin: 'manual' | 'auction') {
  return {
    id: 11,
    title: 'Local Coin Show',
    auctionHouse: '',
    startDate: '2026-11-14T00:00:00Z',
    endDate: null,
    url: '',
    notes: '',
    origin,
    createdAt: '',
    updatedAt: '',
  }
}

describe('Calendar Quick Access deep links', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset())
    route.query = {}
    pinned.value = false
    busy.value = false
    error.value = ''
    mocks.getCalendar.mockResolvedValue({ data: { lots: [], events: [] } })
    mocks.refresh.mockResolvedValue(undefined)
    mocks.pin.mockResolvedValue(undefined)
    mocks.unpin.mockResolvedValue(undefined)
    mocks.updateCalendarEvent.mockResolvedValue({ data: event('manual') })
    mocks.deleteCalendarEvent.mockResolvedValue({ data: undefined })
  })

  it('fetches an event outside the active month and removes only event on close', async () => {
    route.query = { event: '11', view: 'agenda' }
    mocks.getCalendarEvent.mockResolvedValue({ data: { event: event('manual'), lots: [] } })
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    expect(mocks.getCalendarEvent).toHaveBeenCalledWith(11)
    expect(wrapper.text()).toContain('Edit Event')
    await wrapper.get('button[aria-label="Close event"]').trigger('click')
    expect(mocks.replace).toHaveBeenCalledWith({ query: { view: 'agenda' } })
  })

  it('pins manual events and excludes auction-origin events', async () => {
    route.query = { event: '11' }
    mocks.getCalendarEvent.mockResolvedValueOnce({ data: { event: event('manual'), lots: [] } })
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    await wrapper.get('button[aria-label="Pin calendar event to Quick Access"]').trigger('click')
    expect(mocks.pin).toHaveBeenCalledWith('calendar_event', 11)

    mocks.getCalendarEvent.mockResolvedValueOnce({ data: { event: event('auction'), lots: [] } })
    route.query = { event: '12' }
    await flushPromises()
    expect(wrapper.find('button[aria-label*="Quick Access"]').exists()).toBe(false)
  })

  it('fails safely when a deep-linked event is unavailable', async () => {
    route.query = { event: '999' }
    mocks.getCalendarEvent.mockRejectedValue(new Error('not found'))
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Calendar')
    expect(wrapper.text()).not.toContain('Edit Event')
  })

  it('clears previously opened private details before a numeric deep-link fetch fails', async () => {
    route.query = { event: '11', view: 'agenda' }
    mocks.getCalendarEvent.mockResolvedValueOnce({ data: { event: event('manual'), lots: [] } })
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Edit Event')

    mocks.getCalendarEvent.mockRejectedValueOnce(new Error('not found'))
    route.query.event = '999'
    await flushPromises()

    expect(wrapper.text()).not.toContain('Edit Event')
    expect(mocks.replace).not.toHaveBeenCalled()
  })

  it('invalidates an in-flight event request when the query becomes invalid', async () => {
    let resolveEvent!: (value: { data: { event: ReturnType<typeof event>; lots: Array<{ id: number }> } }) => void
    route.query = { event: '11' }
    mocks.getCalendarEvent.mockReturnValue(new Promise((resolve) => { resolveEvent = resolve }))
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    route.query = { event: 'removed' }
    await flushPromises()
    resolveEvent({ data: { event: event('manual'), lots: [{ id: 72 }] } })
    await flushPromises()

    expect(wrapper.text()).not.toContain('Edit Event')
    expect(wrapper.text()).not.toContain('Linked Auction Lots 1')
  })

  it('exposes disabled busy semantics for the manual-event pin control', async () => {
    route.query = { event: '11' }
    busy.value = true
    mocks.getCalendarEvent.mockResolvedValue({ data: { event: event('manual'), lots: [] } })
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    const pinButton = wrapper.get('button[aria-label="Updating calendar event Quick Access pin"]')
    expect(pinButton.attributes('disabled')).toBeDefined()
    expect(pinButton.attributes('aria-busy')).toBe('true')
    expect(pinButton.classes()).toEqual(expect.arrayContaining(['h-11', 'w-11']))
  })

  it.each([
    { initiallyPinned: false, action: 'pin' },
    { initiallyPinned: true, action: 'unpin' },
  ])('preserves calendar pin state when $action fails', async ({ initiallyPinned, action }) => {
    route.query = { event: '11' }
    pinned.value = initiallyPinned
    error.value = `Unable to ${action}`
    mocks[action as 'pin' | 'unpin'].mockRejectedValue(new Error('failed'))
    mocks.getCalendarEvent.mockResolvedValue({ data: { event: event('manual'), lots: [] } })
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    const button = wrapper.get(`button[aria-label="${initiallyPinned ? 'Unpin' : 'Pin'} calendar event ${initiallyPinned ? 'from' : 'to'} Quick Access"]`)
    await button.trigger('click')
    await flushPromises()

    expect(button.attributes('aria-pressed')).toBe(String(initiallyPinned))
    expect(mocks.showToast).toHaveBeenCalledWith(`Unable to ${action}`, 'error')
  })

  it('refreshes a pinned DTO after update and removes it after delete', async () => {
    route.query = { event: '11' }
    mocks.getCalendarEvent.mockResolvedValue({ data: { event: event('manual'), lots: [] } })
    mocks.getCalendar.mockResolvedValue({ data: { lots: [], events: [event('manual')] } })
    const wrapper = mount(CalendarPage, {
      global: { stubs: { SafeExternalLink: { template: '<a><slot /></a>' } } },
    })
    await flushPromises()

    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(mocks.refresh).toHaveBeenCalled()

    await wrapper.get('button[title="Delete event"]').trigger('click')
    await flushPromises()
    expect(mocks.forget).toHaveBeenCalledWith('calendar_event', 11)
  })
})
