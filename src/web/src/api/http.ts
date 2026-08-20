// Shared HTTP infrastructure: the axios instance, auth header injection,
// silent refresh-token rotation, and API error formatting. Endpoint functions
// live in ./endpoints/*.ts and import `api` from here. Nothing in this file
// may import from ./endpoints — that would create a cycle.
import axios from 'axios'
import type { AuthResponse } from '@/types'

export const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

export const api = axios.create({
  baseURL: `${API_BASE}/api`,
})

type ApiErrorPayload = {
  error?: unknown
  message?: unknown
  detail?: unknown
  code?: unknown
}

function formatApiErrorCandidate(candidate: unknown): string {
  if (typeof candidate === 'string') return candidate
  if (Array.isArray(candidate)) {
    return candidate
      .map((item) => {
        if (typeof item === 'string') return item
        if (typeof item !== 'object' || item === null) return ''
        const detail = item as { loc?: unknown; msg?: unknown }
        const field = Array.isArray(detail.loc) ? detail.loc.map(String).filter(Boolean).join('.') : ''
        const message = typeof detail.msg === 'string' ? detail.msg : ''
        if (field && message) return `${field}: ${message}`
        return message
      })
      .filter(Boolean)
      .join('; ')
  }
  if (typeof candidate === 'object' && candidate !== null) {
    const detail = candidate as { error?: unknown; message?: unknown; detail?: unknown; msg?: unknown }
    return formatApiErrorCandidate(detail.error ?? detail.message ?? detail.detail ?? detail.msg)
  }
  return ''
}

export function getApiErrorMessage(error: unknown): string {
  if (typeof error === 'string') return error

  if (typeof error === 'object' && error !== null) {
    const maybeResponse = error as { response?: { data?: ApiErrorPayload } }
    const data = maybeResponse.response?.data ?? (error as ApiErrorPayload)
    const candidate = data.error ?? data.message ?? data.detail
    const message = formatApiErrorCandidate(candidate)
    if (message) return message
  }

  if (error instanceof Error) return error.message

  return ''
}

/**
 * Extracts the backend's machine-readable `code` field (e.g.
 * `job_at_capacity`, `already_applied`) from a failed API response, when
 * present. Callers should still fall back to `getApiErrorMessage` for the
 * user-facing text - `code` is only for branching UI behavior.
 */
export function getApiErrorCode(error: unknown): string {
  if (typeof error === 'object' && error !== null) {
    const maybeResponse = error as { response?: { data?: ApiErrorPayload } }
    const data = maybeResponse.response?.data ?? (error as ApiErrorPayload)
    if (typeof data.code === 'string') return data.code
  }
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

export function formatSetBuilderError(error: unknown): string {
  const message = getApiErrorMessage(error).trim()
  if (/max_slots|maximum.*slots|less than or equal to 300/i.test(message)) {
    return 'The set-builder request exceeded the workflow slot limit. Try a narrower prompt by date range, mint, ruler, or series, then submit again.'
  }
  if (/agent service returned HTTP 422/i.test(message)) {
    return 'The agent rejected the set-builder request before it could run. Try a narrower prompt, or check the agent service configuration if this repeats.'
  }
  if (/set builder prompt is too long/i.test(message)) {
    return 'The prompt is too long. Shorten it to the key series, date range, mints, and completion goal.'
  }
  if (/failed to submit set proposal request/i.test(message)) {
    return 'The proposal request could not be submitted. Check AI provider and agent service configuration, then try again.'
  }
  return message || 'The proposal request could not be submitted. Check AI provider and agent service configuration, then try again.'
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
