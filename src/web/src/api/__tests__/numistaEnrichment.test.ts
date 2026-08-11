import { beforeEach, describe, expect, it, vi } from 'vitest'
import { makeNumistaEnrichmentRequest } from '@/test/numista-fixtures'

const mockApi = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  interceptors: {
    request: { use: vi.fn() },
    response: { use: vi.fn() },
  },
}

vi.mock('axios', () => ({
  default: {
    create: vi.fn(() => mockApi),
    post: vi.fn(),
  },
}))

const client = await import('../client')

describe('Numista enrichment API contract', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockApi.post.mockResolvedValue({ data: {} })
  })

  it('posts the complete typed broad candidate set with an AbortSignal', async () => {
    const request = makeNumistaEnrichmentRequest()
    const controller = new AbortController()

    await client.enrichNumista(request, controller.signal)

    expect(mockApi.post).toHaveBeenCalledWith(
      '/numista/enrich',
      request,
      { signal: controller.signal },
    )
  })
})
