// stats endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  TimeMachineBounds,
  TimeMachineSnapshot,
  CollectionHealthSummary,
  EmperorTrackerResult,
  FeaturedCoin,
  InvestmentBreakdownDimension,
  InvestmentBreakdownResponse,
  StatsResponse,
} from '@/types'

export const getEmperorTrackerProgress = () =>
  api.get<EmperorTrackerResult>('/stats/emperors')

export const updateEmperorTrackerHighlight = (figureId: number, coinId: number | null) =>
  api.put<void>(`/stats/emperors/highlights/${figureId}`, { coinId })

// Stats
export const getStats = () => api.get<StatsResponse>('/stats')

export const getDistribution = () => api.get<{ cells: { era: string; category: string; count: number }[] }>('/stats/distribution')

export const getInvestmentBreakdown = (dimension: InvestmentBreakdownDimension) =>
  api.get<InvestmentBreakdownResponse>('/stats/investment-breakdown', { params: { dimension } })

export const getCollectionHealthSummary = () => api.get<CollectionHealthSummary>('/stats/health')

// Coin of the Day
export const getLatestFeaturedCoin = () =>
  api.get<FeaturedCoin>('/featured-coins/latest')

export const getFeaturedCoin = (id: number) =>
  api.get<FeaturedCoin>(`/featured-coins/${id}`)

// Collection Time Machine
export const getTimeMachineSnapshot = (date: string) =>
  api.get<TimeMachineSnapshot>('/stats/time-machine', { params: { date } })

export const getTimeMachineBounds = () =>
  api.get<TimeMachineBounds>('/stats/time-machine/bounds')
