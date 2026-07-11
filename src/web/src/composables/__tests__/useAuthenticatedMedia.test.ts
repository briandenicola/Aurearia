import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { flushPromises } from '@vue/test-utils'

const mocks = vi.hoisted(() => ({
  isPrivateUploadPath: vi.fn(),
  privateMediaObjectUrl: vi.fn(),
}))

vi.mock('@/utils/media', () => ({
  isPrivateUploadPath: mocks.isPrivateUploadPath,
  privateMediaObjectUrl: mocks.privateMediaObjectUrl,
}))

// Import after mock is established
const { useAuthenticatedMedia } = await import('@/composables/useAuthenticatedMedia')

describe('useAuthenticatedMedia', () => {
  beforeEach(() => {
    mocks.isPrivateUploadPath.mockReset()
    mocks.privateMediaObjectUrl.mockReset()
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn((b: Blob) => `blob:${b.size}`),
      revokeObjectURL: vi.fn(),
    })
  })

  it('sets objectUrl for a public (non-private) path directly without fetch', async () => {
    mocks.isPrivateUploadPath.mockReturnValue(false)

    const path = ref<string | null | undefined>('/coin-logo.jpg')
    const { objectUrl, loadFailed } = useAuthenticatedMedia(path)

    await nextTick()

    expect(objectUrl.value).toBe('/coin-logo.jpg')
    expect(loadFailed.value).toBe(false)
    expect(mocks.privateMediaObjectUrl).not.toHaveBeenCalled()
  })

  it('fetches a private path and sets objectUrl to the blob URL', async () => {
    mocks.isPrivateUploadPath.mockReturnValue(true)
    mocks.privateMediaObjectUrl.mockResolvedValue('blob:test-image')

    const path = ref<string | null | undefined>('coin-1/image.jpg')
    const { objectUrl, loadFailed } = useAuthenticatedMedia(path)

    await flushPromises()

    expect(objectUrl.value).toBe('blob:test-image')
    expect(loadFailed.value).toBe(false)
  })

  it('sets loadFailed when the fetch rejects', async () => {
    mocks.isPrivateUploadPath.mockReturnValue(true)
    mocks.privateMediaObjectUrl.mockRejectedValue(new Error('404'))

    const path = ref<string | null | undefined>('coin-1/missing.jpg')
    const { objectUrl, loadFailed } = useAuthenticatedMedia(path)

    await flushPromises()

    expect(objectUrl.value).toBe('')
    expect(loadFailed.value).toBe(true)
  })

  it('resets loadFailed when the path changes to a working path', async () => {
    mocks.isPrivateUploadPath.mockReturnValue(true)
    mocks.privateMediaObjectUrl.mockRejectedValueOnce(new Error('404'))
    mocks.privateMediaObjectUrl.mockResolvedValue('blob:new-image')

    const path = ref<string | null | undefined>('coin-1/missing.jpg')
    const { objectUrl, loadFailed } = useAuthenticatedMedia(path)
    await flushPromises()
    expect(loadFailed.value).toBe(true)

    path.value = 'coin-1/working.jpg'
    await flushPromises()

    expect(objectUrl.value).toBe('blob:new-image')
    expect(loadFailed.value).toBe(false)
  })

  it('does not wipe objectUrl when a stale cancelled request fails after a newer one succeeds (race condition fix)', async () => {
    mocks.isPrivateUploadPath.mockReturnValue(true)

    // First call (stale) resolves slowly and rejects
    // Second call (new path) resolves successfully
    let rejectStale!: (err: Error) => void
    const stalePromise = new Promise<string>((_, reject) => { rejectStale = reject })
    mocks.privateMediaObjectUrl
      .mockReturnValueOnce(stalePromise)
      .mockResolvedValue('blob:new-image')

    const path = ref<string | null | undefined>('coin-1/stale.jpg')
    const { objectUrl, loadFailed } = useAuthenticatedMedia(path)

    // Change path before the stale promise settles
    path.value = 'coin-1/new.jpg'
    await flushPromises()

    // New path succeeded: image should be displayed
    expect(objectUrl.value).toBe('blob:new-image')
    expect(loadFailed.value).toBe(false)

    // Now the stale request rejects — should NOT wipe the current result
    rejectStale(new Error('404 stale'))
    await flushPromises()

    expect(objectUrl.value).toBe('blob:new-image')
    expect(loadFailed.value).toBe(false)
  })

  it('clears objectUrl and does not set loadFailed when path becomes null', async () => {
    mocks.isPrivateUploadPath.mockReturnValue(true)
    mocks.privateMediaObjectUrl.mockResolvedValue('blob:test-image')

    const path = ref<string | null | undefined>('coin-1/image.jpg')
    const { objectUrl, loadFailed } = useAuthenticatedMedia(path)
    await flushPromises()
    expect(objectUrl.value).toBe('blob:test-image')

    path.value = null
    await nextTick()

    expect(objectUrl.value).toBe('')
    expect(loadFailed.value).toBe(false)
  })
})
