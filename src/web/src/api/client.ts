import axios from 'axios'
import type { Coin, CoinListResponse, CoinImage, AuthResponse, StatsResponse, UserInfo, AppSettings, LogEntry, ApiKey, WebAuthnCredentialInfo, ValueSnapshot, CoinJournal, NumistaSearchResponse, AgentChatMessage, AgentChatResponse, CoinSuggestion, FollowUser, PublicProfile, CoinComment, CoinRating, LimitedCoin, ValueEstimate, CoinValueHistory, PortfolioSummary, AuctionLot, AuctionLotListResponse, AvailabilityRunSummary, AvailabilityRun, Notification, NotificationListResponse, Tag, ValuationRun, CalendarEventDetail } from '@/types'

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

const api = axios.create({
  baseURL: `${API_BASE}/api`,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let isRefreshing = false
let failedQueue: Array<{ resolve: (token: string) => void; reject: (err: unknown) => void }> = []

function processQueue(error: unknown, token: string | null) {
  failedQueue.forEach((p) => {
    if (token) p.resolve(token)
    else p.reject(error)
  })
  failedQueue = []
}

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      const refreshToken = localStorage.getItem('refreshToken')
      if (!refreshToken) {
        clearAuth()
        return Promise.reject(error)
      }

      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({
            resolve: (token: string) => {
              originalRequest.headers.Authorization = `Bearer ${token}`
              resolve(api(originalRequest))
            },
            reject,
          })
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      try {
        const res = await axios.post<AuthResponse>(`${API_BASE}/api/auth/refresh`, { refreshToken })
        const { token, refreshToken: newRefresh, user } = res.data
        localStorage.setItem('token', token)
        localStorage.setItem('refreshToken', newRefresh)
        localStorage.setItem('user', JSON.stringify(user))
        processQueue(null, token)
        originalRequest.headers.Authorization = `Bearer ${token}`
        return api(originalRequest)
      } catch (refreshError) {
        processQueue(refreshError, null)
        clearAuth()
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
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

// Coins
export const getCoins = (params?: {
  category?: string
  search?: string
  wishlist?: string
  sold?: string
  page?: number
  limit?: number
  sort?: string
  order?: string
}) => api.get<CoinListResponse>('/coins', { params })

const NULLABLE_FIELDS = ['weightGrams', 'diameterMm', 'purchasePrice', 'currentValue', 'purchaseDate']

function sanitizeCoin(coin: Partial<Coin>): Partial<Coin> {
  const clean = { ...coin }
  for (const field of NULLABLE_FIELDS) {
    if ((clean as any)[field] === '' || (clean as any)[field] === undefined) {
      ;(clean as any)[field] = null
    }
  }
  // Default currentValue to purchasePrice if not set
  if (!clean.currentValue && clean.purchasePrice) {
    clean.currentValue = clean.purchasePrice
  }
  // Convert date-only strings (YYYY-MM-DD) to RFC3339 for Go
  if (typeof clean.purchaseDate === 'string' && /^\d{4}-\d{2}-\d{2}$/.test(clean.purchaseDate)) {
    clean.purchaseDate = clean.purchaseDate + 'T00:00:00Z'
  }
  return clean
}

export const getCoin = (id: number) => api.get<Coin>(`/coins/${id}`)
export const createCoin = (coin: Partial<Coin>) => api.post<Coin>('/coins', sanitizeCoin(coin))
export const updateCoin = (id: number, coin: Partial<Coin>, params?: Record<string, string>) =>
  api.put<Coin>(`/coins/${id}`, sanitizeCoin(coin), { params })
export const purchaseCoin = (id: number, data?: { purchasePrice?: number; purchaseDate?: string; purchaseLocation?: string }) =>
  api.post<Coin>(`/coins/${id}/purchase`, data || {})
export const sellCoin = (id: number, soldPrice: number | null, soldTo: string) =>
  api.post<Coin>(`/coins/${id}/sell`, { soldPrice, soldTo })
export const deleteCoin = (id: number) => api.delete(`/coins/${id}`)

// Tags
export const getTags = () => api.get<{ tags: Tag[] }>('/tags')
export const createTag = (data: { name: string; color?: string }) => api.post<Tag>('/tags', data)
export const updateTag = (id: number, data: { name?: string; color?: string }) => api.put<Tag>(`/tags/${id}`, data)
export const deleteTag = (id: number) => api.delete(`/tags/${id}`)
export const addTagToCoin = (coinId: number, tagId: number) => api.post(`/coins/${coinId}/tags`, { tagId })
export const removeTagFromCoin = (coinId: number, tagId: number) => api.delete(`/coins/${coinId}/tags/${tagId}`)

// Bulk Operations
export const bulkAction = (coinIds: number[], action: string, tagId?: number) =>
  api.post<{ message: string; affected: number; coins?: Coin[] }>('/coins/bulk', { coinIds, action, tagId })

// Journal
export const getJournalEntries = (coinId: number) => api.get<CoinJournal[]>(`/coins/${coinId}/journal`)
export const addJournalEntry = (coinId: number, entry: string) =>
  api.post<CoinJournal>(`/coins/${coinId}/journal`, { entry })
export const deleteJournalEntry = (coinId: number, entryId: number) =>
  api.delete(`/coins/${coinId}/journal/${entryId}`)

// Numista
export const searchNumista = (q: string) => api.get<NumistaSearchResponse>('/numista/search', { params: { q } })

// Value Estimation
export const estimateCoinValue = (coinId: number) =>
  api.post<ValueEstimate>(`/coins/${coinId}/estimate-value`)

export const getCoinValueHistory = (coinId: number) =>
  api.get<CoinValueHistory[]>(`/coins/${coinId}/value-history`)

export const getValuationPrompt = () => api.get<{ prompt: string; default: string }>('/agent/valuation-prompt')

export const getPortfolioSummary = () => api.get<PortfolioSummary>('/agent/portfolio-summary')

// Agent

export async function agentChatStream(
  message: string,
  history: AgentChatMessage[],
  onText: (text: string) => void,
  onDone: (message: string, suggestions: CoinSuggestion[]) => void,
  onError: (error: string) => void,
  onStatus?: (status: string) => void,
) {
  const token = localStorage.getItem('token')
  const baseURL = import.meta.env.VITE_API_BASE_URL || ''
  try {
    const resp = await fetch(`${baseURL}/api/agent/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ message, history }),
    })

    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: `HTTP ${resp.status}` }))
      onError(err.error || `HTTP ${resp.status}`)
      return
    }

    const reader = resp.body?.getReader()
    if (!reader) { onError('No response body'); return }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })

      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6).trim()
        if (!data || data === '[DONE]') continue

        try {
          const event = JSON.parse(data)
          if (event.type === 'text') {
            onText(event.text)
          } else if (event.type === 'status') {
            onStatus?.(event.message)
          } else if (event.type === 'done') {
            onDone(event.message, event.suggestions || [])
          }
        } catch { /* skip malformed events */ }
      }
    }
  } catch (err: unknown) {
    onError(err instanceof Error ? err.message : 'Stream failed')
  }
}

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
export const uploadImage = (coinId: number, file: File, imageType: string, isPrimary: boolean) => {
  const formData = new FormData()
  formData.append('image', file)
  formData.append('imageType', imageType)
  formData.append('isPrimary', String(isPrimary))
  return api.post<CoinImage>(`/coins/${coinId}/images`, formData)
}
export const deleteImage = (coinId: number, imageId: number) =>
  api.delete(`/coins/${coinId}/images/${imageId}`)

// Analysis
export const analyzeCoin = (coinId: number, side?: 'obverse' | 'reverse') => {
  const params = side ? `?side=${side}` : ''
  return api.post<{ analysis: string; coin: Coin }>(`/coins/${coinId}/analyze${params}`)
}
export const deleteAnalysis = (coinId: number, side: 'obverse' | 'reverse') =>
  api.delete<{ coin: Coin }>(`/coins/${coinId}/analyze?side=${side}`)
export const extractText = (file: File) => {
  const formData = new FormData()
  formData.append('image', file)
  return api.post<{ text: string }>('/extract-text', formData)
}
export const getOllamaStatus = () =>
  api.get<{ available: boolean; model: string; url: string; message: string }>('/ollama-status')

// Stats
export const getStats = () => api.get<StatsResponse>('/stats')
export const getDistribution = () => api.get<{ cells: { era: string; category: string; count: number }[] }>('/stats/distribution')
export const getValueHistory = () => api.get<ValueSnapshot[]>('/value-history')

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
export const importCollection = (coins: Coin[]) => api.post('/user/import', coins)

// API Keys
export const generateApiKey = (name: string) =>
  api.post<{ key: string; apiKey: ApiKey }>('/auth/api-keys', { name })
export const listApiKeys = () => api.get<ApiKey[]>('/auth/api-keys')
export const revokeApiKey = (id: number) => api.delete(`/auth/api-keys/${id}`)

// Admin
export const getUsers = () => api.get<UserInfo[]>('/admin/users')
export const deleteUser = (id: number) => api.delete(`/admin/users/${id}`)
export const resetUserPassword = (id: number, newPassword: string) =>
  api.post(`/admin/users/${id}/reset-password`, { newPassword })
export const getAppSettings = () => api.get<AppSettings>('/admin/settings')
export const getAppSettingDefaults = () => api.get<AppSettings>('/admin/settings/defaults')
export const updateAppSettings = (settings: { key: string; value: string }[]) =>
  api.put('/admin/settings', settings)
export const getAdminLogs = (limit = 500, level?: string) => {
  const params: Record<string, string> = { limit: String(limit) }
  if (level) params.level = level
  return api.get<{ logs: LogEntry[]; count: number; logLevel: string }>('/admin/logs', { params })
}

type ConnectivityResult = { available: boolean; message: string }
export const testAnthropicConnection = () =>
  api.get<ConnectivityResult>('/admin/test-anthropic')
export const testSearXNGConnection = () =>
  api.get<ConnectivityResult>('/admin/test-searxng')

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
  challenge: string
  timeout?: number
  rpId?: string
  allowCredentials?: Array<{ id: string; type: string; transports?: string[] }>
  userVerification?: string
}

// --- Social / Profile API ---

// Profile
export const updateProfile = (data: { email?: string; bio?: string; zipCode?: string; isPublic?: boolean; numisBidsUsername?: string; numisBidsPassword?: string }) =>
  api.put<{ id: number; username: string; role: string; email: string; avatarPath: string; isPublic: boolean; bio: string; zipCode: string; numisBidsUsername: string; numisBidsConfigured: boolean }>('/user/profile', data)
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
export const listAlerts = () => api.get('/alerts')
export const createAlert = (data: { auctionLotId: number; targetPrice: number; direction?: string }) => api.post('/alerts', data)
export const deleteAlert = (id: number) => api.delete(`/alerts/${id}`)

// Bid Reminders
export const listReminders = () => api.get('/reminders')
export const createReminder = (data: { auctionLotId: number; minutesBefore?: number }) => api.post('/reminders', data)
export const deleteReminder = (id: number) => api.delete(`/reminders/${id}`)

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
export const getAuctionLots = (params?: { status?: string; search?: string; sort?: string; order?: string; page?: number; limit?: number }) =>
  api.get<AuctionLotListResponse>('/auctions', { params })
export const getAuctionLotCounts = () =>
  api.get<{ counts: Record<string, number> }>('/auctions/counts')
export const updateAuctionLotStatus = (id: number, status: string, maxBid?: number | null) => api.put<AuctionLot>(`/auctions/${id}/status`, { status, ...(maxBid != null ? { maxBid } : {}) })
export const convertAuctionLotToCoin = (id: number) => api.post<Coin>(`/auctions/${id}/convert`)
export const deleteAuctionLot = (id: number) => api.delete(`/auctions/${id}`)
export const linkAuctionLotEvent = (id: number, eventId: number | null) => api.put<AuctionLot>(`/auctions/${id}/event`, { eventId })
export const bulkLinkAuctionLotEvent = (lotIds: number[], eventId: number | null) => api.put<{ updated: number }>('/auctions/bulk-link-event', { lotIds, eventId })
export const importAuctionLot = (data: { url: string; title?: string; description?: string; auctionHouse?: string; saleName?: string; category?: string; imageUrl?: string; estimate?: number | null; currentBid?: number | null; currency?: string }) =>
  api.post<AuctionLot>('/auctions/import', data)
export const syncNumisBidsWatchlist = () =>
  api.post<{ synced: number; lots: AuctionLot[] }>('/auctions/sync')
export const validateNumisBidsCredentials = (username: string, password: string) =>
  api.post<{ valid: boolean; error?: string }>('/auctions/validate-credentials', { username, password })

// Availability checks
export const checkWishlistAvailability = () =>
  api.post<AvailabilityRunSummary>('/wishlist/check-availability')
export const updateListingStatus = (coinId: number, status: string) =>
  api.put(`/coins/${coinId}/listing-status`, { status })
export const getAvailabilityRuns = (page = 1, limit = 20) =>
  api.get<{ runs: AvailabilityRun[]; total: number }>('/admin/availability-runs', { params: { page, limit } })
export const getAvailabilityRunDetail = (runId: number) =>
  api.get<AvailabilityRun>(`/admin/availability-runs/${runId}`)

// Valuation Runs
export const getValuationRuns = (page = 1, limit = 20) =>
  api.get<{ runs: ValuationRun[]; total: number }>('/admin/valuation-runs', { params: { page, limit } })
export const getValuationRunDetail = (runId: number) =>
  api.get<ValuationRun>(`/admin/valuation-runs/${runId}`)
export const triggerValuation = () =>
  api.post<{ message: string; users: number }>('/admin/valuation-runs/trigger')
export const cancelValuationRun = (runId: number) =>
  api.post<{ message: string }>(`/admin/valuation-runs/${runId}/cancel`)

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
