import { refreshAccessToken } from '@/api/client'

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

export function normalizeUploadPath(path: string | null | undefined): string {
  const value = path?.trim() ?? ''
  if (!value) return ''

  return value
    .replace(API_BASE, '')
    .replace(/^\/api\/uploads\//, '')
    .replace(/^\/uploads\//, '')
    .replace(/^\/+/, '')
}

export function isPrivateUploadPath(path: string | null | undefined): boolean {
  const value = path?.trim() ?? ''
  if (!value) return false
  if (value.startsWith('blob:') || value.startsWith('data:')) return false
  if (value.startsWith('/') && !value.startsWith('/uploads/') && !value.startsWith('/api/uploads/')) return false
  if (/^https?:\/\//i.test(value)) {
    return value.includes('/uploads/') || value.includes('/api/uploads/')
  }
  return true
}

export function privateMediaUrl(path: string): string {
  return `${API_BASE}/api/uploads/${normalizeUploadPath(path)}`
}

export function publicShowcaseMediaUrl(slug: string, path: string): string {
  return `${API_BASE}/api/showcase/${encodeURIComponent(slug)}/uploads/${normalizeUploadPath(path)}`
}

function authHeaders(token: string | null): Headers {
  const headers = new Headers()
  headers.set('Cache-Control', 'no-store')
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  return headers
}

async function fetchPrivateMediaResponse(path: string, token: string | null): Promise<Response> {
  return fetch(privateMediaUrl(path), {
    headers: authHeaders(token),
    cache: 'no-store',
  })
}

export async function fetchPrivateMediaBlob(path: string): Promise<Blob> {
  let response = await fetchPrivateMediaResponse(path, localStorage.getItem('token'))

  if (response.status === 401 && localStorage.getItem('refreshToken')) {
    const token = await refreshAccessToken()
    response = await fetchPrivateMediaResponse(path, token)
  }

  if (!response.ok) {
    throw new Error(`Failed to load media (${response.status})`)
  }

  return response.blob()
}

export async function privateMediaObjectUrl(path: string): Promise<string> {
  if (!isPrivateUploadPath(path)) return path
  const blob = await fetchPrivateMediaBlob(path)
  return URL.createObjectURL(blob)
}
