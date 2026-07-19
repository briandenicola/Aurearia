import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AuctionLotCard from '../AuctionLotCard.vue'
import type { AuctionLot } from '@/types'

vi.mock('@/composables/useProxiedImage', () => ({
  useProxiedImage: () => ({ proxiedImageUrl: { value: '' } }),
}))

const safeExternalLinkStub = {
  props: ['href'],
  template: '<a :href="href"><slot /></a>',
}

function buildAuctionLot(overrides: Partial<AuctionLot> = {}): AuctionLot {
  return {
    id: 1,
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
    maxBid: null,
    winningBid: null,
    currency: 'USD',
    status: 'watching',
    imageUrl: '',
    coinId: null,
    eventId: null,
    userId: 1,
    createdAt: '2026-07-01T00:00:00Z',
    updatedAt: '2026-07-01T00:00:00Z',
    ...overrides,
  }
}

describe('AuctionLotCard', () => {
  it('shows a needs-attention badge for a watching lot whose auction already closed', () => {
    const wrapper = mount(AuctionLotCard, {
      props: { lot: buildAuctionLot({ status: 'watching', auctionEndTime: '2020-01-01T00:00:00Z' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    expect(wrapper.text()).toContain('Needs attention')
  })

  it('does not show a needs-attention badge for a won lot even if its close time has passed', () => {
    const wrapper = mount(AuctionLotCard, {
      props: { lot: buildAuctionLot({ status: 'won', auctionEndTime: '2020-01-01T00:00:00Z' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    expect(wrapper.text()).not.toContain('Needs attention')
  })

  it('does not show a needs-attention badge for an active bidding lot', () => {
    const wrapper = mount(AuctionLotCard, {
      props: { lot: buildAuctionLot({ status: 'bidding', auctionEndTime: '2099-01-01T00:00:00Z' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    expect(wrapper.text()).not.toContain('Needs attention')
  })

  it('shows an auto-detected label for a sync-resolved won lot', () => {
    const wrapper = mount(AuctionLotCard, {
      props: { lot: buildAuctionLot({ status: 'won', statusSource: 'sync' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    expect(wrapper.text()).toContain('Auto-detected')
  })

  it('shows a manually-set label for a manually-resolved lost lot', () => {
    const wrapper = mount(AuctionLotCard, {
      props: { lot: buildAuctionLot({ status: 'lost', statusSource: 'manual' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    expect(wrapper.text()).toContain('Manually set')
  })

  it('does not show a status-source label for non-terminal statuses', () => {
    const wrapper = mount(AuctionLotCard, {
      props: { lot: buildAuctionLot({ status: 'bidding', statusSource: 'sync' }) },
      global: { stubs: { SafeExternalLink: safeExternalLinkStub } },
    })
    expect(wrapper.text()).not.toContain('Auto-detected')
    expect(wrapper.text()).not.toContain('Manually set')
  })
})
