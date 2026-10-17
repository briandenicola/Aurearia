import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  getPreferredShareImage,
  getShareCardFilename,
  getShareImageUrls,
  getShareCardMetadata,
  renderCoinShareCard,
} from '@/utils/coinShareCard'
import { buildByzantineSolidusSetMember, buildImageHeavyDrachm, buildRomanDenariusCore } from '@/test/fixtures/coins'
import type { CoinImage } from '@/types'

const pngBlob = new Blob(['png'], { type: 'image/png' })

function flattenMetadata(coin = buildRomanDenariusCore()) {
  return JSON.stringify(getShareCardMetadata(coin))
}

function makeImage(id: number, imageType: CoinImage['imageType'], isPrimary = false): CoinImage {
  return {
    id,
    coinId: 77,
    filePath: `coins/${id}.webp`,
    imageType,
    isPrimary,
    createdAt: '2026-01-01T00:00:00Z',
  }
}

describe('coinShareCard', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('builds the collector ticket from stored attribution without inventing dates or references', () => {
    const metadata = getShareCardMetadata(buildRomanDenariusCore({ dateRange: 'AD 98-117' }))

    expect(metadata).toEqual({
      title: 'Trajan, AD 98-117',
      details: ['Rome mint. Silver Denarius.', '3.35 g; 18 mm.'],
      obverse: 'IMP TRAIANO AVG GER DAC P M TR P. Laureate bust right',
      reverse: 'SPQR OPTIMO PRINCIPI. Victory standing left',
      references: 'RIC II Trajan 147',
    })
  })

  it('formats structured catalogue references, deduplicates them, and excludes source URLs', () => {
    const reference = {
      id: 1, coinId: 1001, catalog: 'RIC', volume: 'II', number: '147',
      uri: 'https://example.test/private-source', createdAt: '', updatedAt: '',
    }
    const metadata = getShareCardMetadata(buildRomanDenariusCore({
      references: [reference, { ...reference, id: 2 }, { ...reference, id: 3, catalog: 'BMC', volume: '', number: '42' }],
    }))

    expect(metadata.references).toBe('RIC II 147; BMC 42')
    expect(JSON.stringify(metadata)).not.toContain('private-source')
  })

  it('omits missing attribution and invalid measurements and falls back to the coin name', () => {
    const metadata = getShareCardMetadata(buildRomanDenariusCore({
      name: 'Unattributed coin', ruler: '', dateRange: null, mint: '', denomination: '',
      weightGrams: Number.NaN, diameterMm: -1,
      obverseInscription: '', obverseDescription: '', reverseInscription: '', reverseDescription: '',
      references: [], referenceText: '',
    }))

    expect(metadata).toEqual({
      title: 'Unattributed coin', details: ['Silver.'], obverse: '', reverse: '', references: '',
    })
    expect(getShareCardMetadata(buildRomanDenariusCore({ name: '', ruler: '' })).title).toBe('Untitled Coin')
    expect(getShareCardMetadata(buildRomanDenariusCore({ weightGrams: null, diameterMm: null })).details)
      .toEqual(['Rome mint. Silver Denarius.'])
    expect(getShareCardMetadata(buildRomanDenariusCore({ mint: 'Rome mint' })).details[0])
      .toBe('Rome mint. Silver Denarius.')
  })

  it('excludes value, purchase, notes, AI, owner, listing, tag, set, and privacy fields', () => {
    const coin = buildRomanDenariusCore({
      purchasePrice: 999,
      currentValue: 1200,
      purchaseLocation: 'Secret Dealer',
      notes: 'Private owner note',
      aiAnalysis: 'AI private analysis',
      listingStatus: 'available',
      userId: 42,
      isPrivate: true,
    })
    const text = flattenMetadata(coin)

    expect(text).not.toContain('999')
    expect(text).not.toContain('1200')
    expect(text).not.toContain('Secret Dealer')
    expect(text).not.toContain('Private owner note')
    expect(text).not.toContain('AI private analysis')
    expect(text).not.toContain('available')
    expect(text).not.toContain('userId')
    expect(text).not.toContain('isPrivate')
    expect(text).not.toContain('tags')
    expect(text).not.toContain('sets')
  })

  it('prefers obverse image for sharing', () => {
    expect(getPreferredShareImage(buildImageHeavyDrachm())).toBe('/uploads/test-fixtures/1008-obverse-10081.webp')
  })

  it('uses obverse and reverse images together for the share card', () => {
    expect(getShareImageUrls(buildRomanDenariusCore())).toEqual([
      '/uploads/test-fixtures/1001-obverse-10011.webp',
      '/uploads/test-fixtures/1001-reverse-10012.webp',
    ])
  })

  it('falls back to primary, first image, and no image in order', () => {
    const primaryOnly = buildRomanDenariusCore({
      images: [makeImage(1, 'reverse', true), makeImage(2, 'other')],
    })
    const firstOnly = buildRomanDenariusCore({
      images: [makeImage(3, 'detail'), makeImage(4, 'other')],
    })
    const noImages = buildRomanDenariusCore({ images: [] })

    expect(getPreferredShareImage(primaryOnly)).toBe('/uploads/coins/1.webp')
    expect(getPreferredShareImage(firstOnly)).toBe('/uploads/coins/3.webp')
    expect(getPreferredShareImage(noImages)).toBeNull()
  })

  it('generates a safe share-card filename', () => {
    const coin = buildRomanDenariusCore({ name: 'Trajan: Denarius / Rare?' })

    expect(getShareCardFilename(coin)).toBe('trajan-denarius-rare-share-card.png')
  })

  it('renders a PNG blob with a loaded coin image', async () => {
    const toBlob = vi.fn((callback: BlobCallback) => callback(pngBlob))
    const drawImage = vi.fn()
    const arc = vi.fn()
    const ctx = buildCanvasContext({ drawImage, arc })
    const createObjectURL = vi.fn()
      .mockReturnValueOnce('blob:obverse-image')
      .mockReturnValueOnce('blob:reverse-image')
    const revokeObjectURL = vi.fn()
    const fetchMock = vi.fn(async () => new Response(new Blob(['image'], { type: 'image/webp' }), { status: 200 }))
    class TestURL extends URL {
      static createObjectURL = createObjectURL
      static revokeObjectURL = revokeObjectURL
    }

    mockCanvas(ctx, toBlob)
    mockImageLoad()
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('URL', TestURL)

    const blob = await renderCoinShareCard({
      coin: buildRomanDenariusCore(),
      imageUrl: '/uploads/coin.webp',
      imageUrls: ['/uploads/obverse.webp', '/uploads/reverse.webp'],
      appName: 'Aurearia - Coin Collection',
    })

    expect(blob).toBe(pngBlob)
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(fetchMock).toHaveBeenCalledWith('/api/uploads/obverse.webp', {
      headers: expect.any(Headers),
      cache: 'no-store',
    })
    expect(fetchMock).toHaveBeenCalledWith('/api/uploads/reverse.webp', {
      headers: expect.any(Headers),
      cache: 'no-store',
    })
    expect(createObjectURL).toHaveBeenCalledTimes(2)
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:obverse-image')
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:reverse-image')
    expect(toBlob).toHaveBeenCalledWith(expect.any(Function), 'image/png')
    expect(drawImage).toHaveBeenCalledTimes(2)
    expect(arc).not.toHaveBeenCalled()
  })

  it('renders the approved dark image half and cream, left-aligned serif attribution half', async () => {
    const fillRect = vi.fn()
    const fillText = vi.fn()
    const ctx = buildCanvasContext({ fillRect, fillText })
    const textStyles: { text: string; font: string; align: string; color: string }[] = []
    fillText.mockImplementation((text: string) => {
      textStyles.push({ text, font: ctx.font, align: ctx.textAlign, color: String(ctx.fillStyle) })
    })
    const canvas = mockCanvas(ctx, vi.fn((callback: BlobCallback) => callback(pngBlob)))

    await renderCoinShareCard({
      coin: buildRomanDenariusCore({ images: [] }),
      imageUrl: null,
      appName: 'Aurearia - Coin Collection',
    })

    expect(canvas.width).toBe(1080)
    expect(canvas.height).toBeGreaterThanOrEqual(1440)
    expect(fillRect.mock.calls).toEqual([
      [0, 0, 1080, 672],
      [0, 672, 1080, canvas.height - 672],
    ])
    expect(textStyles).toContainEqual({ text: 'Trajan', font: '400 52px Georgia, serif', align: 'left', color: '#322a20' })
    const attribution = fillText.mock.calls.filter(([, x, y]) => Number(x) === 64 && Number(y) >= 728)
    expect(attribution.map(([text]) => String(text)).join(' ')).toContain('Obv. IMP TRAIANO')
    expect(attribution.map(([text]) => String(text)).join(' ')).toContain('Rev. SPQR OPTIMO PRINCIPI.')
    expect(attribution.map(([text]) => String(text)).join(' ')).toContain('References: RIC II Trajan 147')
  })

  it('grows for long attribution without truncating text or overlapping the footer', async () => {
    const toBlob = vi.fn((callback: BlobCallback) => callback(pngBlob))
    const fillText = vi.fn()
    const ctx = buildCanvasContext({ fillText })
    const canvas = mockCanvas(ctx, toBlob)
    const longDescription = `${'Victory holding a wreath and palm branch. '.repeat(30)}Final attribution detail.`

    await renderCoinShareCard({
      coin: buildByzantineSolidusSetMember({
        name: 'Leo the Wise Extremely Long Ceremonial Byzantine Solidus',
        denomination: 'Gold Solidus with Long Denomination Text',
        ruler: 'Leo VI the Wise',
        era: 'Byzantine Middle Period',
        mint: 'Constantinople',
        grade: 'Choice Very Fine',
        reverseDescription: longDescription,
        images: [],
      }),
      imageUrl: null,
      appName: 'Aurearia - Coin Collection',
    })

    const calls = fillText.mock.calls.map(([text, _x, y]) => ({ text: String(text), y: Number(y) }))
    const footer = calls.find((call) => call.text === 'Aurearia - Coin Collection')
    expect(footer).toBeDefined()

    const latestContentY = Math.max(...calls.filter((call) => !['Aurearia - Coin Collection', 'ATTRIBUTION'].includes(call.text)).map((call) => call.y))
    expect(footer!.y - latestContentY).toBeGreaterThanOrEqual(80)
    expect(canvas.height).toBeGreaterThan(1440)
    const renderedText = calls.map((call) => call.text).join(' ')
    expect(renderedText).toContain(longDescription)
    expect(renderedText).not.toContain('...')
  })

  it('wraps long unbroken inscriptions within the ticket width without losing characters', async () => {
    const fillText = vi.fn()
    const ctx = buildCanvasContext({ fillText })
    mockCanvas(ctx, vi.fn((callback: BlobCallback) => callback(pngBlob)))
    const inscription = 'X'.repeat(180)

    await renderCoinShareCard({
      coin: buildRomanDenariusCore({ images: [], obverseInscription: inscription, obverseDescription: '' }),
      imageUrl: null,
      appName: 'Aurearia - Coin Collection',
    })

    const lines = fillText.mock.calls
      .filter(([text]) => String(text).includes('X'))
      .map(([text]) => String(text))
    expect(lines.join('').replace('Obv. ', '')).toBe(`${inscription}.`)
    expect(lines.every((line) => ctx.measureText(line).width <= 952)).toBe(true)
  })

  it('keeps a reverse-only image on the right, uncropped, with the missing obverse labelled', async () => {
    const toBlob = vi.fn((callback: BlobCallback) => callback(pngBlob))
    const fillText = vi.fn()
    const drawImage = vi.fn()
    const ctx = buildCanvasContext({ fillText, drawImage })
    mockCanvas(ctx, toBlob)
    mockImageLoad()
    vi.stubGlobal('fetch', vi.fn(async () => new Response(new Blob(['image']), { status: 200 })))
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:reverse-image'),
      revokeObjectURL: vi.fn(),
    })
    const reverse = makeImage(2, 'reverse')

    await renderCoinShareCard({
      coin: buildRomanDenariusCore({ images: [reverse] }),
      imageUrl: 'data:image/png;base64,reverse',
      imageUrls: ['/uploads/coins/2.webp'],
      appName: 'Aurearia - Coin Collection',
    })

    expect(drawImage).toHaveBeenCalledWith(expect.anything(), 574, 128, 432, 432)
    expect(fillText).toHaveBeenCalledWith('No obverse image', 290, 332)
    expect(fillText).not.toHaveBeenCalledWith('No reverse image', expect.anything(), expect.anything())
  })

  it('scales the whole layout for custom output dimensions', async () => {
    const ctx = buildCanvasContext()
    const canvas = mockCanvas(ctx, vi.fn((callback: BlobCallback) => callback(pngBlob)))
    await renderCoinShareCard({
      coin: buildRomanDenariusCore({ images: [] }),
      imageUrl: null,
      appName: 'Aurearia',
    }, { width: 540, height: 720 })

    expect(canvas.width).toBe(540)
    expect(canvas.height).toBe(720)
    expect(ctx.scale).toHaveBeenCalledWith(0.5, expect.any(Number))
  })

  it('preserves Coin of the Day context alongside the full attribution', async () => {
    const toBlob = vi.fn((callback: BlobCallback) => callback(pngBlob))
    const fillText = vi.fn()
    const ctx = buildCanvasContext({ fillText })
    mockCanvas(ctx, toBlob)

    await renderCoinShareCard({
      coin: buildRomanDenariusCore({ images: [] }),
      imageUrl: null,
      appName: 'Aurearia - Coin Collection',
      context: {
        heading: 'Coin of the Day',
        summary: '**Obverse:** laureate portrait. **Reverse:** Victory holding wreath.',
      },
    })

    const calls = fillText.mock.calls.map(([text]) => String(text))
    expect(calls).toContain('Coin of the Day')
    expect(calls.join(' ')).toContain('Obverse: laureate portrait. Reverse: Victory holding wreath.')
    expect(calls.join(' ')).toContain('Obv. IMP TRAIANO AVG GER DAC P M TR P.')
    expect(calls.join(' ')).toContain('References: RIC II Trajan 147')
  })

  it('reports unavailable canvas and failed PNG generation instead of returning a blank success', async () => {
    const ctx = buildCanvasContext()
    const canvas = mockCanvas(ctx, vi.fn((callback: BlobCallback) => callback(null)))
    const input = { coin: buildRomanDenariusCore({ images: [] }), imageUrl: null, appName: 'Aurearia' }

    await expect(renderCoinShareCard(input)).rejects.toThrow('Unable to generate share card image.')
    canvas.getContext.mockReturnValueOnce(null)
    await expect(renderCoinShareCard(input)).rejects.toThrow('Canvas rendering is not available')
  })
})

function mockCanvas(ctx: CanvasRenderingContext2D, toBlob: ReturnType<typeof vi.fn>) {
  const canvas = {
    width: 0,
    height: 0,
    getContext: vi.fn<() => CanvasRenderingContext2D | null>(() => ctx),
    toBlob,
  }
  const originalCreateElement = document.createElement.bind(document)
  vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
    if (tagName === 'canvas') {
      return canvas as unknown as HTMLCanvasElement
    }
    return originalCreateElement(tagName)
  })
  return canvas
}

function mockImageLoad() {
  vi.stubGlobal('Image', class {
    onload: (() => void) | null = null
    onerror: (() => void) | null = null
    naturalWidth = 400
    naturalHeight = 400
    width = 400
    height = 400
    set src(_value: string) {
      this.onload?.()
    }
  })
}

function buildCanvasContext(overrides: Partial<CanvasRenderingContext2D> = {}): CanvasRenderingContext2D {
  return {
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 1,
    font: '',
    textAlign: 'start',
    textBaseline: 'alphabetic',
    fillRect: vi.fn(),
    beginPath: vi.fn(),
    arc: vi.fn(),
    fill: vi.fn(),
    stroke: vi.fn(),
    save: vi.fn(),
    restore: vi.fn(),
    scale: vi.fn(),
    clip: vi.fn(),
    moveTo: vi.fn(),
    lineTo: vi.fn(),
    quadraticCurveTo: vi.fn(),
    closePath: vi.fn(),
    drawImage: vi.fn(),
    fillText: vi.fn(),
    measureText: vi.fn((text: string) => ({ width: text.length * 18 }) as TextMetrics),
    createLinearGradient: vi.fn(() => ({
      addColorStop: vi.fn(),
    }) as unknown as CanvasGradient),
    ...overrides,
  } as unknown as CanvasRenderingContext2D
}
