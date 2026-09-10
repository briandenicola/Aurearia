import type { AuctionLotStatus } from '@/types/auctions'
import type { CoinSetType } from '@/types/sets'

export type QuickAccessTargetType = 'coin' | 'coin_set' | 'auction_lot' | 'calendar_event'

export interface QuickAccessCoinDTO {
  name: string
  classification: 'owned' | 'wishlist'
  primaryImageUrl: string
}

export interface QuickAccessCoinSetDTO {
  name: string
  setType: CoinSetType
  color: string
  icon: string
}

export interface QuickAccessAuctionLotDTO {
  title: string
  status: Extract<AuctionLotStatus, 'watching' | 'bidding'>
  auctionHouse: string
  saleDate: string | null
  auctionEndTime: string | null
  imageUrl: string
}

export interface QuickAccessCalendarEventDTO {
  title: string
  auctionHouse: string
  startDate: string | null
  endDate: string | null
}

interface QuickAccessItemBase {
  id: number
  pinnedAt: string
}

export type QuickAccessCoinItem = QuickAccessItemBase & {
  type: 'coin'
  coin: QuickAccessCoinDTO
  coinSet?: never
  auctionLot?: never
  calendarEvent?: never
}

export type QuickAccessCoinSetItem = QuickAccessItemBase & {
  type: 'coin_set'
  coin?: never
  coinSet: QuickAccessCoinSetDTO
  auctionLot?: never
  calendarEvent?: never
}

export type QuickAccessAuctionLotItem = QuickAccessItemBase & {
  type: 'auction_lot'
  coin?: never
  coinSet?: never
  auctionLot: QuickAccessAuctionLotDTO
  calendarEvent?: never
}

export type QuickAccessCalendarEventItem = QuickAccessItemBase & {
  type: 'calendar_event'
  coin?: never
  coinSet?: never
  auctionLot?: never
  calendarEvent: QuickAccessCalendarEventDTO
}

export type QuickAccessItem =
  | QuickAccessCoinItem
  | QuickAccessCoinSetItem
  | QuickAccessAuctionLotItem
  | QuickAccessCalendarEventItem

export interface QuickAccessListDTO {
  items: QuickAccessItem[]
}
