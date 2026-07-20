import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AuctionLotDetailModal from '../AuctionLotDetailModal.vue'
import type { AuctionLot } from '@/types'

const mocks = vi.hoisted(() => ({
  updateAuctionLotStatus: vi.fn(),
  updateAuctionLot: vi.fn(),
  convertAuctionLotToCoin: vi.fn(),
  deleteAuctionLot: vi.fn(),
  listCalendarEvents: vi.fn(),
  linkAuctionLotEvent: vi.fn(),
  createAlert: vi.fn(),
  deleteAlert: vi.fn(),
  createReminder: vi.fn(),
  deleteReminder: vi.fn(),
  getAuctionLotBidRecommendation: vi.fn(),
  getAuctionLotMarketSignal: vi.fn(),
  getAgentStatus: vi.fn(),
  push: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  updateAuctionLotStatus: mocks.updateAuctionLotStatus,
  updateAuctionLot: mocks.updateAuctionLot,
  convertAuctionLotToCoin: mocks.convertAuctionLotToCoin,
  deleteAuctionLot: mocks.deleteAuctionLot,
  listCalendarEvents: mocks.listCalendarEvents,
  linkAuctionLotEvent: mocks.linkAuctionLotEvent,
  createAlert: mocks.createAlert,
  deleteAlert: mocks.deleteAlert,
  createReminder: mocks.createReminder,
  deleteReminder: mocks.deleteReminder,
  getAuctionLotBidRecommendation: mocks.getAuctionLotBidRecommendation,
  getAuctionLotMarketSignal: mocks.getAuctionLotMarketSignal,
  getAgentStatus: mocks.getAgentStatus,
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mocks.push }),
}))

vi.mock('@/composables/useProxiedImage', () => ({
  useProxiedImage: () => ({ proxiedImageUrl: { value: '' } }),
}))

const safeExternalLinkStub = {
  props: ['href'],
  template: '<a :href="href"><slot /></a>',
}

describe('AuctionLotDetailModal', () => {
  beforeEach(() => {
    Object.values(mocks).forEach(mock => mock.mockReset())
    mocks.listCalendarEvents.mockResolvedValue({ data: { events: [] } })
    mocks.updateAuctionLotStatus.mockResolvedValue({ data: buildAuctionLot() })
    mocks.createAlert.mockResolvedValue({ data: { id: 91 } })
    mocks.deleteAlert.mockResolvedValue({ data: { message: 'Alert deleted' } })
    mocks.createReminder.mockResolvedValue({ data: { id: 92 } })
    mocks.deleteReminder.mockResolvedValue({ data: { message: 'Reminder deleted' } })
    mocks.getAuctionLotBidRecommendation.mockResolvedValue({
      data: { suggestedMaxBid: null, confidence: 'insufficient_data', sampleSize: 0, rationale: 'Not enough history yet.' },
    })
    mocks.getAgentStatus.mockResolvedValue({ data: { provider: 'anthropic', configured: true } })
    mocks.getAuctionLotMarketSignal.mockResolvedValue({
      data: { status: 'unavailable', rationale: 'Market data lookup is not available on this server.' },
    })
  })

  it('centers the detail card in the desktop overlay', () => {
    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot() },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })

    expect(wrapper.get('.card').classes()).toContain('mx-auto')
  })

  it('persists a max bid change when the status stays bidding', async () => {
    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding', maxBid: 100 }) },
      global: {
        stubs: {
          SafeExternalLink: safeExternalLinkStub,
        },
      },
    })

    const updateButton = wrapper.findAll('button').find(button => button.text().includes('Update Status'))
    expect(updateButton?.attributes('disabled')).toBeDefined()

    await wrapper.find('input.bid-input').setValue('150')
    expect(updateButton?.attributes('disabled')).toBeUndefined()
    await updateButton!.trigger('click')

    expect(mocks.updateAuctionLotStatus).toHaveBeenCalledWith(7, 'bidding', 150, undefined)
  })

  it('persists a winning bid when the status changes to won', async () => {
    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding', maxBid: 100, winningBid: null }) },
      global: {
        stubs: {
          SafeExternalLink: safeExternalLinkStub,
        },
      },
    })

    const statusSelect = wrapper.findAll('select').find(select => select.text().includes('Won'))
    expect(statusSelect).toBeTruthy()
    await statusSelect!.setValue('won')
    await wrapper.get('input.winning-bid-input').setValue('175.5')
    await wrapper.findAll('button').find(button => button.text().includes('Update Status'))!.trigger('click')

    expect(mocks.updateAuctionLotStatus).toHaveBeenCalledWith(7, 'won', undefined, 175.5)
  })

  it('surfaces an error instead of failing silently when a status update is rejected', async () => {
    mocks.updateAuctionLotStatus.mockRejectedValueOnce({ response: { data: { error: 'Invalid status transition' } } })

    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'watching', maxBid: null }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })

    const statusSelect = wrapper.findAll('select').find(select => select.text().includes('Won'))
    await statusSelect!.setValue('won')
    await wrapper.findAll('button').find(button => button.text().includes('Update Status'))!.trigger('click')
    await flushPromises()

    expect(mocks.updateAuctionLotStatus).toHaveBeenCalledWith(7, 'won', undefined, undefined)
    expect(wrapper.text()).toContain('Invalid status transition')
  })

  it('shows a suggested max bid and lets the user apply it', async () => {
    mocks.getAuctionLotBidRecommendation.mockResolvedValue({
      data: { suggestedMaxBid: 833.33, confidence: 'low', sampleSize: 3, rationale: 'Based on 3 of your own resolved lots.' },
    })

    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding', maxBid: 100 }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    expect(mocks.getAuctionLotBidRecommendation).toHaveBeenCalledWith(7)
    expect(wrapper.text()).toContain('Suggested max bid: $833.33')
    expect(wrapper.text()).toContain('low confidence')

    await wrapper.find('button.text-gold').trigger('click')
    expect((wrapper.find('input.bid-input').element as HTMLInputElement).value).toBe('833.33')
  })

  it('shows an honest message instead of a number when there is not enough history', async () => {
    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding', maxBid: 100 }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Not enough history yet.')
    expect(wrapper.text()).not.toContain('Suggested max bid')
  })

  it('creates and deletes price alerts for the selected lot', async () => {
    const wrapper = mount(AuctionLotDetailModal, {
      props: {
        lot: buildAuctionLot({ status: 'watching', currentBid: 125 }),
        priceAlerts: [{
          id: 12,
          auctionLotId: 7,
          targetPrice: 150,
          direction: 'above',
          isTriggered: false,
          triggeredAt: null,
          createdAt: '2026-07-01T00:00:00Z',
        }],
      },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })

    expect(wrapper.text()).toContain('At or above $150.00')
    await wrapper.get('input[aria-label="Target price"]').setValue('175')
    await wrapper.findAll('button').find(button => button.text() === 'Add Alert')!.trigger('click')
    await flushPromises()

    expect(mocks.createAlert).toHaveBeenCalledWith({ auctionLotId: 7, targetPrice: 175, direction: 'above' })
    expect(wrapper.emitted('alertsUpdated')).toBeTruthy()

    await wrapper.findAll('button').find(button => button.text() === 'Delete')!.trigger('click')
    await flushPromises()

    expect(mocks.deleteAlert).toHaveBeenCalledWith(12)
  })

  it('creates and deletes bid reminders for the selected lot', async () => {
    const wrapper = mount(AuctionLotDetailModal, {
      props: {
        lot: buildAuctionLot({ status: 'bidding' }),
        bidReminders: [{
          id: 22,
          auctionLotId: 7,
          minutesBefore: 45,
          isNotified: true,
          notifiedAt: '2026-07-01T10:00:00Z',
          createdAt: '2026-07-01T00:00:00Z',
        }],
      },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })

    expect(wrapper.text()).toContain('45 minutes before close')
    expect(wrapper.text()).toContain('Notified')
    await wrapper.get('input[aria-label="Reminder minutes before close"]').setValue('60')
    await wrapper.findAll('button').find(button => button.text() === 'Add Reminder')!.trigger('click')
    await flushPromises()

    expect(mocks.createReminder).toHaveBeenCalledWith({ auctionLotId: 7, minutesBefore: 60 })

    await wrapper.findAll('button').filter(button => button.text() === 'Delete')[0]?.trigger('click')
    await flushPromises()

    expect(mocks.deleteReminder).toHaveBeenCalledWith(22)
  })

  it('does not fetch the market signal automatically on open', async () => {
    mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    expect(mocks.getAuctionLotMarketSignal).not.toHaveBeenCalled()
  })

  it('fetches and shows the market signal when the user requests it', async () => {
    mocks.getAuctionLotMarketSignal.mockResolvedValue({
      data: {
        status: 'ok',
        trendDirection: 'rising',
        priceLow: 100,
        priceHigh: 250,
        currency: 'USD',
        sampleSize: 6,
        rationale: 'Recent sales trending upward.',
      },
    })

    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === 'Check current market')!.trigger('click')
    await flushPromises()

    expect(mocks.getAuctionLotMarketSignal).toHaveBeenCalledWith(7)
    expect(wrapper.text()).toContain('Market trend: rising')
    expect(wrapper.text()).toContain('$100.00')
    expect(wrapper.text()).toContain('$250.00')
  })

  it('shows a loading state while the market signal request is in flight', async () => {
    let resolveFetch: (value: { data: unknown }) => void = () => {}
    mocks.getAuctionLotMarketSignal.mockReturnValue(new Promise(resolve => { resolveFetch = resolve }))

    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === 'Check current market')!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Checking current auction market')

    resolveFetch({ data: { status: 'unavailable', rationale: 'No usable market data found for this coin.' } })
    await flushPromises()
    expect(wrapper.text()).toContain('No usable market data found for this coin.')
  })

  it('shows the rationale without erroring when the market signal is unavailable', async () => {
    mocks.getAuctionLotMarketSignal.mockResolvedValue({
      data: { status: 'unavailable', rationale: 'AI provider is not configured — set one up in Admin Settings to see current market data for this lot.' },
    })

    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === 'Check current market')!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('AI provider is not configured')
  })

  it('shows a generic error message if the market signal request fails', async () => {
    mocks.getAuctionLotMarketSignal.mockRejectedValue(new Error('network error'))

    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text() === 'Check current market')!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain("Couldn't check the market right now.")
    expect(wrapper.findAll('button').some(button => button.text() === 'Try again')).toBe(true)
  })

  it('shows a nudge to Admin Settings instead of the button when no AI provider is configured', async () => {
    mocks.getAgentStatus.mockResolvedValue({ data: { provider: '', configured: false } })

    const wrapper = mount(AuctionLotDetailModal, {
      props: { lot: buildAuctionLot({ status: 'bidding' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('AI provider not configured')
    expect(wrapper.findAll('button').some(button => button.text() === 'Check current market')).toBe(false)
  })
})

function buildAuctionLot(overrides: Partial<AuctionLot> = {}): AuctionLot {
  return {
    id: 7,
    numisBidsUrl: 'https://auctions.cngcoins.com/lots/view/4-LOT/test',
    source: 'cng',
    sourceUrl: 'https://auctions.cngcoins.com/lots/view/4-LOT/test',
    saleId: '4',
    lotNumber: 1,
    auctionHouse: 'CNG',
    saleName: 'Electronic Auction',
    saleDate: null,
    auctionEndTime: null,
    title: 'CNG test lot',
    description: '',
    notes: '',
    category: 'Roman',
    estimate: null,
    initialBid: null,
    currentBid: null,
    maxBid: 100,
    winningBid: null,
    currency: 'USD',
    status: 'bidding',
    imageUrl: '',
    coinId: null,
    eventId: null,
    userId: 1,
    createdAt: '2026-07-01T00:00:00Z',
    updatedAt: '2026-07-01T00:00:00Z',
    ...overrides,
  }
}
