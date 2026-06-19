import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fetchPrivateMediaBlob, privateMediaUrl, publicShowcaseMediaUrl } from '@/utils/media'

vi.mock('@/api/client', () => ({
  refreshAccessToken: vi.fn(),
}))

describe('media helpers', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.stubGlobal('fetch', vi.fn(async () => new Response(new Blob(['image']), { status: 200 })))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('fetches private uploads as blobs with the bearer token and no-store cache policy', async () => {
    localStorage.setItem('token', 'jwt-token')

    const blob = await fetchPrivateMediaBlob('/uploads/coins/aureus.webp')

    expect(blob).toBeInstanceOf(Blob)
    expect(fetch).toHaveBeenCalledWith('/api/uploads/coins/aureus.webp', {
      headers: expect.any(Headers),
      cache: 'no-store',
    })
    const headers = vi.mocked(fetch).mock.calls[0]?.[1]?.headers as Headers
    expect(headers.get('Authorization')).toBe('Bearer jwt-token')
    expect(headers.get('Cache-Control')).toBe('no-store')
  })

  it('builds private and public showcase media routes', () => {
    expect(privateMediaUrl('coins/denarius.webp')).toBe('/api/uploads/coins/denarius.webp')
    expect(publicShowcaseMediaUrl('featured-set', '/uploads/coins/denarius.webp')).toBe(
      '/api/showcase/featured-set/uploads/coins/denarius.webp',
    )
  })
})
