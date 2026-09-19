export interface CollectorProfileInput {
  budgetMin: number | null
  budgetMax: number | null
  currency: string | null
  preferredPeriods: string[]
  preferredCategories: string[]
  excludedCategories: string[]
  preferredDealers: string[]
  collectingGoals: string[]
}

export interface CollectorProfile extends CollectorProfileInput {
  updatedAt: string | null
  isDefault: boolean
}
