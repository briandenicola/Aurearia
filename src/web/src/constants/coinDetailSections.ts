// T001: Coin detail section constants for #219
// These constants define the standard section links shown on the coin detail overview page

export interface CoinDetailSection {
  id: 'journal' | 'notes' | 'actions' | 'analysis' | 'health' | 'valuation'
  title: string
  description: string
  route: (coinId: number) => string
}

export const COIN_DETAIL_SECTIONS: Record<string, CoinDetailSection> = {
  journal: {
    id: 'journal',
    title: 'Activity Journal',
    description: 'Timeline of events and actions',
    route: (coinId: number) => `/coin/${coinId}/journal`,
  },
  health: {
    id: 'health',
    title: 'Metadata Health',
    description: 'Completeness score and quality checklist',
    route: (coinId: number) => `/coin/${coinId}/health`,
  },
  notes: {
    id: 'notes',
    title: 'Notes',
    description: 'Personal notes and observations',
    route: (coinId: number) => `/coin/${coinId}/notes`,
  },
  actions: {
    id: 'actions',
    title: 'Actions',
    description: 'Upload images, estimate value, and more',
    route: (coinId: number) => `/coin/${coinId}/actions`,
  },
  analysis: {
    id: 'analysis',
    title: 'AI Analysis',
    description: 'Computer vision analysis and insights',
    route: (coinId: number) => `/coin/${coinId}/analysis`,
  },
  valuation: {
    id: 'valuation',
    title: 'Value Trend',
    description: 'Estimated value over time',
    route: (coinId: number) => `/coin/${coinId}/valuation`,
  },
}

export const SECTION_ORDER: CoinDetailSection['id'][] = ['journal', 'health', 'notes', 'actions', 'analysis', 'valuation']
