import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CoinLookupPage from '../CoinLookupPage.vue'
import { createCoin, createCoinReference, lookupCoin, uploadImage } from '@/api/client'

const routerPush = vi.fn()
const routerBack = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: routerPush,
    back: routerBack,
  }),
}))

vi.mock('@/api/client', () => ({
  lookupCoin: vi.fn(),
  createCoin: vi.fn(),
  createCoinReference: vi.fn(),
  uploadImage: vi.fn(),
}))

describe('CoinLookupPage', () => {
  beforeEach(() => {
    vi.mocked(lookupCoin).mockReset()
    vi.mocked(createCoin).mockReset()
    vi.mocked(createCoinReference).mockReset()
    vi.mocked(uploadImage).mockReset()
    routerPush.mockReset()
    routerBack.mockReset()

    vi.stubGlobal('URL', {
      createObjectURL: vi.fn(() => 'blob:lookup-image'),
      revokeObjectURL: vi.fn(),
    })
  })

  it('saves lookup results only as wishlist coins', async () => {
    const file = new File(['obverse'], 'obverse.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'medium',
          rawAnalysis: '{"ruler":"Trajan"}',
          coinFields: {
            ruler: 'Trajan',
            denomination: 'Denarius',
            era: 'ancient',
            material: 'Silver',
            category: 'Roman',
          },
        },
        numistaCandidates: [],
        prefilledDraft: {
          name: 'Trajan Denarius',
          ruler: 'Trajan',
          denomination: 'Denarius',
          era: 'ancient',
          material: 'Silver',
          category: 'Roman',
        },
        candidateReferences: [
          {
            catalog: 'Numista',
            number: '12345',
            uri: 'https://en.numista.com/catalogue/pieces12345.html',
          },
        ],
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(createCoin).mockResolvedValue({ data: { id: 42 } } as Awaited<ReturnType<typeof createCoin>>)
    vi.mocked(uploadImage).mockResolvedValue({ data: {} } as Awaited<ReturnType<typeof uploadImage>>)
    vi.mocked(createCoinReference).mockResolvedValue({ data: {} } as Awaited<ReturnType<typeof createCoinReference>>)

    const wrapper = mount(CoinLookupPage, {
      global: {
        stubs: {
          CameraCaptureModal: true,
          Camera: true,
          Upload: true,
          Search: true,
          ArrowLeft: true,
          X: true,
          AlertCircle: true,
          ShieldCheck: true,
          ExternalLink: true,
          RotateCcw: true,
          Bookmark: true,
        },
      },
    })

    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [file],
      configurable: true,
    })
    await input.trigger('change')

    await wrapper.find('.btn-submit').trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('Add to Collection')
    expect(wrapper.text()).toContain('Save to Wishlist')

    const actionButtons = wrapper.findAll('.result-actions button')
    expect(actionButtons).toHaveLength(3)
    await actionButtons[2]!.trigger('click')
    await flushPromises()

    expect(createCoin).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Trajan Denarius',
      category: 'Roman',
      material: 'Silver',
      era: 'ancient',
      isWishlist: true,
    }))
    expect(uploadImage).toHaveBeenCalledWith(42, file, 'obverse', true, false)
    expect(createCoinReference).toHaveBeenCalledWith(42, {
      catalog: 'Numista',
      number: '12345',
      uri: 'https://en.numista.com/catalogue/pieces12345.html',
    })
    expect(routerPush).toHaveBeenCalledWith('/wishlist')
  })

  it('lets the user cancel results without saving', async () => {
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'medium',
          rawAnalysis: 'uncertain',
        },
        numistaCandidates: [],
        prefilledDraft: {
          name: 'Possible drachm',
        },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)

    const wrapper = mount(CoinLookupPage, {
      global: {
        stubs: {
          CameraCaptureModal: true,
          Camera: true,
          Upload: true,
          Search: true,
          ArrowLeft: true,
          X: true,
          AlertCircle: true,
          ShieldCheck: true,
          ExternalLink: true,
          RotateCcw: true,
          Bookmark: true,
        },
      },
    })

    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [new File(['coin'], 'coin.jpg', { type: 'image/jpeg' })],
      configurable: true,
    })
    await input.trigger('change')

    await wrapper.find('.btn-submit').trigger('click')
    await flushPromises()

    const cancel = wrapper.findAll('button').find(button => button.text().includes('Cancel'))
    expect(cancel).toBeDefined()
    await cancel?.trigger('click')

    expect(createCoin).not.toHaveBeenCalled()
    expect(routerBack).toHaveBeenCalled()
  })
})
