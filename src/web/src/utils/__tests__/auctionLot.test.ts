import { describe, it, expect } from 'vitest'
import { auctionLotNeedsAttention, auctionLotStatusSourceLabel } from '../auctionLot'

describe('auctionLotNeedsAttention', () => {
  it('flags a watching lot whose auctionEndTime has already passed', () => {
    expect(auctionLotNeedsAttention({
      status: 'watching', auctionEndTime: '2020-01-01T00:00:00Z', saleDate: null, maxBid: null, currentBid: null,
    })).toBe(true)
  })

  it('flags a bidding lot whose auctionEndTime has already passed', () => {
    expect(auctionLotNeedsAttention({
      status: 'bidding', auctionEndTime: '2020-01-01T00:00:00Z', saleDate: null, maxBid: null, currentBid: null,
    })).toBe(true)
  })

  it('falls back to saleDate when auctionEndTime is not set', () => {
    expect(auctionLotNeedsAttention({
      status: 'watching', auctionEndTime: null, saleDate: '2020-01-01T00:00:00Z', maxBid: null, currentBid: null,
    })).toBe(true)
  })

  it('does not flag a lot whose close time is in the future', () => {
    expect(auctionLotNeedsAttention({
      status: 'watching', auctionEndTime: '2099-01-01T00:00:00Z', saleDate: null, maxBid: null, currentBid: null,
    })).toBe(false)
  })

  it('does not flag a lot with no close time at all', () => {
    expect(auctionLotNeedsAttention({ status: 'watching', auctionEndTime: null, saleDate: null, maxBid: null, currentBid: null })).toBe(false)
  })

  it('does not flag already-resolved lots (won/lost/passed), even if close time has passed', () => {
    for (const status of ['won', 'lost', 'passed'] as const) {
      expect(auctionLotNeedsAttention({
        status, auctionEndTime: '2020-01-01T00:00:00Z', saleDate: null, maxBid: null, currentBid: null,
      })).toBe(false)
    }
  })

  it('flags a bidding lot when current bid exceeds user max bid (outbid)', () => {
    expect(auctionLotNeedsAttention({
      status: 'bidding', auctionEndTime: null, saleDate: null, maxBid: 550, currentBid: 600,
    })).toBe(true)
  })

  it('does not flag a bidding lot that is still winning by max bid when close time is unknown', () => {
    expect(auctionLotNeedsAttention({
      status: 'bidding', auctionEndTime: null, saleDate: null, maxBid: 650, currentBid: 600,
    })).toBe(false)
  })
})

describe('auctionLotStatusSourceLabel', () => {
  it('returns null for non-terminal statuses', () => {
    for (const status of ['watching', 'bidding', 'passed'] as const) {
      expect(auctionLotStatusSourceLabel({ status, statusSource: 'sync' })).toBeNull()
    }
  })

  it('labels a won lot with statusSource sync as auto-detected', () => {
    expect(auctionLotStatusSourceLabel({ status: 'won', statusSource: 'sync' })?.text).toBe('Auto-detected')
  })

  it('labels a lost lot with statusSource manual as manually set', () => {
    expect(auctionLotStatusSourceLabel({ status: 'lost', statusSource: 'manual' })?.text).toBe('Manually set')
  })

  it('defaults to manually set when statusSource is missing (legacy rows)', () => {
    expect(auctionLotStatusSourceLabel({ status: 'won', statusSource: undefined })?.text).toBe('Manually set')
  })
})
