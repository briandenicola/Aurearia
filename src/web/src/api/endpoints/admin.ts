// admin endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  AdminHealthSummaryResponse,
  AppSettings,
  AuctionAlertReminderRun,
  AuctionEndingRun,
  AuctionWatchBidDigestRun,
  AvailabilityCycleDetail,
  AvailabilityCycleListResponse,
  AvailabilityCycleTriggerResponse,
  AvailabilityRun,
  CatalogRegistry,
  CoinOfDayRun,
  CollectionHealthSnapshotRun,
  CollectionHealthSnapshotRunResult,
  CreateSecurityIpRuleRequest,
  DeepIdentificationObservabilitySummary,
  LogEntry,
  MintLocation,
  NomismaSearchResponse,
  NumistaHealthSummary,
  OCREHealthSummary,
  OIDCAdminProvider,
  OIDCAdminProviderInput,
  OIDCAdminProviderUpdate,
  OIDCAdminProvidersResponse,
  OIDCProviderTestResponse,
  SchedulerStatus,
  SecurityEventFilters,
  SecurityEventsResponse,
  SecurityExposureCheck,
  SecurityIpRule,
  SecuritySummary,
  UserInfo,
  ValuationRun,
} from '@/types'
import type { MintLocationInput } from '@/api/endpoints/collection'

type ConnectivityResult = { available: boolean; message: string }

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

export const adminCreateMintLocation = (data: MintLocationInput) =>
  api.post<MintLocation>('/admin/mint-locations', data)

export const adminUpdateMintLocation = (id: number, data: MintLocationInput) =>
  api.put<MintLocation>(`/admin/mint-locations/${id}`, data)

export const adminDeleteMintLocation = (id: number) => api.delete(`/admin/mint-locations/${id}`)

// Nomisma.org authority linking (global mint locations only; admin-only)
export const searchNomismaMintCandidates = (id: number, query: string) =>
  api.get<NomismaSearchResponse>(`/admin/mint-locations/${id}/nomisma/search`, { params: { query } })

export const linkNomismaMintLocation = (id: number, uri: string, label: string) =>
  api.post<MintLocation>(`/admin/mint-locations/${id}/nomisma`, { uri, label })

export const unlinkNomismaMintLocation = (id: number) =>
  api.delete<{ message: string }>(`/admin/mint-locations/${id}/nomisma`)

export const adminCreateCatalog = (payload: { catalog: string; displayName: string; era: string; volumeRequired: boolean }) =>
  api.post<CatalogRegistry>('/admin/catalogs', payload)

export const adminUpdateCatalog = (id: number, payload: { catalog: string; displayName: string; era: string; volumeRequired: boolean }) =>
  api.put<CatalogRegistry>(`/admin/catalogs/${id}`, payload)

export const adminDeleteCatalog = (id: number) => api.delete(`/admin/catalogs/${id}`)

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

export const getAdminNumistaHealth = () =>
  api.get<NumistaHealthSummary>('/admin/numista/health')

export const getAdminOCREHealth = () =>
  api.get<OCREHealthSummary>('/admin/deep-identification/ocre/health')

export const getAdminDeepIdentificationObservability = () =>
  api.get<DeepIdentificationObservabilitySummary>('/admin/deep-identification/observability')

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

export const testAnthropicConnection = () =>
  api.get<ConnectivityResult>('/admin/test-anthropic')

export const testSearXNGConnection = () =>
  api.get<ConnectivityResult>('/admin/test-searxng')

export const getAdminHealthSummary = () =>
  api.get<AdminHealthSummaryResponse>('/admin/health/summary')

// Legacy admin availability runs (pre-Feature-353 UserID=0 aggregate rows).
// Retained read-only for the "Legacy" section of the admin schedule history.
export const getAvailabilityRuns = (page = 1, limit = 20) =>
  api.get<{ runs: AvailabilityRun[]; total: number }>('/admin/availability-runs', { params: { page, limit } })

export const getAvailabilityRunDetail = (runId: number) =>
  api.get<AvailabilityRun>(`/admin/availability-runs/${runId}`)

export const triggerAvailabilityCheck = () =>
  api.post<AvailabilityCycleTriggerResponse>('/admin/availability/run')

// Admin availability cycles (Feature 353) — parent roll-up rows with expandable per-user children.
export const getAvailabilityCycles = (page = 1, limit = 5) =>
  api.get<AvailabilityCycleListResponse>('/admin/availability-cycles', { params: { page, limit } })

export const getAvailabilityCycleDetail = (cycleId: number) =>
  api.get<AvailabilityCycleDetail>(`/admin/availability-cycles/${cycleId}`)

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

export const triggerCoinOfDayRun = () =>
  api.post<{ runId: number; status: string }>('/admin/coin-of-day/run')

export const getCoinOfDayRuns = (page = 1, limit = 20) =>
  api.get<{ runs: CoinOfDayRun[]; total: number; page: number; limit: number }>('/admin/coin-of-day-runs', { params: { page, limit } })

export const getCoinOfDayRunDetail = (runId: number) =>
  api.get<CoinOfDayRun>(`/admin/coin-of-day-runs/${runId}`)
