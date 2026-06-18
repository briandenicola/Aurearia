import { describe, it, expect, beforeEach, vi, type Mock } from 'vitest'
import type { InternalAxiosRequestConfig, AxiosResponse, AxiosError } from 'axios'
import type { AuthResponse, Coin } from '@/types'

// We need to mock axios BEFORE importing client
vi.mock('axios', async () => {
  const create = vi.fn()

  // Interceptor registries — we capture the handlers so we can invoke them in tests
  const requestHandlers: Array<(config: InternalAxiosRequestConfig) => InternalAxiosRequestConfig> = []
  const responseHandlers: Array<{
    onFulfilled: (res: AxiosResponse) => AxiosResponse
    onRejected: (err: AxiosError) => Promise<never>
  }> = []

  const mockInstance: Record<string, unknown> = {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: {
        use: vi.fn((handler: (config: InternalAxiosRequestConfig) => InternalAxiosRequestConfig) => {
          requestHandlers.push(handler)
        }),
      },
      response: {
        use: vi.fn((
          onFulfilled: (res: AxiosResponse) => AxiosResponse,
          onRejected: (err: AxiosError) => Promise<never>,
        ) => {
          responseHandlers.push({ onFulfilled, onRejected })
        }),
      },
    },
    defaults: { headers: { common: {} } },
  }

  create.mockReturnValue(mockInstance)

  return {
    default: {
      create,
      post: vi.fn(),
      get: vi.fn(),
    },
    __mockInstance: mockInstance,
    __requestHandlers: requestHandlers,
    __responseHandlers: responseHandlers,
  }
})

// Import after mock is established
interface MockedAxiosModule {
  __mockInstance: Record<string, unknown>
  __requestHandlers: Array<(config: InternalAxiosRequestConfig) => InternalAxiosRequestConfig>
  __responseHandlers: Array<{
    onFulfilled: (res: AxiosResponse) => AxiosResponse
    onRejected: (err: AxiosError) => Promise<never>
  }>
}
const { __mockInstance: mockApi, __requestHandlers: requestHandlers, __responseHandlers: responseHandlers } =
  (await import('axios')) as unknown as MockedAxiosModule

// Now import the client — this triggers interceptor registration
const client = await import('../client')

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function getRequestInterceptor() {
  return requestHandlers[0]
}

function getResponseErrorHandler() {
  return responseHandlers[0]?.onRejected
}

function makeStorageMock(): Record<string, string> {
  const store: Record<string, string> = {}
  return store
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('API Client', () => {
  let storageMock: Record<string, string>

  beforeEach(() => {
    storageMock = makeStorageMock()

    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storageMock[key] ?? null),
      setItem: vi.fn((key: string, value: string) => { storageMock[key] = value }),
      removeItem: vi.fn((key: string) => { delete storageMock[key] }),
    })

    vi.stubGlobal('window', { location: { href: '' } })

    vi.clearAllMocks()
  })

  // ========================================================================
  // sanitizeCoin
  // ========================================================================

  describe('sanitizeCoin (via createCoin / updateCoin)', () => {
    // sanitizeCoin is not exported directly, so we test it through the public API wrappers

    it('converts empty strings to null on nullable fields', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Denarius',
        weightGrams: '' as unknown as number,
        diameterMm: '' as unknown as number,
        purchasePrice: '' as unknown as number,
        currentValue: '' as unknown as number,
        purchaseDate: '' as unknown as string,
      }

      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.weightGrams).toBeNull()
      expect(sentData.diameterMm).toBeNull()
      expect(sentData.purchasePrice).toBeNull()
      expect(sentData.currentValue).toBeNull()
      expect(sentData.purchaseDate).toBeNull()
    })

    it('converts undefined to null on nullable fields', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = { name: 'Sestertius' }
      // weightGrams, diameterMm etc. are undefined
      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.weightGrams).toBeNull()
      expect(sentData.diameterMm).toBeNull()
      expect(sentData.purchasePrice).toBeNull()
    })

    it('preserves 0 as a valid numeric value (not treated as falsy)', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Free Coin',
        weightGrams: 0,
        diameterMm: 0,
        purchasePrice: 0,
        currentValue: 0,
      }

      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.weightGrams).toBe(0)
      expect(sentData.diameterMm).toBe(0)
      expect(sentData.purchasePrice).toBe(0)
      expect(sentData.currentValue).toBe(0)
    })

    it('preserves valid numeric values', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Aureus',
        weightGrams: 7.8,
        diameterMm: 20.5,
        purchasePrice: 1500,
        currentValue: 2000,
      }

      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.weightGrams).toBe(7.8)
      expect(sentData.diameterMm).toBe(20.5)
      expect(sentData.purchasePrice).toBe(1500)
      expect(sentData.currentValue).toBe(2000)
    })

    it('defaults currentValue to purchasePrice when currentValue is empty', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Tetradrachm',
        purchasePrice: 500,
        currentValue: '' as unknown as number,
      }

      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.currentValue).toBe(500)
    })

    it('does not default currentValue when purchasePrice is also empty', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Mystery Coin',
        purchasePrice: '' as unknown as number,
        currentValue: '' as unknown as number,
      }

      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.currentValue).toBeNull()
      expect(sentData.purchasePrice).toBeNull()
    })

    it('converts date-only string to RFC3339 format', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Dated Coin',
        purchaseDate: '2024-03-15',
      }

      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.purchaseDate).toBe('2024-03-15T00:00:00Z')
    })

    it('does not modify already-formatted datetime strings', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Full Date Coin',
        purchaseDate: '2024-03-15T10:30:00Z',
      }

      await client.createCoin(coin)

      const sentData = mockApi.post.mock.calls[0]![1]
      expect(sentData.purchaseDate).toBe('2024-03-15T10:30:00Z')
    })

    it('does not mutate the original coin object', async () => {
      mockApi.post.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Immutable Coin',
        weightGrams: '' as unknown as number,
        purchaseDate: '2024-01-01',
      }

      await client.createCoin(coin)

      expect(coin.weightGrams).toBe('')
      expect(coin.purchaseDate).toBe('2024-01-01')
    })

    it('sanitizes through updateCoin as well', async () => {
      mockApi.put.mockResolvedValue({ data: {} })

      const coin: Partial<Coin> = {
        name: 'Updated Coin',
        weightGrams: '' as unknown as number,
      }

      await client.updateCoin(42, coin)

      const sentData = mockApi.put.mock.calls[0]![1]
      expect(sentData.weightGrams).toBeNull()
    })
  })

  // ========================================================================
  // Request interceptor — JWT token
  // ========================================================================

  describe('JWT request interceptor', () => {
    it('adds Authorization header when token exists in localStorage', () => {
      storageMock['token'] = 'my-jwt-token'
      const interceptor = getRequestInterceptor()
      const config = { headers: {} } as InternalAxiosRequestConfig

      const result = interceptor(config)

      expect(result.headers.Authorization).toBe('Bearer my-jwt-token')
    })

    it('does not add Authorization header when no token', () => {
      const interceptor = getRequestInterceptor()
      const config = { headers: {} } as InternalAxiosRequestConfig

      const result = interceptor(config)

      expect(result.headers.Authorization).toBeUndefined()
    })
  })

  // ========================================================================
  // Response interceptor — 401 handling
  // ========================================================================

  describe('401 response interceptor', () => {
    it('clears auth and redirects when no refreshToken exists', async () => {
      const onError = getResponseErrorHandler()

      const error = {
        response: { status: 401 },
        config: { _retry: false, headers: {} },
      }

      await expect(onError(error)).rejects.toBe(error)
      expect(localStorage.removeItem).toHaveBeenCalledWith('token')
      expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken')
    })

    it('attempts token refresh when refreshToken exists', async () => {
      storageMock['refreshToken'] = 'valid-refresh-token'
      const onError = getResponseErrorHandler()

      const newAuth: AuthResponse = {
        token: 'new-jwt',
        refreshToken: 'new-refresh',
        user: { id: 1, username: 'u', role: 'user', email: '', avatarPath: '', isPublic: false, bio: '', zipCode: '' },
      }

      const defaultAxios = (await import('axios')).default
      ;(defaultAxios.post as Mock).mockResolvedValue({ data: newAuth })
      mockApi.mockImplementation?.(() => Promise.resolve({ data: 'retried' }))

      // Note: callableMock setup omitted — difficult to test full retry flow with mockApi

      const error = {
        response: { status: 401 },
        config: { _retry: false, headers: {} },
      }

      // We can't easily test the full retry flow because mockApi isn't callable,
      // but we can verify the refresh request is made
      try {
        await onError(error)
      } catch {
        // may reject if mock isn't fully callable
      }

      expect(defaultAxios.post).toHaveBeenCalledWith(
        '/api/auth/refresh',
        { refreshToken: 'valid-refresh-token' },
      )
    })

    it('passes through non-401 errors without refresh attempt', async () => {
      const onError = getResponseErrorHandler()

      const error = {
        response: { status: 500 },
        config: { headers: {} },
      }

      await expect(onError(error)).rejects.toBe(error)
      const defaultAxios = (await import('axios')).default
      expect(defaultAxios.post).not.toHaveBeenCalled()
    })

    it('does not retry if request was already retried', async () => {
      storageMock['refreshToken'] = 'some-token'
      const onError = getResponseErrorHandler()

      const error = {
        response: { status: 401 },
        config: { _retry: true, headers: {} },
      }

      await expect(onError(error)).rejects.toBe(error)
    })
  })

  // ========================================================================
  // API wrapper methods — URL construction
  // ========================================================================

  describe('API method wrappers', () => {
    it('login sends POST to /auth/login', async () => {
      mockApi.post.mockResolvedValue({ data: {} })
      await client.login('user', 'pass')
      expect(mockApi.post).toHaveBeenCalledWith('/auth/login', { username: 'user', password: 'pass' })
    })

    it('register sends POST to /auth/register with optional email', async () => {
      mockApi.post.mockResolvedValue({ data: {} })
      await client.register('user', 'pass', 'u@example.com')
      expect(mockApi.post).toHaveBeenCalledWith('/auth/register', { username: 'user', password: 'pass', email: 'u@example.com' })
    })

    it('getCoins sends GET to /coins with params', async () => {
      mockApi.get.mockResolvedValue({ data: {} })
      await client.getCoins({ category: 'Roman', page: 2, limit: 10 })
      expect(mockApi.get).toHaveBeenCalledWith('/coins', { params: { category: 'Roman', page: 2, limit: 10 } })
    })

    it('getCoin sends GET to /coins/:id', async () => {
      mockApi.get.mockResolvedValue({ data: {} })
      await client.getCoin(42)
      expect(mockApi.get).toHaveBeenCalledWith('/coins/42')
    })

    it('deleteCoin sends DELETE to /coins/:id', async () => {
      mockApi.delete.mockResolvedValue({ data: {} })
      await client.deleteCoin(99)
      expect(mockApi.delete).toHaveBeenCalledWith('/coins/99')
    })

    it('notes CRUD methods use the /notes contract', async () => {
      mockApi.get.mockResolvedValue({ data: { notes: [] } })
      mockApi.post.mockResolvedValue({ data: {} })
      mockApi.put.mockResolvedValue({ data: {} })
      mockApi.delete.mockResolvedValue({ data: {} })

      await client.getNotes()
      await client.createNote({ title: 'Research links', body: '- Dealer listing' })
      await client.updateNote(7, { title: 'Updated links', body: '**Important**' })
      await client.deleteNote(7)

      expect(mockApi.get).toHaveBeenCalledWith('/notes')
      expect(mockApi.post).toHaveBeenCalledWith('/notes', { title: 'Research links', body: '- Dealer listing' })
      expect(mockApi.put).toHaveBeenCalledWith('/notes/7', { title: 'Updated links', body: '**Important**' })
      expect(mockApi.delete).toHaveBeenCalledWith('/notes/7')
    })

    it('searchNumista sends GET with query param', async () => {
      mockApi.get.mockResolvedValue({ data: {} })
      await client.searchNumista('denarius')
      expect(mockApi.get).toHaveBeenCalledWith('/numista/search', { params: { q: 'denarius' } })
    })

    it('getStats sends GET to /stats', async () => {
      mockApi.get.mockResolvedValue({ data: {} })
      await client.getStats()
      expect(mockApi.get).toHaveBeenCalledWith('/stats')
    })

    it('sellCoin sends POST with soldPrice and soldTo', async () => {
      mockApi.post.mockResolvedValue({ data: {} })
      await client.sellCoin(5, 100, 'buyer')
      expect(mockApi.post).toHaveBeenCalledWith('/coins/5/sell', { soldPrice: 100, soldTo: 'buyer' })
    })

    it('triggerCollectionHealthSnapshots sends POST to the admin run endpoint', async () => {
      mockApi.post.mockResolvedValue({ data: {} })
      await client.triggerCollectionHealthSnapshots()
      expect(mockApi.post).toHaveBeenCalledWith('/admin/collection-health-snapshots/run')
    })
  })
})
