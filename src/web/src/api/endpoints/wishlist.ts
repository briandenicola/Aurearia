// wishlist endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  AdjustWishlistSearchAlertCriteriaInput,
  AlertCandidate,
  AlertCandidateListResponse,
  AlertCandidateState,
  AlertRun,
  AlertRunListResponse,
  AlertRunResult,
  AvailabilityRun,
  AvailabilityRunListResponse,
  AvailabilityRunSummary,
  CandidateProvenanceStatus,
  ConvertWishlistSearchAlertCandidateInput,
  ConvertWishlistSearchAlertCandidateResponse,
  DismissWishlistSearchAlertCandidateInput,
  WishlistSearchAlert,
  WishlistSearchAlertInput,
  WishlistSearchAlertListResponse,
} from '@/types'

// Wishlist Search Alerts (acquisition discovery; separate from availability checking)
export const listWishlistSearchAlerts = (params?: { active?: boolean; page?: number; limit?: number }) =>
  api.get<WishlistSearchAlertListResponse>('/wishlist/search-alerts', { params })

export const createWishlistSearchAlert = (alert: WishlistSearchAlertInput) =>
  api.post<WishlistSearchAlert>('/wishlist/search-alerts', alert)

export const getWishlistSearchAlert = (id: number) =>
  api.get<WishlistSearchAlert>(`/wishlist/search-alerts/${id}`)

export const updateWishlistSearchAlert = (id: number, alert: WishlistSearchAlertInput) =>
  api.put<WishlistSearchAlert>(`/wishlist/search-alerts/${id}`, alert)

export const deleteWishlistSearchAlert = (id: number) =>
  api.delete<void>(`/wishlist/search-alerts/${id}`)

export const runWishlistSearchAlert = (id: number, maxCandidates = 20) =>
  api.post<AlertRunResult>(`/wishlist/search-alerts/${id}/run`, { maxCandidates })

export const listWishlistSearchAlertRuns = (id: number, params?: { page?: number; limit?: number }) =>
  api.get<AlertRunListResponse>(`/wishlist/search-alerts/${id}/runs`, { params })

export const getWishlistSearchAlertRun = (alertId: number, runId: number) =>
  api.get<AlertRun>(`/wishlist/search-alerts/${alertId}/runs/${runId}`)

export const listWishlistSearchAlertCandidates = (id: number, params?: { state?: AlertCandidateState | ''; provenanceStatus?: CandidateProvenanceStatus | ''; page?: number; limit?: number }) =>
  api.get<AlertCandidateListResponse>(`/wishlist/search-alerts/${id}/candidates`, { params })

export const dismissWishlistSearchAlertCandidate = (alertId: number, candidateId: number, input: DismissWishlistSearchAlertCandidateInput) =>
  api.post<AlertCandidate>(`/wishlist/search-alerts/${alertId}/candidates/${candidateId}/dismiss`, input)

export const restoreWishlistSearchAlertCandidate = (alertId: number, candidateId: number) =>
  api.post<AlertCandidate>(`/wishlist/search-alerts/${alertId}/candidates/${candidateId}/restore`)

export const convertWishlistSearchAlertCandidate = (alertId: number, candidateId: number, input: ConvertWishlistSearchAlertCandidateInput) =>
  api.post<ConvertWishlistSearchAlertCandidateResponse>(`/wishlist/search-alerts/${alertId}/candidates/${candidateId}/convert`, input)

export const adjustWishlistSearchAlertCriteria = (alertId: number, input: AdjustWishlistSearchAlertCriteriaInput) =>
  api.post<WishlistSearchAlert>(`/wishlist/search-alerts/${alertId}/criteria-adjustments`, input)

// Availability checks
export const checkWishlistAvailability = () =>
  api.post<AvailabilityRunSummary>('/wishlist/check-availability')

// Owner wishlist availability run history (Feature 353) — scoped to the caller.
export const listMyAvailabilityRuns = (page = 1, limit = 20) =>
  api.get<AvailabilityRunListResponse>('/wishlist/availability-runs', { params: { page, limit } })

export const getMyAvailabilityRunDetail = (runId: number) =>
  api.get<AvailabilityRun>(`/wishlist/availability-runs/${runId}`)
