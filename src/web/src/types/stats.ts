// stats types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.

export interface StatsResponse {
  totalCoins: number
  totalWishlist: number
  byCategory: { category: string; count: number }[]
  byMaterial: { material: string; count: number }[]
  byGrade: { grade: string; count: number }[]
  byEra: { era: string; count: number }[]
  byRuler: { ruler: string; count: number }[]
  byPriceRange: { range: string; count: number }[]
  values: {
    totalPurchasePrice: number
    totalCurrentValue: number
    avgPurchasePrice: number
    avgCurrentValue: number
  }
}

export type InvestmentBreakdownDimension = 'purchase-year' | 'material'

export interface InvestmentBreakdownSegment {
  label: string
  year: number | null
  month: number | null
  invested: number
  currentValue: number
  gainLoss: number
  gainLossPct: number | null
  coinCount: number
  missingCurrentValueCount: number
  missingPurchasePriceCount: number
}

export interface InvestmentMovementCoin {
  coinId: number
  name: string
  initialValue: number
  currentValue: number
  changeAmount: number
  changePct: number
  changeExplanation?: string | null
}

export interface StaleValuationCoin {
  coinId: number
  name: string
  lastValuationAt: string | null
}

export type InvestmentBreakdownResponse =
  | InvestmentBreakdownSegment[]
  | {
      dimension?: InvestmentBreakdownDimension
      segments: InvestmentBreakdownSegment[]
      topIncreases?: InvestmentMovementCoin[]
      topDrops?: InvestmentMovementCoin[]
      staleValuations?: StaleValuationCoin[]
    }

export type HealthGrade = 'A' | 'B' | 'C' | 'D' | 'F'

export type HealthTrendDirection = 'up' | 'flat' | 'down' | 'unavailable'

export type HealthChecklistDimension = 'metadata' | 'images' | 'valuation' | 'ai'

export type HealthChecklistSeverity = 'high' | 'medium' | 'low'

export type HealthQuickAction = 'edit_metadata' | 'upload_images' | 'run_valuation' | 'run_ai_analysis'

export interface HealthWeights {
  metadata: number
  imageCoverage: number
  valuationFreshness: number
  aiCoverage: number
}

export interface HealthDimensions {
  metadata: number
  imageCoverage: number
  valuationFreshness: number
  aiCoverage: number
}

export interface CollectionHealthTrend {
  status: 'available' | 'unavailable'
  delta: number | null
  direction: HealthTrendDirection
}

export interface CollectionHealthSummary {
  score: number
  grade: HealthGrade
  eligibleCoinCount: number
  weights: HealthWeights
  dimensions: HealthDimensions
  trend30d: CollectionHealthTrend
}

export interface MissingChecklistItem {
  key: string
  dimension: HealthChecklistDimension
  label: string
  severity: HealthChecklistSeverity
  actionHint: HealthQuickAction
}

export interface CoinHealthItem {
  coinId: number
  title: string
  score: number
  grade: HealthGrade
  dimensions: HealthDimensions
  missingItems: MissingChecklistItem[]
  quickActions: HealthQuickAction[]
}

export interface CoinHealthListResponse {
  coins: CoinHealthItem[]
  pagination: {
    page: number
    limit: number
    total: number
  }
}

export interface MissingFieldStat {
  key: string
  count: number
  percentage: number
}

export interface AdminHealthSummaryResponse {
  medianScore: number
  lowScorePercentage: number
  lowScoreThreshold: number
  eligibleCoinCount: number
  topMissingFields: MissingFieldStat[]
}

export interface PortfolioSummary {
  totalCoins: number
  totalValue: number
  totalInvested: number
  categories: { category: string; count: number }[]
  materials: { material: string; count: number }[]
  eras: { era: string; count: number }[]
  rulers: { ruler: string; count: number }[]
  topCoins: { name: string; category: string; currentValue: number | null; ruler: string; era: string }[]
  missingFields?: Record<string, number>
}
