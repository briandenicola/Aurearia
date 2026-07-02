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

    expect(mocks.updateAuctionLotStatus).toHaveBeenCalledWith(7, 'bidding', 150)
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
    currentBid: null,
    maxBid: 100,
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
