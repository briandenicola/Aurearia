// emperors types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { Coin } from '@/types/coin'

// F028: Roman Emperor collection tracker
export type ImperialFigureRole = 'emperor' | 'empress' | 'caesar' | 'usurper' | 'other'

export type ImperialFigureRegion = 'west' | 'east'

export type RarityTier = 'common' | 'scarce' | 'rare' | 'very_rare'

export interface RomanImperialFigure {
  id: number
  name: string
  aliases: string[]
  role: ImperialFigureRole
  region: ImperialFigureRegion
  dynasty: string
  reignStart: number
  reignEnd: number
  sortOrder: number
  rarityTier: RarityTier
  notes?: string
}

export interface ImperialFigureSlot {
  figure: RomanImperialFigure
  coin: Coin | null
  coins: Coin[]
  highlightedCoinId: number | null
}

export interface DynastyProgress {
  dynasty: string
  owned: number
  total: number
  figures: ImperialFigureSlot[]
}

export interface CategoryProgress {
  roles: ImperialFigureRole[]
  owned: number
  total: number
  percentage: number
  dynasties: DynastyProgress[]
}

export interface EmperorTrackerResult {
  emperor: CategoryProgress
  suggestions: RomanImperialFigure[]
  usurpers?: CategoryProgress
  empresses?: CategoryProgress
  other?: CategoryProgress
}
