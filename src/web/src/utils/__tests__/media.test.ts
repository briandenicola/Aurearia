import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { clearPrivateMediaBlobCache, fetchPrivateMediaBlob, privateMediaUrl, publicShowcaseMediaUrl } from '@/utils/media'

vi.mock('@/api/client', () => ({
  refreshAccessToken: vi.fn(),
}))

describe('media helpers', () => {
  beforeEach(() => {
    clearPrivateMediaBlobCache()
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

  it('deduplicates concurrent private upload fetches for the same media path', async () => {
    localStorage.setItem('token', 'jwt-token')

    const [first, second] = await Promise.all([
      fetchPrivateMediaBlob('/uploads/coins/aureus.webp'),
      fetchPrivateMediaBlob('coins/aureus.webp'),
    ])

    expect(first).toBeInstanceOf(Blob)
    expect(second).toBe(first)
    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('reuses cached private upload blobs across repeated callers', async () => {
    localStorage.setItem('token', 'jwt-token')

    await fetchPrivateMediaBlob('/uploads/coins/aureus.webp')
    await fetchPrivateMediaBlob('/api/uploads/coins/aureus.webp')

    expect(fetch).toHaveBeenCalledTimes(1)
  })

  it('clears cached private upload blobs on demand', async () => {
    localStorage.setItem('token', 'jwt-token')

    await fetchPrivateMediaBlob('/uploads/coins/aureus.webp')
    clearPrivateMediaBlobCache()
    await fetchPrivateMediaBlob('/uploads/coins/aureus.webp')

    expect(fetch).toHaveBeenCalledTimes(2)
  })
})
