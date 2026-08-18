import { describe, expect, it, vi, beforeEach } from 'vitest'
import { useCoinImageUpload } from '../useCoinImageUpload'
import { proxyImage, uploadImage } from '@/api/client'

vi.mock('@/api/client', () => ({
  uploadImage: vi.fn(),
  proxyImage: vi.fn(),
}))

describe('useCoinImageUpload', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('uploads a picked file and reports completion', async () => {
    vi.mocked(uploadImage).mockResolvedValue({} as Awaited<ReturnType<typeof uploadImage>>)
    const onUploaded = vi.fn()
    const upload = useCoinImageUpload(42, 0, { onUploaded })

    const file = new File(['x'], 'obverse.jpg', { type: 'image/jpeg' })
    const event = { target: { files: [file] } } as unknown as Event
    await upload.handleImageUpload(event)

    expect(uploadImage).toHaveBeenCalledWith(42, file, 'obverse', true)
    expect(upload.uploadStatus.value).toBe('Upload complete!')
    expect(upload.uploadError.value).toBe(false)
    expect(onUploaded).toHaveBeenCalledTimes(1)
  })

  it('marks isPrimary false once the coin already has images', async () => {
    vi.mocked(uploadImage).mockResolvedValue({} as Awaited<ReturnType<typeof uploadImage>>)
    const upload = useCoinImageUpload(42, 3)

    const file = new File(['x'], 'reverse.jpg', { type: 'image/jpeg' })
    await upload.handleImageUpload({ target: { files: [file] } } as unknown as Event)

    expect(uploadImage).toHaveBeenCalledWith(42, file, 'obverse', false)
  })

  it('does nothing when no file was chosen', async () => {
    const upload = useCoinImageUpload(42, 0)
    await upload.handleImageUpload({ target: { files: [] } } as unknown as Event)
    expect(uploadImage).not.toHaveBeenCalled()
    expect(upload.uploadStatus.value).toBe('')
  })

  it('surfaces an upload failure without throwing', async () => {
    vi.mocked(uploadImage).mockRejectedValue(new Error('boom'))
    const onUploaded = vi.fn()
    const upload = useCoinImageUpload(42, 0, { onUploaded })

    const file = new File(['x'], 'obverse.jpg', { type: 'image/jpeg' })
    await upload.handleImageUpload({ target: { files: [file] } } as unknown as Event)

    expect(upload.uploadStatus.value).toBe('Upload failed')
    expect(upload.uploadError.value).toBe(true)
    expect(onUploaded).not.toHaveBeenCalled()
  })

  it('circle-clips camera captures for obverse/reverse but not for other slots', async () => {
    vi.mocked(uploadImage).mockResolvedValue({} as Awaited<ReturnType<typeof uploadImage>>)
    const upload = useCoinImageUpload(42, 1)

    const file = new File(['x'], 'capture.jpg', { type: 'image/jpeg' })
    await upload.handleCameraCaptured(file)
    expect(uploadImage).toHaveBeenLastCalledWith(42, file, 'obverse', false, true)

    upload.uploadType.value = 'other'
    await upload.handleCameraCaptured(file)
    expect(uploadImage).toHaveBeenLastCalledWith(42, file, 'other', false, false)
  })

  it('fetches an image from a pasted URL and clears the field on success', async () => {
    const blob = new Blob(['data'], { type: 'image/png' })
    vi.mocked(proxyImage).mockResolvedValue({ data: blob } as Awaited<ReturnType<typeof proxyImage>>)
    vi.mocked(uploadImage).mockResolvedValue({} as Awaited<ReturnType<typeof uploadImage>>)
    const onUploaded = vi.fn()
    const upload = useCoinImageUpload(42, 0, { onUploaded })
    upload.imageUrl.value = 'https://example.com/coin.png'

    await upload.handleUrlUpload()

    expect(proxyImage).toHaveBeenCalledWith('https://example.com/coin.png')
    expect(uploadImage).toHaveBeenCalled()
    expect(upload.uploadStatus.value).toBe('Image saved from URL!')
    expect(upload.imageUrl.value).toBe('')
    expect(upload.urlLoading.value).toBe(false)
    expect(onUploaded).toHaveBeenCalledTimes(1)
  })

  it('rejects an empty response body from a pasted URL', async () => {
    vi.mocked(proxyImage).mockResolvedValue({ data: new Blob([]) } as Awaited<ReturnType<typeof proxyImage>>)
    const upload = useCoinImageUpload(42, 0)
    upload.imageUrl.value = 'https://example.com/coin.png'

    await upload.handleUrlUpload()

    expect(uploadImage).not.toHaveBeenCalled()
    expect(upload.uploadStatus.value).toBe('No image data received from URL')
    expect(upload.uploadError.value).toBe(true)
  })

  it('does nothing when the URL field is blank', async () => {
    const upload = useCoinImageUpload(42, 0)
    await upload.handleUrlUpload()
    expect(proxyImage).not.toHaveBeenCalled()
  })
})
