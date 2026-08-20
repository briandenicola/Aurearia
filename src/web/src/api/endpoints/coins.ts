// coins endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  AIJob,
  AIJobStartResponse,
  CatalogRegistry,
  Coin,
  CoinHealthItem,
  CoinHealthListResponse,
  CoinImage,
  CoinJournal,
  CoinListResponse,
  CoinLookupImageRole,
  CoinLookupResponse,
  CoinMutationPayload,
  CoinRecommendation,
  CoinReference,
  CoinReferenceInput,
  CoinValueHistory,
  IntakeCommitRequest,
  IntakeCommitResponse,
  IntakeDraft,
  LegacyMigrationResult,
  NumistaEnrichmentRequest,
  NumistaLookupOutcome,
  NumistaLookupRequest,
  NumistaQueryProposal,
  NumistaQueryProposalRequest,
  NumistaSearchResponse,
  QuickCaptureDraft,
  QuickCaptureDraftInput,
  QuickCaptureDraftListResponse,
  QuickCaptureDraftStatus,
  QuickCaptureDraftUpdateInput,
  QuickCapturePromoteRequest,
  QuickCapturePromotionResponse,
  ShipmentEnvelopeResponse,
  ShipmentStatus,
  ShipmentUpsertInput,
  ValueSnapshot,
} from '@/types'
import { appendOptionalFormValue } from '@/api/endpoints/_shared'

// Coins
export const getCoins = (params?: {
  category?: string
  era?: string
  search?: string
  wishlist?: string
  sold?: string
  tag?: string
  set?: string
  page?: number
  limit?: number
  sort?: string
  order?: string
  seed?: number
}) => api.get<CoinListResponse>('/coins', { params })

const NULLABLE_FIELDS: (keyof Coin)[] = ['weightGrams', 'diameterMm', 'purchasePrice', 'currentValue', 'purchaseDate', 'storageLocationId', 'romanImperialFigureId', 'mintLocationId']

function sanitizeCoin(coin: CoinMutationPayload): CoinMutationPayload {
  const clean: Record<string, unknown> = { ...coin }
  for (const field of NULLABLE_FIELDS) {
    if (clean[field] === '' || clean[field] === undefined) {
      clean[field] = null
    }
  }
  delete clean.storageLocation
  delete clean.mintLocation
  // Default currentValue to purchasePrice if not set (preserve 0 as valid)
  if (clean.currentValue == null && clean.purchasePrice != null) {
    clean.currentValue = clean.purchasePrice
  }
  // Convert date-only strings (YYYY-MM-DD) to RFC3339 for Go
  if (typeof clean.purchaseDate === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(clean.purchaseDate)) {
    clean.purchaseDate = clean.purchaseDate + 'T00:00:00Z'
  }
  return clean as CoinMutationPayload
}

export const getCoin = (id: number) => api.get<Coin>(`/coins/${id}`)

export const createCoin = (coin: CoinMutationPayload) => api.post<Coin>('/coins', sanitizeCoin(coin))

export interface MatchCategoryEraResponse {
  match: string
  matched: boolean
}

export const matchCategoryEra = (type: 'category' | 'era', value: string) =>
  api.post<MatchCategoryEraResponse>('/coins/match-category-era', { type, value })

export async function createIntakeDraft(images: File[], coinCardImage?: File) {
  const formData = new FormData()
  for (const image of images) {
    formData.append('images', image)
  }
  if (coinCardImage) {
    formData.append('coinCardImage', coinCardImage)
  }
  return api.post<IntakeDraft>('/coins/intake/draft', formData)
}


export async function createQuickCaptureDraft(input: QuickCaptureDraftInput) {
  const formData = new FormData()
  appendOptionalFormValue(formData, 'workingTitle', input.workingTitle)
  appendOptionalFormValue(formData, 'dateRange', input.dateRange)
  appendOptionalFormValue(formData, 'era', input.era)
  appendOptionalFormValue(formData, 'acquisitionSource', input.acquisitionSource)
  appendOptionalFormValue(formData, 'notes', input.notes)
  appendOptionalFormValue(formData, 'source', input.source)
  appendOptionalFormValue(formData, 'ngcCertNumber', input.ngcCertNumber)
  appendOptionalFormValue(formData, 'ngcLookupUrl', input.ngcLookupUrl)
  appendOptionalFormValue(formData, 'ngcGrade', input.ngcGrade)
  appendOptionalFormValue(formData, 'labelText', input.labelText)
  appendOptionalFormValue(formData, 'aiConfidence', input.aiConfidence)
  appendOptionalFormValue(formData, 'selectedNumistaId', input.selectedNumistaId)
  appendOptionalFormValue(formData, 'selectedNumistaUrl', input.selectedNumistaUrl)
  if (input.purchasePrice !== undefined && input.purchasePrice !== null) {
    formData.append('purchasePrice', String(input.purchasePrice))
  }
  if (input.obverseImage) formData.append('obverseImage', input.obverseImage)
  if (input.reverseImage) formData.append('reverseImage', input.reverseImage)
  for (const image of input.detailImages ?? []) {
    formData.append('detailImages', image)
  }
  return api.post<QuickCaptureDraft>('/quick-capture/drafts', formData)
}

export const listQuickCaptureDrafts = (params?: { status?: QuickCaptureDraftStatus; page?: number; limit?: number }) =>
  api.get<QuickCaptureDraftListResponse>('/quick-capture/drafts', { params })

export const getQuickCaptureDraft = (id: number) => api.get<QuickCaptureDraft>(`/quick-capture/drafts/${id}`)

export async function updateQuickCaptureDraft(id: number, input: QuickCaptureDraftUpdateInput) {
  const formData = new FormData()
  formData.append('workingTitle', input.workingTitle)
  formData.append('dateRange', input.dateRange)
  formData.append('era', input.era)
  formData.append('acquisitionSource', input.acquisitionSource)
  formData.append('notes', input.notes)
  appendOptionalFormValue(formData, 'source', input.source)
  appendOptionalFormValue(formData, 'ngcCertNumber', input.ngcCertNumber)
  appendOptionalFormValue(formData, 'ngcLookupUrl', input.ngcLookupUrl)
  appendOptionalFormValue(formData, 'ngcGrade', input.ngcGrade)
  appendOptionalFormValue(formData, 'labelText', input.labelText)
  appendOptionalFormValue(formData, 'aiConfidence', input.aiConfidence)
  appendOptionalFormValue(formData, 'selectedNumistaId', input.selectedNumistaId)
  appendOptionalFormValue(formData, 'selectedNumistaUrl', input.selectedNumistaUrl)
  if (input.clearSelectedNumista) formData.append('clearSelectedNumista', 'true')
  if (input.purchasePrice !== null && input.purchasePrice !== undefined) {
    formData.append('purchasePrice', String(input.purchasePrice))
  }
  if (input.removeImageIds) {
    formData.append('removeImageIds', input.removeImageIds)
  }
  if (input.replaceObverse) formData.append('replaceObverse', 'true')
  if (input.replaceReverse) formData.append('replaceReverse', 'true')
  if (input.obverseImage) formData.append('obverseImage', input.obverseImage)
  if (input.reverseImage) formData.append('reverseImage', input.reverseImage)
  for (const file of input.detailImages ?? []) {
    formData.append('detailImages', file)
  }
  return api.put<QuickCaptureDraft>(`/quick-capture/drafts/${id}`, formData)
}

export const discardQuickCaptureDraft = (id: number) =>
  api.post<QuickCaptureDraft>(`/quick-capture/drafts/${id}/discard`)

export const promoteQuickCaptureDraft = (id: number, request: QuickCapturePromoteRequest) =>
  api.post<QuickCapturePromotionResponse>(`/quick-capture/drafts/${id}/promote`, request)

export const commitIntakeDraft = (request: IntakeCommitRequest) =>
  api.post<IntakeCommitResponse>('/coins/intake/commit', {
    ...request,
    overrides: request.overrides ? sanitizeCoin(request.overrides) : undefined,
  })

export const updateCoin = (id: number, coin: CoinMutationPayload, params?: Record<string, string>) =>
  api.put<Coin>(`/coins/${id}`, sanitizeCoin(coin), { params })

export const purchaseCoin = (id: number, data?: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string; shipment?: ShipmentUpsertInput }) =>
  api.post<Coin>(`/coins/${id}/purchase`, data || {})

export const sellCoin = (id: number, soldPrice: number | null, soldTo: string) =>
  api.post<Coin>(`/coins/${id}/sell`, { soldPrice, soldTo })

export const duplicateCoin = (id: number) => api.post<Coin>(`/coins/${id}/duplicate`)

export const deleteCoin = (id: number) => api.delete(`/coins/${id}`)

export const getCoinReferences = (coinId: number) => api.get<CoinReference[]>(`/coins/${coinId}/references`)

export const createCoinReference = (coinId: number, reference: CoinReferenceInput) =>
  api.post<CoinReference>(`/coins/${coinId}/references`, reference)

export const updateCoinReference = (coinId: number, referenceId: number, reference: CoinReferenceInput) =>
  api.put<CoinReference>(`/coins/${coinId}/references/${referenceId}`, reference)

export const deleteCoinReference = (coinId: number, referenceId: number) =>
  api.delete(`/coins/${coinId}/references/${referenceId}`)

export const migrateLegacyReferences = () =>
  api.post<LegacyMigrationResult>('/references/migrate-legacy')

export const addTagToCoin = (coinId: number, tagId: number) => api.post(`/coins/${coinId}/tags`, { tagId })

export const removeTagFromCoin = (coinId: number, tagId: number) => api.delete(`/coins/${coinId}/tags/${tagId}`)

export const getCoinRecommendations = (coinId: number) =>
  api.get<{ recommendations: CoinRecommendation[] }>(`/coins/${coinId}/recommendations`)

export const acceptCoinRecommendation = (coinId: number, recommendationId: number) =>
  api.post(`/coins/${coinId}/recommendations/${recommendationId}/accept`)

export const rejectCoinRecommendation = (coinId: number, recommendationId: number) =>
  api.post(`/coins/${coinId}/recommendations/${recommendationId}/reject`)

// Catalog Registry
export const listCatalogs = async () => {
  const res = await api.get<{ catalogs: CatalogRegistry[] }>('/catalogs')
  return res.data.catalogs ?? []
}

// Bulk Operations
export const bulkAction = (
  coinIds: number[],
  action: string,
  opts?: { tagId?: number; setId?: number; storageLocationId?: number | null }
) => {
  const payload: { coinIds: number[]; action: string; tagId?: number; setId?: number; storageLocationId?: number | null } = {
    coinIds,
    action,
  }
  if (opts?.tagId !== undefined) payload.tagId = opts.tagId
  if (opts?.setId !== undefined) payload.setId = opts.setId
  if (opts?.storageLocationId !== undefined) payload.storageLocationId = opts.storageLocationId
  return api.post<{ message: string; affected: number; coins?: Coin[] }>('/coins/bulk', payload)
}

// Journal
export const getJournalEntries = (coinId: number) => api.get<CoinJournal[]>(`/coins/${coinId}/journal`)

export const addJournalEntry = (coinId: number, entry: string) =>
  api.post<CoinJournal>(`/coins/${coinId}/journal`, { entry })

export const deleteJournalEntry = (coinId: number, entryId: number) =>
  api.delete(`/coins/${coinId}/journal/${entryId}`)

// Numista
export const searchNumista = (q: string) => api.get<NumistaSearchResponse>('/numista/search', { params: { q } })

export const proposeNumistaQuery = (request: NumistaQueryProposalRequest) =>
  api.post<NumistaQueryProposal>('/numista/query-proposal', request)

export const lookupNumista = (request: NumistaLookupRequest) =>
  api.post<NumistaLookupOutcome>('/numista/lookup', request)

export const enrichNumista = (request: NumistaEnrichmentRequest, signal?: AbortSignal) =>
  api.post<NumistaLookupOutcome>('/numista/enrich', request, { signal })

// Coin Lookup
export async function lookupCoin(images: File[], notes = '', imageRoles: CoinLookupImageRole[] = []) {
  const formData = new FormData()
  for (const [index, image] of images.entries()) {
    formData.append('images', image)
    const role = imageRoles[index]
    if (role) formData.append('imageRoles', role)
  }
  if (notes.trim()) {
    formData.append('notes', notes.trim())
  }
  return api.post<CoinLookupResponse>('/coins/lookup', formData)
}

// Value Estimation
export const estimateCoinValue = (coinId: number) =>
  api.post<AIJobStartResponse>(`/coins/${coinId}/estimate-value`)

export const getCoinValueHistory = (coinId: number) =>
  api.get<CoinValueHistory[]>(`/coins/${coinId}/value-history`)

// Images
export const uploadImage = (coinId: number, file: File, imageType: string, isPrimary: boolean, circleClip?: boolean) => {
  const formData = new FormData()
  formData.append('image', file)
  formData.append('imageType', imageType)
  formData.append('isPrimary', String(isPrimary))
  if (circleClip) {
    formData.append('circleClip', 'true')
  }
  return api.post<CoinImage>(`/coins/${coinId}/images`, formData)
}

export const deleteImage = (coinId: number, imageId: number) =>
  api.delete(`/coins/${coinId}/images/${imageId}`)

// Analysis
export const analyzeCoin = (coinId: number, side?: 'obverse' | 'reverse') => {
  const params = side ? `?side=${side}` : ''
  return api.post<AIJobStartResponse>(`/coins/${coinId}/analyze${params}`)
}

export const gradeCoin = (coinId: number) =>
  api.post<AIJobStartResponse>(`/coins/${coinId}/grade`)

export const getAIJob = (id: string | number) =>
  api.get<AIJob>(`/ai-jobs/${id}`)

export const getCoinAIJobs = (coinId: number, activeOnly = false) =>
  api.get<AIJob[] | { jobs?: AIJob[] }>(`/coins/${coinId}/ai-jobs`, { params: { activeOnly } })

export const deleteAnalysis = (coinId: number, side: 'obverse' | 'reverse') =>
  api.delete<{ coin: Coin }>(`/coins/${coinId}/analyze?side=${side}`)

export const extractText = (file: File) => {
  const formData = new FormData()
  formData.append('image', file)
  return api.post<{ text: string }>('/extract-text', formData)
}

export const getValueHistory = () => api.get<ValueSnapshot[]>('/value-history')

export const getCoinHealthList = (params?: { scope?: 'all' | 'needs_attention'; page?: number; limit?: number }) =>
  api.get<CoinHealthListResponse>('/coins/health', { params })

export const getCoinHealth = (coinId: number) =>
  api.get<CoinHealthItem>(`/coins/${coinId}/health`)

// Autocomplete suggestions
export const getSuggestions = (field: string, q: string) =>
  api.get<string[]>('/suggestions', { params: { field, q } })

export const proxyImage = (url: string) =>
  api.get('/proxy-image', { params: { url }, responseType: 'blob' })

export const scrapeImage = (url: string) =>
  api.get<{ imageUrl: string }>('/scrape-image', { params: { url } })

export const getCoinShipment = (coinId: number) => api.get<ShipmentEnvelopeResponse>(`/coins/${coinId}/shipment`)

export const upsertCoinShipment = (coinId: number, input: ShipmentUpsertInput) => api.put<ShipmentEnvelopeResponse>(`/coins/${coinId}/shipment`, input)

export const deleteCoinShipment = (coinId: number) => api.delete<{ message: string }>(`/coins/${coinId}/shipment`)

export const setCoinShipmentManualOverride = (
  coinId: number,
  input: { enabled: boolean; status: ShipmentStatus; note?: string },
) => api.put<ShipmentEnvelopeResponse>(`/coins/${coinId}/shipment/manual-override`, input)

export const syncCoinShipment = (coinId: number) => api.post<ShipmentEnvelopeResponse>(`/coins/${coinId}/shipment/sync`)

export const updateListingStatus = (coinId: number, status: string) =>
  api.put(`/coins/${coinId}/listing-status`, { status })
