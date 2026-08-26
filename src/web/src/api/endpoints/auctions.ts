// auctions endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  AuctionLot,
  AuctionLotListResponse,
  BidRecommendation,
  BidReminder,
  CalendarEventDetail,
  Coin,
  MarketSignal,
  PriceAlert,
  PriceAlertDirection,
  ShipmentUpsertInput,
} from '@/types'

// Calendar / Auction Events
export const getCalendar = (start?: string, end?: string) => {
  const params: Record<string, string> = {}
  if (start) params.start = start
  if (end) params.end = end
  return api.get('/calendar', { params })
}

export const listCalendarEvents = () => api.get<{ events: Array<{ id: number; title: string; auctionHouse: string; startDate: string | null }> }>('/calendar/events')

export const getCalendarEvent = (id: number) => api.get<{ event: CalendarEventDetail; lots: AuctionLot[] }>(`/calendar/events/${id}`)

export const createCalendarEvent = (data: { title: string; auctionHouse?: string; startDate?: string; endDate?: string; url?: string; notes?: string }) => api.post('/calendar/events', data)

export const updateCalendarEvent = (id: number, data: Record<string, unknown>) => api.put(`/calendar/events/${id}`, data)

export const deleteCalendarEvent = (id: number) => api.delete(`/calendar/events/${id}`)

// Price Alerts
export const listAlerts = () => api.get<{ alerts: PriceAlert[] }>('/alerts')

export const createAlert = (data: { auctionLotId: number; targetPrice: number; direction?: PriceAlertDirection }) => api.post<PriceAlert>('/alerts', data)

export const deleteAlert = (id: number) => api.delete<{ message: string }>(`/alerts/${id}`)

// Bid Reminders
export const listReminders = () => api.get<{ reminders: BidReminder[] }>('/reminders')

export const createReminder = (data: { auctionLotId: number; minutesBefore?: number }) => api.post<BidReminder>('/reminders', data)

export const deleteReminder = (id: number) => api.delete<{ message: string }>(`/reminders/${id}`)

// Auction lots
export const getAuctionLots = (params?: { status?: string; search?: string; source?: string; sort?: string; order?: string; page?: number; limit?: number }) =>
  api.get<AuctionLotListResponse>('/auctions', { params })

export const getAuctionLot = (id: number) => api.get<AuctionLot>(`/auctions/${id}`)

export const getAuctionLotCounts = (params?: { source?: string }) =>
  api.get<{ counts: Record<string, number> }>('/auctions/counts', { params })

export const updateAuctionLotStatus = (id: number, status: string, maxBid?: number | null, winningBid?: number | null) => api.put<AuctionLot>(`/auctions/${id}/status`, { status, ...(maxBid != null ? { maxBid } : {}), ...(winningBid != null ? { winningBid } : {}) })

export const updateAuctionLot = (id: number, data: {
  title?: string
  numisBidsUrl?: string
  auctionHouse?: string
  saleName?: string
  lotNumber?: number
  saleDate?: string | null
  auctionEndTime?: string | null
  description?: string
  notes?: string
  category?: string
  estimate?: number | null
  initialBid?: number | null
  currentBid?: number | null
  maxBid?: number | null
  winningBid?: number | null
  currency?: string
}) => api.put<AuctionLot>(`/auctions/${id}`, data)

export const convertAuctionLotToCoin = (id: number, data?: { shipment?: ShipmentUpsertInput }) => api.post<Coin>(`/auctions/${id}/convert`, data || {})

export const getAuctionLotBidRecommendation = (id: number) => api.get<BidRecommendation>(`/auctions/${id}/bid-recommendation`)

export const getAuctionLotMarketSignal = (id: number) => api.post<MarketSignal>(`/auctions/${id}/market-signal`)

export const deleteAuctionLot = (id: number) => api.delete(`/auctions/${id}`)

export const linkAuctionLotEvent = (id: number, eventId: number | null) => api.put<AuctionLot>(`/auctions/${id}/event`, { eventId })

export const bulkLinkAuctionLotEvent = (lotIds: number[], eventId: number | null) => api.put<{ updated: number }>('/auctions/bulk-link-event', { lotIds, eventId })

export const importAuctionLot = (data: { url: string; source?: string; title?: string; description?: string; auctionHouse?: string; saleName?: string; category?: string; imageUrl?: string; estimate?: number | null; currentBid?: number | null; currency?: string }) =>
  api.post<AuctionLot>('/auctions/import', data)

export const syncNumisBidsWatchlist = (source = 'numisbids') =>
  api.post<{ synced: number; lots: AuctionLot[] }>('/auctions/sync', { source })

export const validateNumisBidsCredentials = (username: string, password: string, source = 'numisbids') =>
  api.post<{ valid: boolean; error?: string }>('/auctions/validate-credentials', { username, password, source })
