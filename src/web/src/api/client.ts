import axios from 'axios'
import type { Coin, CoinListResponse, CoinImage, AuthResponse, StatsResponse, UserInfo, AppSettings, LogEntry, ApiKey, WebAuthnCredentialInfo, ValueSnapshot, CoinJournal, NumistaSearchResponse, AgentChatMessage, AgentChatAppContext, CoinSuggestion, CollectionChatResponse, FollowUser, PublicProfile, CoinComment, CoinRating, LimitedCoin, CoinValueHistory, PortfolioSummary, AuctionLot, AuctionLotListResponse, AvailabilityRunSummary, AvailabilityRun, NotificationListResponse, Tag, StorageLocation, MintLocation, ValuationRun, AuctionEndingRun, AuctionWatchBidDigestRun, CollectionHealthSnapshotRunResult, CollectionHealthSnapshotRun, SchedulerStatus, CalendarEventDetail, FeaturedCoin, CollectionHealthSummary, CoinHealthListResponse, CoinHealthItem, AdminHealthSummaryResponse, CoinReference, CoinReferenceInput, CoinMutationPayload, IntakeDraft, IntakeCommitRequest, IntakeCommitResponse, CoinLookupResponse, LegacyMigrationResult, CatalogRegistry, CoinSetSummary, CoinSetDetail, CreateCoinSetRequest, UpdateCoinSetRequest, AddCoinToSetRequest, ReorderSetCoinsRequest, CoinSetTemplate, CoinSetCompletion, CreateCoinSetFromCsvRequest, CoinSetSnapshot, CoinSetAnalytics, CoinSetComparison, SmartCriteriaGroup, SmartSetPreview, SmartCriteriaTemplate, SuggestedSmartCriteria, UserNote, NoteInput, NoteListResponse, SecuritySummary, SecurityEventFilters, SecurityEventsResponse, SecurityIpRule, CreateSecurityIpRuleRequest, SecurityExposureCheck, InvestmentBreakdownDimension, InvestmentBreakdownResponse, OIDCPublicProvidersResponse, OIDCStartFlowRequest, OIDCStartFlowResponse, OIDCLinkCallbackResponse, OIDCLinkedIdentitiesResponse, OIDCMessageResponse, OIDCAdminProvidersResponse, OIDCAdminProvider, OIDCAdminProviderInput, OIDCAdminProviderUpdate, OIDCProviderTestResponse, AIJob, AIJobStartResponse, PriceAlert, BidReminder, PriceAlertDirection, AuctionAlertReminderRun, CoinOfDayRun, BidRecommendation, MarketSignal } from '@/types'
import type { QuickCaptureDraft, QuickCaptureDraftInput, QuickCaptureDraftUpdateInput, QuickCaptureDraftListResponse, QuickCaptureDraftStatus, QuickCapturePromoteRequest, QuickCapturePromotionResponse } from '@/types'
import type { WishlistSearchAlert, WishlistSearchAlertInput, WishlistSearchAlertListResponse, AlertRun, AlertRunListResponse, AlertRunResult, AlertCandidate, AlertCandidateListResponse, AlertCandidateState, CandidateProvenanceStatus, DismissWishlistSearchAlertCandidateInput, ConvertWishlistSearchAlertCandidateInput, ConvertWishlistSearchAlertCandidateResponse, AdjustWishlistSearchAlertCriteriaInput } from '@/types'

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

const api = axios.create({
  baseURL: `${API_BASE}/api`,
})

type ApiErrorPayload = {
  error?: unknown
  message?: unknown
  detail?: unknown
}

export function getApiErrorMessage(error: unknown): string {
  if (typeof error === 'string') return error

  if (typeof error === 'object' && error !== null) {
    const maybeResponse = error as { response?: { data?: ApiErrorPayload } }
    const data = maybeResponse.response?.data ?? (error as ApiErrorPayload)
    const candidate = data.error ?? data.message ?? data.detail
    if (typeof candidate === 'string') return candidate
  }

  if (error instanceof Error) return error.message

  return ''
}

export function formatAgentServiceError(error: unknown, fallback = 'Agent service unavailable. Check the internal agent service configuration.'): string {
  const message = getApiErrorMessage(error).trim()
  if (!message) return fallback

  if (/internal service credential is not configured/i.test(message)) {
    return 'Internal agent service credential is not configured. Check the internal agent service configuration.'
  }

  if (/agent service unavailable/i.test(message) || /^HTTP 503$/i.test(message)) {
    return 'Agent service unavailable. Check the internal agent service configuration.'
  }

  if (/check agent service configuration/i.test(message)) {
    return 'Check the agent service configuration and retry.'
  }

  return message
}

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let isRefreshing = false
let failedQueue: Array<{ resolve: (token: string) => void; reject: (err: unknown) => void }> = []

// Callback for syncing Pinia auth store after silent token refresh.
// Registered by the auth store to avoid circular imports.
let _onTokenRefreshed: ((data: AuthResponse) => void) | null = null
export function onTokenRefreshed(cb: (data: AuthResponse) => void) {
  _onTokenRefreshed = cb
}

function processQueue(error: unknown, token: string | null) {
  failedQueue.forEach((p) => {
    if (token) p.resolve(token)
    else p.reject(error)
  })
  failedQueue = []
}

export async function refreshAccessToken(): Promise<string> {
  const refreshToken = localStorage.getItem('refreshToken')
  if (!refreshToken) {
    clearAuth()
    throw new Error('Missing refresh token')
  }

  if (isRefreshing) {
    return new Promise((resolve, reject) => {
      failedQueue.push({ resolve, reject })
    })
  }

  isRefreshing = true
  try {
    const res = await axios.post<AuthResponse>(`${API_BASE}/api/auth/refresh`, { refreshToken })
    const { token, refreshToken: newRefresh, user } = res.data
    localStorage.setItem('token', token)
    localStorage.setItem('refreshToken', newRefresh)
    localStorage.setItem('user', JSON.stringify(user))
    _onTokenRefreshed?.(res.data)
    processQueue(null, token)
    return token
  } catch (refreshError) {
    processQueue(refreshError, null)
    clearAuth()
    throw refreshError
  } finally {
    isRefreshing = false
  }
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true

      try {
        const token = await refreshAccessToken()
        originalRequest.headers = {
          ...(originalRequest.headers ?? {}),
          Authorization: `Bearer ${token}`,
        }
        return api(originalRequest)
      } catch (refreshError: unknown) {
        if (refreshError instanceof Error && refreshError.message === 'Missing refresh token') {
          return Promise.reject(error)
        }
        return Promise.reject(refreshError)
      }
    }
    return Promise.reject(error)
  },
)

function clearAuth() {
  localStorage.removeItem('token')
  localStorage.removeItem('refreshToken')
  localStorage.removeItem('user')
  window.location.href = '/login'
}

// Auth
export const login = (username: string, password: string) =>
  api.post<AuthResponse>('/auth/login', { username, password })
export const register = (username: string, password: string, email?: string) =>
  api.post<AuthResponse>('/auth/register', { username, password, email })

// OIDC auth
export const getOIDCPublicProviders = () =>
  api.get<OIDCPublicProvidersResponse>('/auth/oidc/providers')
export const startOIDCLogin = (providerId: number, request: OIDCStartFlowRequest) =>
  api.post<OIDCStartFlowResponse>(`/auth/oidc/${providerId}/start`, request)
export const completeOIDCLoginCallback = (providerId: number, code: string, state: string) =>
  api.get<AuthResponse>(`/auth/oidc/${providerId}/callback`, { params: { code, state } })

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

const NULLABLE_FIELDS: (keyof Coin)[] = ['weightGrams', 'diameterMm', 'purchasePrice', 'currentValue', 'purchaseDate', 'storageLocationId']

function sanitizeCoin(coin: CoinMutationPayload): CoinMutationPayload {
  const clean: Record<string, unknown> = { ...coin }
  for (const field of NULLABLE_FIELDS) {
    if (clean[field] === '' || clean[field] === undefined) {
      clean[field] = null
    }
  }
  delete clean.storageLocation
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

function appendOptionalFormValue(formData: FormData, key: string, value?: string | null) {
  const trimmed = value?.trim()
  if (trimmed) formData.append(key, trimmed)
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
export const purchaseCoin = (id: number, data?: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string }) =>
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

// User notes
export const getNotes = () => api.get<NoteListResponse>('/notes')
export const createNote = (note: NoteInput) => api.post<UserNote>('/notes', note)
export const updateNote = (id: number, note: NoteInput) => api.put<UserNote>(`/notes/${id}`, note)
export const deleteNote = (id: number) => api.delete(`/notes/${id}`)

// Tags
export const getTags = () => api.get<{ tags: Tag[] }>('/tags')
export const createTag = (data: { name: string; color?: string }) => api.post<Tag>('/tags', data)
export const updateTag = (id: number, data: { name?: string; color?: string }) => api.put<Tag>(`/tags/${id}`, data)
export const deleteTag = (id: number) => api.delete(`/tags/${id}`)
export const addTagToCoin = (coinId: number, tagId: number) => api.post(`/coins/${coinId}/tags`, { tagId })
export const removeTagFromCoin = (coinId: number, tagId: number) => api.delete(`/coins/${coinId}/tags/${tagId}`)

// Storage Locations
export const getStorageLocations = () => api.get<{ storageLocations: StorageLocation[] }>('/storage-locations')
export const createStorageLocation = (data: { name: string; sortOrder?: number }) => api.post<StorageLocation>('/storage-locations', data)
export const updateStorageLocation = (id: number, data: { name?: string; sortOrder?: number }) => api.put<StorageLocation>(`/storage-locations/${id}`, data)
export const deleteStorageLocation = (id: number) => api.delete(`/storage-locations/${id}`)

export type MintLocationInput = {
  displayName: string
  lat: number
  lng: number
  region?: string
  aliases: string[]
}

export type MintLocationsResponse = MintLocation[] | { mintLocations?: MintLocation[] }

// Mint Locations
export const getMintLocations = () => api.get<MintLocationsResponse>('/mint-locations')
export const adminCreateMintLocation = (data: MintLocationInput) =>
  api.post<MintLocation>('/admin/mint-locations', data)
export const adminUpdateMintLocation = (id: number, data: MintLocationInput) =>
  api.put<MintLocation>(`/admin/mint-locations/${id}`, data)
export const adminDeleteMintLocation = (id: number) => api.delete(`/admin/mint-locations/${id}`)

// Catalog Registry
export const listCatalogs = async () => {
  const res = await api.get<{ catalogs: CatalogRegistry[] }>('/catalogs')
  return res.data.catalogs ?? []
}
export const adminCreateCatalog = (payload: { catalog: string; displayName: string; era: string; volumeRequired: boolean }) =>
  api.post<CatalogRegistry>('/admin/catalogs', payload)
export const adminUpdateCatalog = (id: number, payload: { catalog: string; displayName: string; era: string; volumeRequired: boolean }) =>
  api.put<CatalogRegistry>(`/admin/catalogs/${id}`, payload)
export const adminDeleteCatalog = (id: number) => api.delete(`/admin/catalogs/${id}`)

// Sets
export const getSets = () => api.get<{ sets: CoinSetSummary[] }>('/sets')
export const getSet = (id: number) => api.get<CoinSetDetail>(`/sets/${id}`)
export const createSet = (data: CreateCoinSetRequest) => api.post<CoinSetDetail>('/sets', data)
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

// Coin Lookup
export async function lookupCoin(images: File[]) {
  const formData = new FormData()
  for (const image of images) {
    formData.append('images', image)
  }
  return api.post<CoinLookupResponse>('/coins/lookup', formData)
}

// Value Estimation
export const estimateCoinValue = (coinId: number) =>
  api.post<AIJobStartResponse>(`/coins/${coinId}/estimate-value`)

export const getCoinValueHistory = (coinId: number) =>
  api.get<CoinValueHistory[]>(`/coins/${coinId}/value-history`)

export const getValuationPrompt = () => api.get<{ prompt: string; default: string }>('/agent/valuation-prompt')

export const getPortfolioSummary = () => api.get<PortfolioSummary>('/agent/portfolio-summary')

// Agent

export async function agentChatStream(
  message: string,
  history: AgentChatMessage[],
  onText: (text: string) => void,
  onDone: (message: string, suggestions: CoinSuggestion[], collection?: CollectionChatResponse) => void,
  onError: (error: string) => void,
  onStatus?: (status: string) => void,
  appContext?: AgentChatAppContext,
) {
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''

  async function fetchWithAuthRetry(url: string, init: RequestInit): Promise<Response> {
    const firstHeaders = new Headers(init.headers ?? {})
    const token = localStorage.getItem('token')
    if (token) {
      firstHeaders.set('Authorization', `Bearer ${token}`)
    }

    const firstResp = await fetch(url, { ...init, headers: firstHeaders })
    if (firstResp.status !== 401) {
      return firstResp
    }

    const refreshedToken = await refreshAccessToken()
    const retryHeaders = new Headers(init.headers ?? {})
    retryHeaders.set('Authorization', `Bearer ${refreshedToken}`)
    return fetch(url, { ...init, headers: retryHeaders })
  }

  try {
    const resp = await fetchWithAuthRetry(`${baseURL}/api/agent/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ message, history, appContext }),
    })

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: `HTTP ${resp.status}` }))
      onError(formatAgentServiceError(err, `HTTP ${resp.status}`))
      return
    }

    const reader = resp.body?.getReader()
    if (!reader) { onError('No response body'); return }

    const decoder = new TextDecoder()
    let buffer = ''
    let accumulatedText = ''
    let terminalSent = false

    const sendDone = (
      finalMessage?: string,
      suggestions?: CoinSuggestion[],
      collection?: CollectionChatResponse,
    ) => {
      if (terminalSent) return
      terminalSent = true
      onDone(finalMessage || accumulatedText, Array.isArray(suggestions) ? suggestions : [], collection)
    }

    const sendError = (message: string) => {
      if (terminalSent) return
      terminalSent = true
      onError(message)
    }

    const handleDataLine = (line: string) => {
      if (!line.startsWith('data:')) return
      const data = line.replace(/^data:\s*/, '').trim()
      if (!data || data === '[DONE]') return

      try {
        const event = JSON.parse(data)
        if (event.type === 'text' && typeof event.text === 'string') {
          accumulatedText += event.text
          onText(event.text)
        } else if (event.type === 'status' && typeof event.message === 'string') {
          onStatus?.(event.message)
        } else if (event.type === 'done') {
          sendDone(
            typeof event.message === 'string' ? event.message : undefined,
            event.suggestions,
            event.collection,
          )
        } else if (event.type === 'error') {
          sendError(formatAgentServiceError(
            typeof event.message === 'string' ? event.message : '',
            'Agent stream error',
          ))
        }
      } catch {
        // Ignore malformed stream chunks.
      }
    }

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        handleDataLine(line)
      }
    }

    buffer += decoder.decode()
    if (buffer.trim()) {
      handleDataLine(buffer.trim())
    }

    if (!terminalSent) {
      if (accumulatedText.trim()) {
        sendDone(accumulatedText, [])
      } else {
        sendError('Stream ended unexpectedly')
      }
    }
  } catch (err: unknown) {
    onError(err instanceof Error ? err.message : 'Stream failed')
  }
}

export const commitCollectionProposal = (proposalId: string, proposalToken: string) =>
  api.post(`/agent/collection/proposals/${proposalId}/commit`, {
    proposalToken,
    confirm: true,
  })

export const cancelCollectionProposal = (proposalId: string) =>
  api.post(`/agent/collection/proposals/${proposalId}/cancel`, {})

export interface AnthropicModel {
  id: string
  name: string
}

export const getAnthropicModels = () => api.get<AnthropicModel[]>('/agent/models')

export const getCoinSearchPrompt = () => api.get<{ prompt: string; default: string }>('/agent/coin-search-prompt')
export const getCoinShowsPrompt = () => api.get<{ prompt: string; default: string }>('/agent/coin-shows-prompt')

// Agent Conversations
export interface ConversationSummary {
  id: number
  title: string
  createdAt: string
  updatedAt: string
}

export interface SavedConversation {
  id: number
  userId: number
  title: string
  messages: string
  createdAt: string
  updatedAt: string
}

export const listConversations = () => api.get<ConversationSummary[]>('/agent/conversations')
export const getConversation = (id: number) => api.get<SavedConversation>(`/agent/conversations/${id}`)
export const saveConversation = (data: { id?: number; title: string; messages: string }) =>
  api.post<SavedConversation>('/agent/conversations', data)
export const deleteConversation = (id: number) => api.delete(`/agent/conversations/${id}`)

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
export const getOllamaStatus = () =>
  api.get<{ available: boolean; model: string; url: string; message: string }>('/ollama-status')
export const getAIStatus = () =>
  api.get<{ available: boolean; provider: string; model: string; message: string }>('/ai-status')

// Stats
export const getStats = () => api.get<StatsResponse>('/stats')
export const getDistribution = () => api.get<{ cells: { era: string; category: string; count: number }[] }>('/stats/distribution')
export const getInvestmentBreakdown = (dimension: InvestmentBreakdownDimension) =>
  api.get<InvestmentBreakdownResponse>('/stats/investment-breakdown', { params: { dimension } })
export const getValueHistory = () => api.get<ValueSnapshot[]>('/value-history')
export const getCollectionHealthSummary = () => api.get<CollectionHealthSummary>('/stats/health')
export const getCoinHealthList = (params?: { scope?: 'all' | 'needs_attention'; page?: number; limit?: number }) =>
  api.get<CoinHealthListResponse>('/coins/health', { params })

export const getCoinHealth = (coinId: number) =>
  api.get<CoinHealthItem>(`/coins/${coinId}/health`)

// Autocomplete suggestions
export const getSuggestions = (field: string, q: string) =>
  api.get<string[]>('/suggestions', { params: { field, q } })

// User self-service
export const getMe = () => api.get<UserInfo>('/auth/me')
export const changePassword = (currentPassword: string, newPassword: string) =>
  api.post('/auth/change-password', { currentPassword, newPassword })
export const exportCollection = () => api.get('/user/export', { responseType: 'blob' })
export const exportCatalogPDF = () => api.get('/user/export/catalog', { responseType: 'blob' })
export const proxyImage = (url: string) =>
  api.get('/proxy-image', { params: { url }, responseType: 'blob' })
export const scrapeImage = (url: string) =>
  api.get<{ imageUrl: string }>('/scrape-image', { params: { url } })
export const importCollection = (coins: Partial<Coin>[]) => api.post('/user/import', coins)

// OIDC account linking
export const startOIDCLink = (providerId: number, request: OIDCStartFlowRequest) =>
  api.post<OIDCStartFlowResponse>(`/auth/oidc/${providerId}/link/start`, request)
export const completeOIDCLinkCallback = (providerId: number, code: string, state: string) =>
  api.get<OIDCLinkCallbackResponse>(`/auth/oidc/${providerId}/link/callback`, { params: { code, state } })
export const getOIDCIdentities = () =>
  api.get<OIDCLinkedIdentitiesResponse>('/user/oidc-identities')
export const deleteOIDCIdentity = (identityId: number) =>
  api.delete<OIDCMessageResponse>(`/user/oidc-identities/${identityId}`)

// API Keys
export const generateApiKey = (name: string, scope?: 'read' | 'read,write') =>
  api.post<{ key: string; apiKey: ApiKey }>('/auth/api-keys', { name, scope })
export const listApiKeys = () => api.get<ApiKey[]>('/auth/api-keys')
export const revokeApiKey = (id: number) => api.delete(`/auth/api-keys/${id}`)

// Admin
export const getUsers = () => api.get<UserInfo[]>('/admin/users')
export const deleteUser = (id: number) => api.delete(`/admin/users/${id}`)
export const resetUserPassword = (id: number, newPassword: string) =>
  api.post(`/admin/users/${id}/reset-password`, { newPassword })
export const updateUserRole = (id: number, role: UserInfo['role']) =>
  api.put(`/admin/users/${id}/role`, { role })
export const unlockUser = (id: number) => api.post(`/admin/users/${id}/unlock`)
export const getAppSettings = () => api.get<AppSettings>('/admin/settings')
export const getAppSettingDefaults = () => api.get<AppSettings>('/admin/settings/defaults')
export const updateAppSettings = (settings: { key: string; value: string }[]) =>
  api.put('/admin/settings', settings)
export const getSecuritySummary = () =>
  api.get<SecuritySummary | { summary?: Partial<SecuritySummary>; backupStatus?: string }>('/admin/security/summary')
export const getSecurityEvents = (filters?: SecurityEventFilters) => {
  const params = filters ? { ...filters } : undefined
  if (params?.ip && !params.clientIp) {
    params.clientIp = params.ip
    delete params.ip
  }
  return api.get<SecurityEventsResponse>('/admin/security/events', { params })
}
export const getSecurityIpRules = () =>
  api.get<{ rules?: SecurityIpRule[]; ipRules?: SecurityIpRule[] } | SecurityIpRule[]>('/admin/security/ip-rules')
export const createSecurityIpRule = (payload: CreateSecurityIpRuleRequest) => {
  const body: { cidr: string; reason: string; durationMinutes?: number; expiresAt?: string } = {
    cidr: payload.cidr,
    reason: payload.reason,
  }
  const durationMinutes = payload.durationMinutes ?? parseDurationMinutes(payload.duration)
  if (durationMinutes) body.durationMinutes = durationMinutes
  if (payload.expiresAt) body.expiresAt = payload.expiresAt
  return api.post<SecurityIpRule>('/admin/security/ip-rules', body)
}
export const deleteSecurityIpRule = (id: number) =>
  api.delete(`/admin/security/ip-rules/${id}`)
export const getSecurityExposureCheck = () =>
  api.get<SecurityExposureCheck>('/admin/security/exposure-check')
export const getAdminLogs = (limit = 500, level?: string) => {
  const params: Record<string, string> = { limit: String(limit) }
  if (level) params.level = level
  return api.get<{ logs: LogEntry[]; count: number; logLevel: string }>('/admin/logs', { params })
}

// Admin OIDC providers
export const getAdminOIDCProviders = () =>
  api.get<OIDCAdminProvidersResponse>('/admin/oidc/providers')
export const createAdminOIDCProvider = (provider: OIDCAdminProviderInput) =>
  api.post<OIDCAdminProvider>('/admin/oidc/providers', provider)
export const updateAdminOIDCProvider = (providerId: number, provider: OIDCAdminProviderUpdate) =>
  api.put<OIDCAdminProvider>(`/admin/oidc/providers/${providerId}`, provider)
export const deleteAdminOIDCProvider = (providerId: number) =>
  api.delete(`/admin/oidc/providers/${providerId}`)
export const testAdminOIDCProvider = (providerId: number) =>
  api.post<OIDCProviderTestResponse>(`/admin/oidc/providers/${providerId}/test`)

type ConnectivityResult = { available: boolean; message: string }
export const testAnthropicConnection = () =>
  api.get<ConnectivityResult>('/admin/test-anthropic')
export const testSearXNGConnection = () =>
  api.get<ConnectivityResult>('/admin/test-searxng')
export const getAdminHealthSummary = () =>
  api.get<AdminHealthSummaryResponse>('/admin/health/summary')

// Agent status
export const getAgentStatus = () =>
  api.get<{ provider: string; configured: boolean }>('/agent/status')

// WebAuthn
export const webauthnRegisterBegin = () =>
  api.post('/auth/webauthn/register/begin')
export const webauthnRegisterFinish = (credential: PublicKeyCredential) => {
  const attestation = credential.response as AuthenticatorAttestationResponse
  const body = {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    authenticatorAttachment: credential.authenticatorAttachment || undefined,
    response: {
      attestationObject: bufferToBase64url(attestation.attestationObject),
      clientDataJSON: bufferToBase64url(attestation.clientDataJSON),
      transports: attestation.getTransports ? attestation.getTransports() : undefined,
    },
  }
  return api.post('/auth/webauthn/register/finish', body)
}
export const webauthnLoginBegin = (username: string) =>
  api.post<{ options: PublicKeyCredentialRequestOptionsJSON; username: string }>('/auth/webauthn/login/begin', { username })
export const webauthnLoginFinish = (username: string, credential: PublicKeyCredential) => {
  const assertion = credential.response as AuthenticatorAssertionResponse
  const body = {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      authenticatorData: bufferToBase64url(assertion.authenticatorData),
      clientDataJSON: bufferToBase64url(assertion.clientDataJSON),
      signature: bufferToBase64url(assertion.signature),
      userHandle: assertion.userHandle ? bufferToBase64url(assertion.userHandle) : null,
    },
  }
  return api.post<AuthResponse>(`/auth/webauthn/login/finish?username=${encodeURIComponent(username)}`, body)
}
export const webauthnCheck = (username: string) =>
  api.get<{ available: boolean }>('/auth/webauthn/check', { params: { username } })
export const webauthnListCredentials = () =>
  api.get<WebAuthnCredentialInfo[]>('/auth/webauthn/credentials')
export const webauthnDeleteCredential = (id: number) =>
  api.delete(`/auth/webauthn/credentials/${id}`)

function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  bytes.forEach((b) => (binary += String.fromCharCode(b)))
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

// Helper types for WebAuthn JSON format
interface PublicKeyCredentialRequestOptionsJSON {
  publicKey?: PublicKeyCredentialRequestOptionsPublicKeyJSON
  challenge?: string
  timeout?: number
  rpId?: string
  allowCredentials?: Array<{ id: string; type: string; transports?: string[] }>
  userVerification?: string
}

interface PublicKeyCredentialRequestOptionsPublicKeyJSON {
  challenge: string
  timeout?: number
  rpId?: string
  allowCredentials?: Array<{ id: string; type: string; transports?: string[] }>
  userVerification?: string
}

// --- Social / Profile API ---

// Profile
export const updateProfile = (data: { email?: string; bio?: string; zipCode?: string; isPublic?: boolean; numisBidsUsername?: string; numisBidsPassword?: string; cngUsername?: string; cngPassword?: string; pushoverUserKey?: string; coinOfDayEnabled?: boolean; emperorTrackerEnabled?: boolean; emperorTrackerShowUsurpers?: boolean; emperorTrackerShowEmpresses?: boolean; emperorTrackerShowOtherFigures?: boolean }) =>
  api.put<{ id: number; username: string; role: string; email: string; avatarPath: string; isPublic: boolean; bio: string; zipCode: string; numisBidsUsername: string; numisBidsConfigured: boolean; cngUsername: string; cngConfigured: boolean; pushoverEnabled: boolean; coinOfDayEnabled: boolean; emperorTrackerEnabled: boolean; emperorTrackerShowUsurpers: boolean; emperorTrackerShowEmpresses: boolean; emperorTrackerShowOtherFigures: boolean }>('/user/profile', data)
export const uploadAvatar = (file: File) => {
  const form = new FormData()
  form.append('avatar', file)
  return api.post<{ avatarPath: string }>('/user/avatar', form)
}
export const deleteAvatar = () => api.delete('/user/avatar')

// Follow
export const followUser = (userId: number) => api.post(`/social/follow/${userId}`)
export const unfollowUser = (userId: number) => api.delete(`/social/follow/${userId}`)
export const acceptFollower = (userId: number) => api.put(`/social/followers/${userId}/accept`)
export const blockFollower = (userId: number) => api.put(`/social/followers/${userId}/block`)

// Showcases
export const listShowcases = () => api.get('/showcases')
export const getShowcase = (id: number) => api.get(`/showcases/${id}`)
export const createShowcase = (data: { title: string; description?: string }) => api.post('/showcases', data)
export const updateShowcase = (id: number, data: { title?: string; description?: string; isActive?: boolean }) => api.put(`/showcases/${id}`, data)
export const deleteShowcase = (id: number) => api.delete(`/showcases/${id}`)
export const setShowcaseCoins = (id: number, coinIds: number[]) => api.put(`/showcases/${id}/coins`, { coinIds })
export const getPublicShowcase = (slug: string) => axios.get(`${API_BASE}/api/showcase/${slug}`)

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

export const unblockFollower = (userId: number) => api.delete(`/social/followers/${userId}/block`)
export const getFollowers = () => api.get<{ followers: FollowUser[] }>('/social/followers')
export const getFollowing = () => api.get<{ following: FollowUser[] }>('/social/following')
export const getBlockedUsers = () => api.get<{ blocked: { id: number; username: string; avatarPath: string }[] }>('/social/blocked')

// User discovery
export const searchUsers = (query: string) => api.get<{ users: FollowUser[] }>('/users/search', { params: { q: query } })
export const getPublicProfile = (username: string) => api.get<PublicProfile>(`/users/${encodeURIComponent(username)}`)

// Follower coins
export const getFollowingCoins = (userId: number) =>
  api.get<{ coins: LimitedCoin[]; username: string }>(`/social/following/${userId}/coins`)
export const getFollowingCoinDetail = (userId: number, coinId: number) =>
  api.get<LimitedCoin & { comments: CoinComment[]; rating: CoinRating }>(`/social/following/${userId}/coins/${coinId}`)

// Comments & ratings
export const addComment = (coinId: number, comment: string, rating?: number) =>
  api.post<CoinComment>(`/social/coins/${coinId}/comments`, { comment, rating: rating || 0 })
export const deleteComment = (coinId: number, commentId: number) =>
  api.delete(`/social/coins/${coinId}/comments/${commentId}`)
export const rateCoin = (coinId: number, rating: number) =>
  api.put<CoinRating>(`/social/coins/${coinId}/rating`, { rating })

// Auction lots
export const getAuctionLots = (params?: { status?: string; search?: string; source?: string; sort?: string; order?: string; page?: number; limit?: number }) =>
  api.get<AuctionLotListResponse>('/auctions', { params })
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
export const convertAuctionLotToCoin = (id: number) => api.post<Coin>(`/auctions/${id}/convert`)
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

// Pushover notifications
export const testPushover = () =>
  api.post<{ message: string }>('/notifications/test-pushover')

// Availability checks
export const checkWishlistAvailability = () =>
  api.post<AvailabilityRunSummary>('/wishlist/check-availability')
export const updateListingStatus = (coinId: number, status: string) =>
  api.put(`/coins/${coinId}/listing-status`, { status })
export const getAvailabilityRuns = (page = 1, limit = 20) =>
  api.get<{ runs: AvailabilityRun[]; total: number }>('/admin/availability-runs', { params: { page, limit } })
export const getAvailabilityRunDetail = (runId: number) =>
  api.get<AvailabilityRun>(`/admin/availability-runs/${runId}`)
export const triggerAvailabilityCheck = () =>
  api.post<{ runId: number; status: string; message: string }>('/admin/availability/run')

// Valuation Runs
export const getValuationRuns = (page = 1, limit = 20) =>
  api.get<{ runs: ValuationRun[]; total: number }>('/admin/valuation-runs', { params: { page, limit } })
export const getValuationRunDetail = (runId: number) =>
  api.get<ValuationRun>(`/admin/valuation-runs/${runId}`)
export const triggerValuation = () =>
  api.post<{ message: string; users: number }>('/admin/valuation-runs/trigger')
export const cancelValuationRun = (runId: number) =>
  api.post<{ message: string }>(`/admin/valuation-runs/${runId}/cancel`)

// Auction Ending Runs
export const getAuctionEndingRuns = (page = 1, limit = 20) =>
  api.get<{ runs: AuctionEndingRun[]; total: number; page: number; limit: number }>('/admin/auction-ending-runs', { params: { page, limit } })
export const getAuctionEndingRun = (id: number) =>
  api.get<AuctionEndingRun>(`/admin/auction-ending-runs/${id}`)
export const triggerAuctionEndingCheck = () =>
  api.post<{ runId: number; status: string }>('/admin/auction-ending/run')

// Auction Alert and Reminder Runs
export const getAuctionAlertReminderRuns = (page = 1, limit = 20) =>
  api.get<{ runs: AuctionAlertReminderRun[]; total: number; page: number; limit: number }>('/admin/auction-alert-runs', { params: { page, limit } })
export const triggerAuctionAlertReminderCheck = () =>
  api.post<{ message?: string; runId?: number; lotsChecked?: number; alertsTriggered?: number; alertsSent?: number; priceAlertsTriggered?: number; remindersNotified?: number; remindersSent?: number; bidRemindersSent?: number; status?: string; durationMs?: number }>('/admin/auction-alerts/run')

// Auction Watch Bid Digest Runs
export const getAuctionWatchBidDigestRuns = (page = 1, limit = 20) =>
  api.get<{ runs: AuctionWatchBidDigestRun[]; total: number; page: number; limit: number }>('/admin/auction-watch-bid-digest-runs', { params: { page, limit } })
export const triggerAuctionWatchBidDigest = () =>
  api.post<{ message: string }>('/admin/auction-watch-bid-digest/run')

// Collection Health Snapshots
export const triggerCollectionHealthSnapshots = () =>
  api.post<CollectionHealthSnapshotRunResult>('/admin/collection-health-snapshots/run')
export const getCollectionHealthSnapshotRuns = (page = 1, limit = 20) =>
  api.get<{ runs: CollectionHealthSnapshotRun[]; total: number; page: number; limit: number }>('/admin/collection-health-snapshot-runs', { params: { page, limit } })
export const getCollectionHealthSnapshotStatus = () =>
  api.get<SchedulerStatus>('/admin/collection-health/status')

// Coin of the Day
export const getLatestFeaturedCoin = () =>
  api.get<FeaturedCoin>('/featured-coins/latest')
export const getFeaturedCoin = (id: number) =>
  api.get<FeaturedCoin>(`/featured-coins/${id}`)
export const triggerCoinOfDayRun = () =>
  api.post<{ runId: number; status: string }>('/admin/coin-of-day/run')
export const getCoinOfDayRuns = (page = 1, limit = 20) =>
  api.get<{ runs: CoinOfDayRun[]; total: number; page: number; limit: number }>('/admin/coin-of-day-runs', { params: { page, limit } })
export const getCoinOfDayRunDetail = (runId: number) =>
  api.get<CoinOfDayRun>(`/admin/coin-of-day-runs/${runId}`)

// Notifications
export const getNotifications = (page = 1, limit = 20) =>
  api.get<NotificationListResponse>('/notifications', { params: { page, limit } })
export const getUnreadNotificationCount = () =>
  api.get<{ count: number }>('/notifications/unread-count')
export const markNotificationRead = (id: number) =>
  api.put(`/notifications/${id}/read`)
export const markAllNotificationsRead = () =>
  api.put('/notifications/read-all')
export const deleteNotification = (id: number) =>
  api.delete(`/notifications/${id}`)

export default api

function parseDurationMinutes(duration: string | undefined) {
  const value = duration?.trim()
  if (!value) return undefined
  const match = value.match(/^(\d+)\s*([mhdw])?$/i)
  if (!match) return undefined
  const amount = Number(match[1] ?? 0)
  const unit = (match[2] ?? 'm').toLowerCase()
  const multipliers: Record<string, number> = { m: 1, h: 60, d: 1440, w: 10080 }
  return amount * (multipliers[unit] ?? 1)
}
