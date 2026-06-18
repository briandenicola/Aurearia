import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { useCoinShareCard } from '@/composables/useCoinShareCard'
import { buildRomanDenariusCore } from '@/test/fixtures/coins'
import { renderCoinShareCard } from '@/utils/coinShareCard'

const showAlert = vi.fn()

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
  }),
}))

describe('useCoinShareCard', () => {
  const blob = new Blob(['png'], { type: 'image/png' })

  beforeEach(() => {
    vi.mocked(renderCoinShareCard).mockReset()
    vi.mocked(renderCoinShareCard).mockResolvedValue(blob)
    showAlert.mockReset()
    showAlert.mockResolvedValue(true)
    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:share-card'),
      revokeObjectURL: vi.fn(),
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('uses native Web Share when file sharing is supported', async () => {
    const share = vi.fn().mockResolvedValue(undefined)
    const canShare = vi.fn(() => true)
    vi.stubGlobal('navigator', { share, canShare })
    const { shareCoinCard, sharing } = useCoinShareCard()

    const result = await shareCoinCard(buildRomanDenariusCore())

    expect(result).toEqual({ mode: 'shared' })
    expect(canShare).toHaveBeenCalledWith(expect.objectContaining({
      files: [expect.any(File)],
    }))
    expect(share).toHaveBeenCalledWith(expect.objectContaining({
      files: [expect.any(File)],
      title: 'Trajan Denarius Core',
    }))
    expect(sharing.value).toBe(false)
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
    const appendChild = vi.spyOn(document.body, 'appendChild').mockImplementation((node: Node) => node)
    vi.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
      if (tagName === 'a') return anchor
      return document.createElement(tagName)
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
})
