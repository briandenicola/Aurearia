// sets endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  AddCoinToSetRequest,
  Coin,
  CoinSetAnalytics,
  CoinSetComparison,
  CoinSetCompletion,
  CoinSetDetail,
  CoinSetSnapshot,
  CoinSetSummary,
  CoinSetTemplate,
  CreateCoinSetFromCsvRequest,
  CreateCoinSetRequest,
  CreateSetBuilderRunRequest,
  RegenerateSetProposalRequest,
  ReorderSetCoinsRequest,
  SetBuilderRun,
  SetProposal,
  SmartCriteriaGroup,
  SmartCriteriaTemplate,
  SmartSetPreview,
  SuggestedSmartCriteria,
  UpdateCoinSetRequest,
  UpdateSetProposalRequest,
} from '@/types'

// Sets
export const getSets = () => api.get<{ sets: CoinSetSummary[] }>('/sets')

export const getSet = (id: number) => api.get<CoinSetDetail>(`/sets/${id}`)

export const createSet = (data: CreateCoinSetRequest) => api.post<CoinSetDetail>('/sets', data)

export const createSetBuilderRun = (data: CreateSetBuilderRunRequest) => api.post<{ run: SetBuilderRun }>('/set-builder/runs', data)

export const listSetProposals = () => api.get<{ proposals: SetProposal[] }>('/set-builder/proposals')

export const getSetProposal = (id: number) => api.get<SetProposal>(`/set-builder/proposals/${id}`)

export const updateSetProposal = (id: number, data: UpdateSetProposalRequest) => api.put<SetProposal>(`/set-builder/proposals/${id}`, data)

export const approveSetProposal = (id: number) => api.post<{ set: CoinSetDetail }>(`/set-builder/proposals/${id}/approve`)

export const rejectSetProposal = (id: number, reason = '') => api.post<{ status: string }>(`/set-builder/proposals/${id}/reject`, { reason })

export const regenerateSetProposal = (id: number, data: RegenerateSetProposalRequest) => api.post<{ run: SetBuilderRun }>(`/set-builder/proposals/${id}/regenerate`, data)

export const updateSet = (id: number, data: UpdateCoinSetRequest) => api.put<CoinSetDetail>(`/sets/${id}`, data)

export const deleteSet = (id: number) => api.delete(`/sets/${id}`)

export const getCoinsInSet = (id: number) => api.get<{ coins: Coin[] }>(`/sets/${id}/coins`)

export const addCoinToSet = (setId: number, data: AddCoinToSetRequest) => api.post(`/sets/${setId}/coins`, data)

export const reorderSetCoins = (setId: number, data: ReorderSetCoinsRequest) => api.put(`/sets/${setId}/coins/order`, data)

export const removeCoinFromSet = (setId: number, coinId: number) => api.delete(`/sets/${setId}/coins/${coinId}`)

// US2: Templates and Completion
export const getSetTemplates = () => api.get<{ templates: CoinSetTemplate[] }>('/sets/templates')

export const getSetCompletion = (setId: number) => api.get<CoinSetCompletion>(`/sets/${setId}/completion`)

export const createSetFromCsv = (data: CreateCoinSetFromCsvRequest) => api.post<CoinSetDetail>('/sets/import-csv', data)

export const createSetSnapshot = (setId: number) => api.post<CoinSetSnapshot>(`/sets/${setId}/snapshot`)

export const getSetTrends = (setId: number, range = '1y') => api.get<{ snapshots: CoinSetSnapshot[] }>(`/sets/${setId}/trends`, { params: { range } })

export const getSetAnalytics = (setId: number) => api.get<CoinSetAnalytics>(`/sets/${setId}/analytics`)

export const compareSets = (setIds: number[], range = '1y') => api.post<{ sets: CoinSetComparison[] }>('/sets/compare', { setIds, range })

export const previewSmartSet = (criteria: SmartCriteriaGroup) => api.post<SmartSetPreview>('/sets/preview-smart', criteria)

export const getSuggestedCriteria = () => api.get<{ suggestions: SuggestedSmartCriteria[] }>('/sets/suggested-criteria')

export const listCriteriaTemplates = () => api.get<{ templates: SmartCriteriaTemplate[] }>('/sets/criteria-templates')

export const saveCriteriaTemplate = (data: { name: string; description?: string; criteria: SmartCriteriaGroup }) =>
  api.post<SmartCriteriaTemplate>('/sets/criteria-templates', data)

export const deleteCriteriaTemplate = (id: number) => api.delete(`/sets/criteria-templates/${id}`)
