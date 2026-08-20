// admin types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.
import type { NumistaSettings } from '@/types/numista'

export interface AppSettings extends Partial<NumistaSettings> {
  AIProvider: string
  OllamaURL: string
  OllamaModel: string
  ObversePrompt: string
  ReversePrompt: string
  TextExtractionPrompt: string
  OllamaTimeout: string
  SearXNGURL: string
  LogLevel: string
  PublicAppURL?: string
  RegistrationMode?: string
  AuctionAlertsCheckEnabled?: string
  AuctionAlertsCheckStartTime?: string
  AuctionAlertsCheckInterval?: string
  WishlistSearchAlertsCheckEnabled?: string
  WishlistSearchAlertsCheckStartTime?: string
  ReminderCheckEnabled?: string
  ReminderCheckStartTime?: string
  CoinCategories?: string
  CoinEras?: string
  DeepIdentificationOCREEnabled?: string
  DeepIdentificationOCRECallBudget?: string
  [key: string]: string | undefined
}

// OCREHealthSummary mirrors the Go models.OCREHealthSummary bounded admin
// view of the OCRE Deep Analysis provider (Feature 345 US4). It carries only
// enablement/gate state and the last recorded outcome class — no per-job
// user content.
export interface OCREHealthSummary {
  enabled: boolean
  callBudget: number
  gateValidated: boolean
  lastOutcome?: string | null
  lastCheckedAt?: string | null
}

export interface SecuritySummary {
  failedLogins: number
  lockedAccounts: number
  activeBans: number
  recentEvents: number
  loginFailures?: number
  activeIpRuleCount?: number
}

export interface SecurityEvent {
  id: number | string
  timestamp: string
  type: string
  severity: string
  username?: string | null
  ip?: string | null
  clientIp?: string | null
  outcome?: string | null
  message?: string | null
  userAgent?: string | null
  createdAt?: string
}

export interface SecurityEventFilters {
  type?: string
  severity?: string
  username?: string
  ip?: string
  clientIp?: string
  outcome?: string
  since?: string
  limit?: number
}

export interface SecurityEventsResponse {
  events: SecurityEvent[]
  total?: number
}

export interface SecurityIpRule {
  id: number
  cidr: string
  reason: string
  expiresAt?: string | null
  createdBy?: string | number | null
  createdAt?: string
}

export interface CreateSecurityIpRuleRequest {
  cidr: string
  duration?: string
  durationMinutes?: number
  expiresAt?: string
  reason: string
}

export interface SecurityExposureCheck {
  publicIp?: string
  proxy?: boolean
  proxyWarning?: string
  cors?: boolean
  corsWarning?: string
  webAuthn?: boolean
  webAuthnWarning?: string
  publicAppUrl?: boolean
  publicAppURL?: boolean
  publicAppUrlWarning?: string
  registration?: boolean
  registrationWarning?: string
  agentToken?: boolean
  agentTokenWarning?: string
  warnings?: string[]
  checks?: Record<string, boolean | string | null | undefined>
  config?: {
    publicAppURL?: string
    webauthnOrigin?: string
    trustedProxiesConfigured?: boolean
    agentInternalTokenSet?: boolean
    registrationMode?: string
    backupStatus?: string
  }
}

export type Theme = 'dark' | 'light' | 'british-museum' | 'louvre' | 'capitoline' | 'byzantine' | 'modern-greek'

export const LOG_LEVELS = ['trace', 'debug', 'info', 'warn', 'error'] as const

export interface LogEntry {
  timestamp: string
  level: string
  message: string
}
