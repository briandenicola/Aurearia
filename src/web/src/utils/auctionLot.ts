import type { AuctionLot } from '@/types'

/**
 * True when a lot's real-world auction has already closed but its tracked status is still
 * watching/bidding — i.e. nothing (sync or the user) has confirmed the actual outcome yet.
 * CNG lots normally resolve themselves on the next sync; NumisBids lots currently never do
 * (see specs/_backlog/F021/F022), so this is the only signal a NumisBids user gets that a
 * lot needs a manual check.
 */
export function auctionLotNeedsAttention(lot: Pick<AuctionLot, 'status' | 'auctionEndTime' | 'saleDate'>): boolean {
  if (lot.status !== 'watching' && lot.status !== 'bidding') return false
  const closeTime = lot.auctionEndTime ?? lot.saleDate
  if (!closeTime) return false
  return new Date(closeTime).getTime() < Date.now()
}
