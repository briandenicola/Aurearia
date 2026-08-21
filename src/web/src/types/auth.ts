// auth types. Split out of the former single-file src/types/index.ts;
// re-exported from '@/types' so existing imports keep working.

export interface User {
  id: number
  username: string
  role: 'admin' | 'user'
  email: string
  avatarPath: string
  isPublic: boolean
  bio: string
  zipCode: string
  numisBidsUsername?: string
  numisBidsConfigured?: boolean
  cngUsername?: string
  cngConfigured?: boolean
  parcelAppConfigured?: boolean
  pushoverEnabled?: boolean
  coinOfDayEnabled?: boolean
  coinOfDayIncludeWishlist?: boolean
  emperorTrackerEnabled?: boolean
  emperorTrackerShowUsurpers?: boolean
  emperorTrackerShowEmpresses?: boolean
  emperorTrackerShowOtherFigures?: boolean
  pwaSwipeNavEnabled?: boolean
}

export interface AuthResponse {
  token: string
  refreshToken: string
  user: User
}

export type OIDCProviderType = 'entra' | 'pocket_id' | 'generic'

export type OIDCTestStatus = 'unknown' | 'ok' | 'failed'

export interface OIDCPublicProvider {
  id: number
  name: string
  displayName: string
  providerType: OIDCProviderType
}

export interface OIDCPublicProvidersResponse {
  providers: OIDCPublicProvider[]
}

export interface OIDCStartFlowRequest {
  redirectPath: string
  callbackPath?: string
}

export interface OIDCStartFlowResponse {
  authorizationUrl: string
  expiresAt: string
}

export interface OIDCLinkedIdentity {
  id: number
  providerId: number
  providerDisplayName: string
  issuer: string
  subjectPreview: string
  email: string
  emailVerified: boolean
  createdAt: string
  lastLoginAt?: string | null
}

export interface OIDCLinkedIdentitiesResponse {
  identities: OIDCLinkedIdentity[]
}

export interface OIDCLinkCallbackResponse {
  message: string
  identity: OIDCLinkedIdentity
}

export interface OIDCMessageResponse {
  message: string
}

export interface OIDCAdminProvider {
  id: number
  name: string
  displayName: string
  providerType: OIDCProviderType
  enabled: boolean
  issuerUrl: string
  clientId: string
  clientSecretConfigured: boolean
  scopes: string[]
  callbackPath: string
  requireVerifiedEmail?: boolean
  lastTestedAt?: string | null
  lastTestStatus: OIDCTestStatus
  lastTestMessage: string
  createdAt?: string
  updatedAt?: string
}

export interface OIDCAdminProvidersResponse {
  providers: OIDCAdminProvider[]
}

export interface OIDCAdminProviderInput {
  name: string
  displayName: string
  providerType: OIDCProviderType
  enabled: boolean
  issuerUrl: string
  clientId: string
  clientSecret?: string
  scopes: string[]
  callbackPath?: string
  requireVerifiedEmail?: boolean
}

export type OIDCAdminProviderUpdate = Partial<OIDCAdminProviderInput>

export interface OIDCProviderTestResponse {
  available: boolean
  message: string
  issuer: string
  authorizationEndpoint: string
  tokenEndpoint: string
}

export interface WebAuthnCredentialInfo {
  id: number
  credentialId: string
  name: string
  createdAt: string
}

export interface UserInfo {
  id: number
  username: string
  role: 'admin' | 'user'
  email: string
  avatarPath: string
  isPublic: boolean
  bio: string
  zipCode: string
  emailMissing: boolean
  numisBidsUsername: string
  numisBidsConfigured: boolean
  cngUsername?: string
  cngConfigured?: boolean
  parcelAppConfigured?: boolean
  pushoverEnabled?: boolean
  coinOfDayEnabled?: boolean
  /** Spec 354 D11: per-user opt-out; default-on. When false, Coin of the Day falls back to owned-only eligibility. */
  coinOfDayIncludeWishlist?: boolean
  emperorTrackerEnabled?: boolean
  emperorTrackerShowUsurpers?: boolean
  emperorTrackerShowEmpresses?: boolean
  emperorTrackerShowOtherFigures?: boolean
  pwaSwipeNavEnabled?: boolean
  lockedUntil?: string | null
  failedLoginAttempts?: number
  createdAt: string
}

export interface ApiKey {
  id: number
  userId: number
  keyPrefix: string
  name: string
  capabilities: string // "read" or "read,write"
  createdAt: string
  lastUsedAt: string | null
  revokedAt: string | null
}
