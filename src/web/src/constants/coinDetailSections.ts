// T001: Coin detail section constants for #219
// These constants define the standard section links shown on the coin detail overview page

export interface CoinDetailSection {
  id: 'journal' | 'notes' | 'actions' | 'analysis'
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
}

export const SECTION_ORDER: CoinDetailSection['id'][] = ['journal', 'notes', 'actions', 'analysis']
