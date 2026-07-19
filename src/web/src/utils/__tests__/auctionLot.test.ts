import { describe, it, expect } from 'vitest'
import { auctionLotNeedsAttention } from '../auctionLot'

describe('auctionLotNeedsAttention', () => {
  it('flags a watching lot whose auctionEndTime has already passed', () => {
    expect(auctionLotNeedsAttention({
      status: 'watching', auctionEndTime: '2020-01-01T00:00:00Z', saleDate: null,
    })).toBe(true)
  })

  it('flags a bidding lot whose auctionEndTime has already passed', () => {
    expect(auctionLotNeedsAttention({
      status: 'bidding', auctionEndTime: '2020-01-01T00:00:00Z', saleDate: null,
    })).toBe(true)
  })

  it('falls back to saleDate when auctionEndTime is not set', () => {
    expect(auctionLotNeedsAttention({
      status: 'watching', auctionEndTime: null, saleDate: '2020-01-01T00:00:00Z',
    })).toBe(true)
  })

  it('does not flag a lot whose close time is in the future', () => {
    expect(auctionLotNeedsAttention({
      status: 'watching', auctionEndTime: '2099-01-01T00:00:00Z', saleDate: null,
    })).toBe(false)
  })

  it('does not flag a lot with no close time at all', () => {
    expect(auctionLotNeedsAttention({ status: 'watching', auctionEndTime: null, saleDate: null })).toBe(false)
  })

  it('does not flag already-resolved lots (won/lost/passed), even if close time has passed', () => {
    for (const status of ['won', 'lost', 'passed'] as const) {
      expect(auctionLotNeedsAttention({
        status, auctionEndTime: '2020-01-01T00:00:00Z', saleDate: null,
      })).toBe(false)
    }
  })
})
