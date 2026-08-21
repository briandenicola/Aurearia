// auth endpoints. Split out of the former monolithic client.ts;
// re-exported from '@/api/client' so existing imports keep working.
import { api } from '@/api/http'
import type {
  ApiKey,
  AuthResponse,
  Coin,
  OIDCLinkCallbackResponse,
  OIDCLinkedIdentitiesResponse,
  OIDCMessageResponse,
  OIDCPublicProvidersResponse,
  OIDCStartFlowRequest,
  OIDCStartFlowResponse,
  UserInfo,
  WebAuthnCredentialInfo,
} from '@/types'

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

// User self-service
export const getMe = () => api.get<UserInfo>('/auth/me')

export const changePassword = (currentPassword: string, newPassword: string) =>
  api.post('/auth/change-password', { currentPassword, newPassword })

export const exportCollection = () => api.get('/user/export', { responseType: 'blob' })

export const exportCatalogPDF = () => api.get('/user/export/catalog', { responseType: 'blob' })

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
export const updateProfile = (data: { email?: string; bio?: string; zipCode?: string; isPublic?: boolean; numisBidsUsername?: string; numisBidsPassword?: string; cngUsername?: string; cngPassword?: string; parcelAppAPIKey?: string; pushoverUserKey?: string; coinOfDayEnabled?: boolean; coinOfDayIncludeWishlist?: boolean; emperorTrackerEnabled?: boolean; emperorTrackerShowUsurpers?: boolean; emperorTrackerShowEmpresses?: boolean; emperorTrackerShowOtherFigures?: boolean; pwaSwipeNavEnabled?: boolean }) =>
  api.put<{ id: number; username: string; role: string; email: string; avatarPath: string; isPublic: boolean; bio: string; zipCode: string; numisBidsUsername: string; numisBidsConfigured: boolean; cngUsername: string; cngConfigured: boolean; parcelAppConfigured: boolean; pushoverEnabled: boolean; coinOfDayEnabled: boolean; coinOfDayIncludeWishlist: boolean; emperorTrackerEnabled: boolean; emperorTrackerShowUsurpers: boolean; emperorTrackerShowEmpresses: boolean; emperorTrackerShowOtherFigures: boolean; pwaSwipeNavEnabled: boolean }>('/user/profile', data)

export const uploadAvatar = (file: File) => {
  const form = new FormData()
  form.append('avatar', file)
  return api.post<{ avatarPath: string }>('/user/avatar', form)
}

export const deleteAvatar = () => api.delete('/user/avatar')
