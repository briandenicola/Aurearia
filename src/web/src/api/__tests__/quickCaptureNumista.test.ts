import { beforeEach, describe, expect, it, vi } from 'vitest'

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

function formEntries(call: unknown[]): Record<string, FormDataEntryValue[]> {
  const form = call[1] as FormData
  const entries: Record<string, FormDataEntryValue[]> = {}
  for (const [key, value] of form.entries()) {
    entries[key] = [...(entries[key] ?? []), value]
  }
  return entries
}

describe('Quick Capture Numista API contracts', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockApi.post.mockResolvedValue({ data: {} })
    mockApi.put.mockResolvedValue({ data: {} })
    mockApi.get.mockResolvedValue({ data: {} })
  })

  it('serializes the photo-analysis multipart request without starting Numista lookup', async () => {
    const obverse = new File(['obverse'], 'obverse.jpg', { type: 'image/jpeg' })
    const reverse = new File(['reverse'], 'reverse.jpg', { type: 'image/jpeg' })

    await client.lookupCoin([obverse, reverse], 'Weight 3.2 g', ['obverse', 'reverse'])

    expect(mockApi.post).toHaveBeenCalledTimes(1)
    expect(mockApi.post.mock.calls[0]?.[0]).toBe('/coins/lookup')
    expect(formEntries(mockApi.post.mock.calls[0] ?? []).images).toHaveLength(2)
    expect(formEntries(mockApi.post.mock.calls[0] ?? []).notes).toEqual(['Weight 3.2 g'])
    expect(formEntries(mockApi.post.mock.calls[0] ?? []).imageRoles).toEqual(['obverse', 'reverse'])
  })

  it('creates a draft with exactly the selected Numista id and canonical URL', async () => {
    await client.createQuickCaptureDraft({
      workingTitle: 'Trajan denarius',
      selectedNumistaId: '12345',
      selectedNumistaUrl: 'https://en.numista.com/catalogue/pieces12345.html',
    })

    expect(formEntries(mockApi.post.mock.calls[0] ?? [])).toMatchObject({
      workingTitle: ['Trajan denarius'],
      selectedNumistaId: ['12345'],
      selectedNumistaUrl: ['https://en.numista.com/catalogue/pieces12345.html'],
    })
  })

  it('reads the additive selected-reference response through the typed draft endpoint', async () => {
    await client.getQuickCaptureDraft(12)

    expect(mockApi.get).toHaveBeenCalledWith('/quick-capture/drafts/12')
  })

  it('omits selection fields on unrelated updates, replaces with a pair, and clears explicitly', async () => {
    const base = {
      workingTitle: 'Edited title',
      dateRange: '',
      era: 'ancient',
      acquisitionSource: '',
      purchasePrice: null,
      notes: '',
    }

    await client.updateQuickCaptureDraft(12, base)
    let entries = formEntries(mockApi.put.mock.calls[0] ?? [])
    expect(entries.selectedNumistaId).toBeUndefined()
    expect(entries.selectedNumistaUrl).toBeUndefined()
    expect(entries.clearSelectedNumista).toBeUndefined()

    await client.updateQuickCaptureDraft(12, {
      ...base,
      selectedNumistaId: '67890',
      selectedNumistaUrl: 'https://en.numista.com/catalogue/pieces67890.html',
    })
    entries = formEntries(mockApi.put.mock.calls[1] ?? [])
    expect(entries.selectedNumistaId).toEqual(['67890'])
    expect(entries.selectedNumistaUrl).toEqual(['https://en.numista.com/catalogue/pieces67890.html'])

    await client.updateQuickCaptureDraft(12, { ...base, clearSelectedNumista: true })
    entries = formEntries(mockApi.put.mock.calls[2] ?? [])
    expect(entries.clearSelectedNumista).toEqual(['true'])
    expect(entries.selectedNumistaId).toBeUndefined()
  })

  it('keeps the promotion request additive and idempotency response typed', async () => {
    await client.promoteQuickCaptureDraft(12, {
      confirm: true,
      target: 'wishlist',
      overrides: { name: 'Wanted denarius' },
    })

    expect(mockApi.post).toHaveBeenCalledWith('/quick-capture/drafts/12/promote', {
      confirm: true,
      target: 'wishlist',
      overrides: { name: 'Wanted denarius' },
    })
  })
})
