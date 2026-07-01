import { mount } from '@vue/test-utils'
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
  push: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  updateAuctionLotStatus: mocks.updateAuctionLotStatus,
  updateAuctionLot: mocks.updateAuctionLot,
  convertAuctionLotToCoin: mocks.convertAuctionLotToCoin,
  deleteAuctionLot: mocks.deleteAuctionLot,
  listCalendarEvents: mocks.listCalendarEvents,
  linkAuctionLotEvent: mocks.linkAuctionLotEvent,
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
