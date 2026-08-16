import { afterEach, describe, expect, it, vi } from 'vitest'

import { normalizeGalleryImage } from '../galleryImage'

class TestImage {
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  naturalWidth = 4032
  naturalHeight = 3024

  set src(_value: string) {
    queueMicrotask(() => this.onload?.())
  }
}

describe('normalizeGalleryImage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('converts a gallery image to a bounded JPEG without upscaling', async () => {
    vi.stubGlobal('Image', TestImage)
    vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:gallery-image')
    const revoke = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
    const drawImage = vi.fn()
    const canvas = {
      width: 0,
      height: 0,
      getContext: vi.fn(() => ({ drawImage })),
      toBlob: vi.fn((callback: BlobCallback, type?: string) => {
        callback(new Blob(['jpeg'], { type }))
      }),
    } as unknown as HTMLCanvasElement
    vi.spyOn(document, 'createElement').mockReturnValue(canvas)

    const source = new File(['heic-data'], 'IMG_1234.HEIC', {
      type: 'image/heic',
      lastModified: 123,
    })
    const result = await normalizeGalleryImage(source)

    expect(result.name).toBe('IMG_1234.jpg')
    expect(result.type).toBe('image/jpeg')
    expect(result.lastModified).toBe(123)
    expect(canvas.width).toBe(1920)
    expect(canvas.height).toBe(1440)
    expect(drawImage).toHaveBeenCalledWith(expect.any(TestImage), 0, 0, 1920, 1440)
    expect(canvas.toBlob).toHaveBeenCalledWith(expect.any(Function), 'image/jpeg', 0.85)
    expect(revoke).toHaveBeenCalledWith('blob:gallery-image')
  })
})

