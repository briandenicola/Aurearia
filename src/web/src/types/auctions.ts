// auctions types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { Category, Coin } from '@/types/coin'

export type AuctionLotStatus = 'watching' | 'bidding' | 'won' | 'lost' | 'passed'

export type AuctionSource = 'numisbids' | 'cng'

export type AuctionLotStatusSource = 'sync' | 'manual'

export interface AuctionLot {
  id: number
  numisBidsUrl: string
  source: AuctionSource
  sourceUrl: string
  sourceLotId?: string
  sourceSaleId?: string
  saleId: string
  lotNumber: number
  auctionHouse: string
  saleName: string
  saleDate: string | null
  auctionEndTime: string | null
  title: string
  description: string
  notes: string
  category: Category
  estimate: number | null
  initialBid: number | null
  currentBid: number | null
  maxBid: number | null
  winningBid: number | null
  currency: string
  status: AuctionLotStatus
  statusSource?: AuctionLotStatusSource
  imageUrl: string
  coinId: number | null
  coin?: Coin
  eventId: number | null
  userId: number
  createdAt: string
  updatedAt: string
}

export interface AuctionLotListResponse {
  lots: AuctionLot[]
  total: number
}

export type PriceAlertDirection = 'above' | 'below'

export interface PriceAlert {
  id: number
  auctionLotId: number
  lotTitle?: string
  targetPrice: number
  direction: PriceAlertDirection
  isTriggered: boolean
  triggeredAt: string | null
  createdAt: string
}

export interface BidReminder {
  id: number
  auctionLotId: number
  lotTitle?: string
  minutesBefore: number
  isNotified: boolean
  notifiedAt: string | null
  createdAt: string
}

export type BidRecommendationConfidence = 'insufficient_data' | 'low' | 'medium' | 'high'

export interface BidRecommendation {
  suggestedMaxBid: number | null
  confidence: BidRecommendationConfidence
  sampleSize: number
  rationale: string
}

export type MarketSignalStatus = 'unavailable' | 'ok'

export type MarketTrendDirection = 'rising' | 'stable' | 'declining' | 'unknown'

export interface MarketSignal {
  status: MarketSignalStatus
  trendDirection?: MarketTrendDirection
  priceLow?: number
  priceHigh?: number
  currency?: string
  sampleSize?: number
  rationale: string
  sources?: string[]
}

export interface CalendarEventDetail {
  id: number
  title: string
  auctionHouse: string
  startDate: string | null
  endDate: string | null
  url: string
  notes: string
  createdAt: string
  updatedAt: string
}

export interface AuctionEndingRun {
  id: number
  triggerType: 'scheduled' | 'manual'
  triggerUserId: number | null
  status: 'queued' | 'running' | 'success' | 'error'
  lotsChecked: number
  alertsSent: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage: string
  createdAt: string
}

export interface AuctionWatchBidDigestRun {
  id: number
  triggerType: 'scheduled' | 'manual'
  triggerUserId: number | null
  status: 'running' | 'success' | 'error'
  lotsChecked: number
  digestsSent: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage: string
  createdAt: string
}

export interface AuctionAlertReminderRun {
  id: number
  triggerType: 'scheduled' | 'manual'
  triggerUserId: number | null
  status: 'running' | 'success' | 'error'
  lotsChecked?: number
  alertsChecked?: number
  alertsTriggered?: number
  alertsSent?: number
  priceAlertsTriggered?: number
  remindersChecked?: number
  remindersNotified?: number
  remindersSent?: number
  bidRemindersSent?: number
  durationMs: number
  startedAt: string
  completedAt: string | null
  errorMessage?: string
  createdAt: string
}
