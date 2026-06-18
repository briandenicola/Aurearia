import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  getPreferredShareImage,
  getShareCardFilename,
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

  it('returns only approved public metadata fields for a share card', () => {
    const coin = buildRomanDenariusCore()
    const metadata = getShareCardMetadata(coin)

    expect(metadata.title).toBe(coin.name)
    expect(metadata.category).toBe(coin.category)
    expect(metadata.fields.map((field) => field.label)).toEqual([
      'Ruler',
      'Denomination',
      'Era',
      'Mint',
      'Material',
      'Grade',
    ])
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
    const ctx = buildCanvasContext({ drawImage })
    mockCanvas(ctx, toBlob)
    mockImageLoad()

    const blob = await renderCoinShareCard({
      coin: buildRomanDenariusCore(),
      imageUrl: '/uploads/coin.webp',
      appName: 'Ed-Mar Ancient Coins',
    })

    expect(blob).toBe(pngBlob)
    expect(toBlob).toHaveBeenCalledWith(expect.any(Function), 'image/png')
    expect(drawImage).toHaveBeenCalled()
  })

  it('keeps title, metadata, and footer in separate vertical zones', async () => {
    const toBlob = vi.fn((callback: BlobCallback) => callback(pngBlob))
    const fillText = vi.fn()
    const ctx = buildCanvasContext({ fillText })
    mockCanvas(ctx, toBlob)

    await renderCoinShareCard({
      coin: buildByzantineSolidusSetMember({
        name: 'Leo the Wise Extremely Long Ceremonial Byzantine Solidus',
        denomination: 'Gold Solidus with Long Denomination Text',
        ruler: 'Leo VI the Wise',
        era: 'Byzantine Middle Period',
        mint: 'Constantinople',
        grade: 'Choice Very Fine',
        images: [],
      }),
      imageUrl: null,
      appName: 'Ed-Mar Ancient Coins',
    })

    const calls = fillText.mock.calls.map(([text, _x, y]) => ({ text: String(text), y: Number(y) }))
    const footer = calls.find((call) => call.text === 'Ed-Mar Ancient Coins')
    expect(footer).toBeDefined()

    const latestContentY = Math.max(...calls.filter((call) => call.text !== 'Ed-Mar Ancient Coins').map((call) => call.y))
    expect(footer!.y - latestContentY).toBeGreaterThanOrEqual(80)
  })
})

function mockCanvas(ctx: CanvasRenderingContext2D, toBlob: ReturnType<typeof vi.fn>) {
  const originalCreateElement = document.createElement.bind(document)
  vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
    if (tagName === 'canvas') {
      return {
        width: 0,
        height: 0,
        getContext: vi.fn(() => ctx),
        toBlob,
      } as unknown as HTMLCanvasElement
    }
    return originalCreateElement(tagName)
  })
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
    clip: vi.fn(),
    drawImage: vi.fn(),
    fillText: vi.fn(),
    measureText: vi.fn((text: string) => ({ width: text.length * 18 }) as TextMetrics),
    createLinearGradient: vi.fn(() => ({
      addColorStop: vi.fn(),
    }) as unknown as CanvasGradient),
    ...overrides,
  } as unknown as CanvasRenderingContext2D
}
