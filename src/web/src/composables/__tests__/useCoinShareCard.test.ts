import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { useCoinShareCard } from '@/composables/useCoinShareCard'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'
import { renderCoinShareCard } from '@/utils/coinShareCard'

const showAlert = vi.fn()
const showConfirm = vi.fn()
const { getAttributionCards, fetchPrivateMediaBlob } = vi.hoisted(() => ({
  getAttributionCards: vi.fn(),
  fetchPrivateMediaBlob: vi.fn(),
}))

vi.mock('@/api/client', () => ({ getAttributionCards }))
vi.mock('@/utils/media', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/utils/media')>(),
  fetchPrivateMediaBlob,
}))

vi.mock('@/utils/coinShareCard', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/coinShareCard')>()
  return {
    ...actual,
    renderCoinShareCard: vi.fn(),
  }
})

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showAlert,
    showConfirm,
  }),
}))

describe('useCoinShareCard', () => {
  const blob = new Blob(['png'], { type: 'image/png' })

  beforeEach(() => {
    vi.mocked(renderCoinShareCard).mockReset()
    vi.mocked(renderCoinShareCard).mockResolvedValue(blob)
    showAlert.mockReset()
    showAlert.mockResolvedValue(true)
    showConfirm.mockReset()
    showConfirm.mockResolvedValue(false)
    vi.mocked(getAttributionCards).mockReset()
    vi.mocked(getAttributionCards).mockResolvedValue({ data: [] })
    vi.mocked(fetchPrivateMediaBlob).mockReset()
    vi.mocked(fetchPrivateMediaBlob).mockResolvedValue(new Blob(['original'], { type: 'image/png' }))
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:share-card'),
      revokeObjectURL: vi.fn(),
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('uses native Web Share when file sharing is supported', async () => {
    const share = vi.fn().mockResolvedValue(undefined)
    const canShare = vi.fn(() => true)
    vi.stubGlobal('navigator', { share, canShare })
    const { shareCoinCard, sharing } = useCoinShareCard()

    const result = await shareCoinCard(buildRomanDenariusCore())

    expect(result).toEqual({ mode: 'shared' })
    expect(renderCoinShareCard).toHaveBeenCalledWith(expect.objectContaining({
      imageUrls: [
        '/uploads/test-fixtures/1001-obverse-10011.webp',
        '/uploads/test-fixtures/1001-reverse-10012.webp',
      ],
    }))
    expect(canShare).toHaveBeenCalledWith(expect.objectContaining({
      files: [expect.any(File)],
    }))
    expect(share).toHaveBeenCalledWith(expect.objectContaining({
      files: [expect.any(File)],
      title: 'Trajan Denarius Core',
    }))
    expect(sharing.value).toBe(false)
    expect(showConfirm).not.toHaveBeenCalled()
  })

  it('passes optional share context into the rendered card and native share text', async () => {
    const share = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { share, canShare: vi.fn(() => true) })
    const { shareCoinCard } = useCoinShareCard()

    const obverseReverseSummary = 'Obverse: laureate portrait of Trajan. Reverse: Victory holding wreath.'

    await shareCoinCard(buildRomanDenariusCore(), {
      context: {
        heading: 'Coin of the Day',
        summary: obverseReverseSummary,
      },
    })

    expect(renderCoinShareCard).toHaveBeenCalledWith(expect.objectContaining({
      context: {
        heading: 'Coin of the Day',
        summary: obverseReverseSummary,
      },
    }))
    expect(share).toHaveBeenCalledWith(expect.objectContaining({
      text: `${obverseReverseSummary}\n\nShared from Aurearia - Coin Collection`,
    }))
  })

  it('downloads the generated card when file sharing is unsupported', async () => {
    const click = vi.fn()
    const remove = vi.fn()
    const anchor = {
      href: '',
      download: '',
      rel: '',
      click,
      remove,
    } as HTMLAnchorElement
    const originalCreateElement = document.createElement.bind(document)
    const appendChild = vi.spyOn(document.body, 'appendChild').mockImplementation((node: Node) => node)
    vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
      if (tagName === 'a') return anchor
      return originalCreateElement(tagName)
    })
    vi.stubGlobal('navigator', { canShare: vi.fn(() => false) })
    const { shareCoinCard } = useCoinShareCard()

    const result = await shareCoinCard(buildRomanDenariusCore({ name: 'Trajan: Denarius' }))

    expect(result).toEqual({ mode: 'downloaded' })
    expect(anchor.href).toBe('blob:share-card')
    expect(anchor.download).toBe('trajan-denarius-share-card.png')
    expect(click).toHaveBeenCalled()
    expect(remove).toHaveBeenCalled()
    expect(appendChild).toHaveBeenCalledWith(anchor)
    expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:share-card')
  })

  it('shows an alert and resets sharing state when generation fails', async () => {
    vi.mocked(renderCoinShareCard).mockRejectedValue(new Error('Canvas failed'))
    vi.stubGlobal('navigator', {})
    const { shareCoinCard, sharing } = useCoinShareCard()
    const promise = shareCoinCard(buildRomanDenariusCore())
    await nextTick()
    expect(sharing.value).toBe(true)

    await expect(promise).rejects.toThrow('Canvas failed')

    expect(showAlert).toHaveBeenCalledWith('Canvas failed', { title: 'Share Failed' })
    expect(sharing.value).toBe(false)
  })

  const cards = [
    { id: 71, coinId: 1001, filePath: 'coin-1001/71-card.jpg', imageType: 'card' as const, isPrimary: false, createdAt: '' },
    { id: 72, coinId: 1001, filePath: 'coin-1001/72-card.png', imageType: 'card' as const, isPrimary: false, createdAt: '' },
  ]

  it('includes every unchanged original as a separate native-share attachment only after consent', async () => {
    vi.mocked(getAttributionCards).mockResolvedValue({ data: cards })
    showConfirm.mockResolvedValue(true)
    vi.mocked(fetchPrivateMediaBlob)
      .mockResolvedValueOnce(new Blob(['original JPEG'], { type: 'image/jpeg' }))
      .mockResolvedValueOnce(new Blob(['original PNG'], { type: 'image/png' }))
    const share = vi.fn().mockResolvedValue(undefined)
    const canShare = vi.fn(() => true)
    vi.stubGlobal('navigator', { share, canShare })
    const { shareCoinCard, sharing } = useCoinShareCard()

    await shareCoinCard(buildRomanDenariusCore())

    expect(getAttributionCards).toHaveBeenCalledWith(1001)
    expect(showConfirm).toHaveBeenCalledWith(expect.stringContaining('handwritten notes'), expect.objectContaining({
      confirmLabel: 'Include originals', cancelLabel: 'Generated card only',
    }))
    expect(fetchPrivateMediaBlob).toHaveBeenNthCalledWith(1, cards[0]!.filePath)
    expect(fetchPrivateMediaBlob).toHaveBeenNthCalledWith(2, cards[1]!.filePath)
    const files: File[] = share.mock.calls[0]?.[0].files
    expect(files.map(file => [file.name, file.type])).toEqual([
      ['trajan-denarius-core-share-card.png', 'image/png'],
      ['trajan-denarius-core-original-attribution-71.jpg', 'image/jpeg'],
      ['trajan-denarius-core-original-attribution-72.png', 'image/png'],
    ])
    expect(canShare).toHaveBeenCalledWith(expect.objectContaining({ files }))
    for (const [index, expected] of ['original JPEG', 'original PNG'].entries()) {
      expect(await files[index + 1]!.text()).toBe(expected)
    }
    expect(sharing.value).toBe(false)
  })

  it('does not fetch or attach originals when the owner chooses generated card only', async () => {
    vi.mocked(getAttributionCards).mockResolvedValue({ data: cards })
    const share = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { share, canShare: vi.fn(() => true) })

    await useCoinShareCard().shareCoinCard(buildRomanDenariusCore())

    expect(showConfirm).toHaveBeenCalledTimes(1)
    expect(fetchPrivateMediaBlob).not.toHaveBeenCalled()
    expect(share.mock.calls[0]?.[0].files).toHaveLength(1)
  })

  it('fails visibly instead of sharing a partial packet when an original cannot be retrieved', async () => {
    vi.mocked(getAttributionCards).mockResolvedValue({ data: cards })
    showConfirm.mockResolvedValue(true)
    vi.mocked(fetchPrivateMediaBlob).mockRejectedValue(new Error('Original unavailable'))
    const share = vi.fn()
    vi.stubGlobal('navigator', { share, canShare: vi.fn(() => true) })
    const { shareCoinCard, sharing } = useCoinShareCard()

    await expect(shareCoinCard(buildRomanDenariusCore())).rejects.toThrow('Original unavailable')

    expect(share).not.toHaveBeenCalled()
    expect(URL.createObjectURL).not.toHaveBeenCalled()
    expect(showAlert).toHaveBeenCalledWith('Original unavailable', { title: 'Share Failed' })
    expect(sharing.value).toBe(false)
  })

  it('does not silently treat a failed private card listing as an empty list', async () => {
    vi.mocked(getAttributionCards).mockRejectedValue(new Error('Cards unavailable'))
    const { shareCoinCard } = useCoinShareCard()
    await expect(shareCoinCard(buildRomanDenariusCore())).rejects.toThrow('Cards unavailable')
    expect(renderCoinShareCard).not.toHaveBeenCalled()
    expect(showAlert).toHaveBeenCalledWith('Cards unavailable', { title: 'Share Failed' })
  })

  it('downloads the generated PNG and every opted-in original separately on unsupported browsers', async () => {
    vi.mocked(getAttributionCards).mockResolvedValue({ data: cards })
    showConfirm.mockResolvedValue(true)
    vi.mocked(fetchPrivateMediaBlob).mockResolvedValue(new Blob(['original'], { type: 'image/png' }))
    const filenames: string[] = []
    const anchor = {
      href: '', download: '', rel: '',
      click: () => filenames.push(anchor.download), remove: vi.fn(),
    } as HTMLAnchorElement
    const createElement = document.createElement.bind(document)
    vi.spyOn(document.body, 'appendChild').mockImplementation((node: Node) => node)
    vi.spyOn(document, 'createElement').mockImplementation(tag => tag === 'a' ? anchor : createElement(tag))
    vi.stubGlobal('navigator', { canShare: vi.fn(() => false) })

    expect(await useCoinShareCard().shareCoinCard(buildRomanDenariusCore())).toEqual({ mode: 'downloaded' })

    expect(filenames).toEqual([
      'trajan-denarius-core-share-card.png',
      'trajan-denarius-core-original-attribution-71.jpg',
      'trajan-denarius-core-original-attribution-72.png',
    ])
    expect(showAlert).toHaveBeenCalledWith(expect.stringContaining('Allow multiple downloads'), { title: 'Download share files' })
    expect(URL.createObjectURL).toHaveBeenCalledTimes(3)
    expect(URL.revokeObjectURL).toHaveBeenCalledTimes(3)
    expect(anchor.remove).toHaveBeenCalledTimes(3)
  })
})
