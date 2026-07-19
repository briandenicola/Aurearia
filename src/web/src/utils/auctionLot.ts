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

export interface AuctionLotStatusSourceLabel {
  text: string
  title: string
}

/**
 * Label describing whether a lot's Won/Lost outcome was auto-detected by sync (currently
 * CNG only) or set by an explicit manual override (the only path NumisBids lots have today
 * — see specs/_backlog/F021/F022). Only meaningful for won/lost; returns null otherwise.
 */
export function auctionLotStatusSourceLabel(
  lot: Pick<AuctionLot, 'status' | 'statusSource'>,
): AuctionLotStatusSourceLabel | null {
  if (lot.status !== 'won' && lot.status !== 'lost') return null
  if (lot.statusSource === 'sync') {
    return { text: 'Auto-detected', title: 'This outcome was detected automatically from the provider — no action needed.' }
  }
  return { text: 'Manually set', title: 'This outcome was set by a manual status override.' }
}
