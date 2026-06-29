import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import CoinLookupPage from '../CoinLookupPage.vue'
import { createQuickCaptureDraft, lookupCoin } from '@/api/client'

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
  createQuickCaptureDraft: vi.fn(),
}))

describe('CoinLookupPage', () => {
  beforeEach(() => {
    vi.mocked(lookupCoin).mockReset()
    vi.mocked(createQuickCaptureDraft).mockReset()
    routerPush.mockReset()
    routerBack.mockReset()

    Object.defineProperty(URL, 'createObjectURL', {
      value: vi.fn(() => 'blob:lookup-image'),
      configurable: true,
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      value: vi.fn(),
      configurable: true,
    })
  })

  it('saves lookup results as a quick capture draft', async () => {
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
          obverseDescription: 'Laureate bust of Trajan right',
          reverseDescription: 'Victory standing left',
          notes: 'Well-preserved example',
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
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 42 } } as Awaited<ReturnType<typeof createQuickCaptureDraft>>)

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

    // No NGC cert, so should show editable review form
    expect(wrapper.text()).toContain('Review Coin Details')
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('Add to Collection')
    expect(wrapper.text()).toContain('Save as Draft')

    const nameInput = wrapper.find('input[type="text"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('Trajan Denarius')

    const actionButtons = wrapper.findAll('.result-actions button')
    expect(actionButtons).toHaveLength(3)
    await actionButtons[2]!.trigger('click')
    await flushPromises()

    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      workingTitle: 'Trajan Denarius',
      era: 'ancient',
      notes: expect.stringContaining('Laureate bust of Trajan right'),
      source: 'find_coin_ai',
      obverseImage: file,
      reverseImage: null,
    }))
    expect(routerPush).toHaveBeenCalledWith('/quick-capture/drafts/42')
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

    expect(createQuickCaptureDraft).not.toHaveBeenCalled()
    expect(routerBack).toHaveBeenCalled()
  })

  it('shows read-only details when NGC cert is detected', async () => {
    const file = new File(['slab'], 'slab.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'high',
          rawAnalysis: 'NGC cert detected',
          ngc: {
            certNumber: '1234567-001',
            normalizedCert: '1234567001',
            lookupURL: 'https://www.ngccoin.com/certlookup/1234567001/NGCAncients/',
            grade: 'Ch VF',
            description: 'Augustus Denarius',
          },
        },
        numistaCandidates: [],
        prefilledDraft: {
          name: 'Augustus Denarius',
          ruler: 'Augustus',
          denomination: 'Denarius',
          era: 'ancient',
          material: 'Silver',
          category: 'Roman',
          grade: 'Ch VF',
        },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 99 } } as Awaited<ReturnType<typeof createQuickCaptureDraft>>)

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

    // NGC path should show read-only display
    expect(wrapper.text()).toContain('Extracted Details')
    expect(wrapper.text()).toContain('NGC Certification: 1234567001')
    expect(wrapper.text()).toContain('Verify on NGC')

    // Should NOT show the full editable coin form, but the captured NGC number remains editable.
    expect(wrapper.text()).not.toContain('Review Coin Details')
    expect(wrapper.findAll('input[type="text"]')).toHaveLength(1)
    expect((wrapper.find('input[type="text"]').element as HTMLInputElement).value).toBe('1234567001')
    const actionButtons = wrapper.findAll('.result-actions button')
    await actionButtons[2]!.trigger('click')
    await flushPromises()

    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      workingTitle: 'Augustus Denarius',
      source: 'find_coin_ai',
      ngcCertNumber: '1234567001',
      ngcLookupUrl: 'https://www.ngccoin.com/certlookup/1234567001/NGCAncients/',
      ngcGrade: 'Ch VF',
      obverseImage: file,
    }))
  })

  it('renders only safe external lookup links from API results', async () => {
    const file = new File(['coin'], 'coin.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'medium',
          rawAnalysis: 'candidate matches',
        },
        numistaCandidates: [
          { id: 'js', title: 'Script', issuer: 'Bad', year: '', url: 'javascript:alert(1)' },
          { id: 'data', title: 'Data', issuer: 'Bad', year: '', url: 'data:text/html,<p>x</p>' },
          { id: 'relative', title: 'Relative', issuer: 'Bad', year: '', url: '/catalogue/pieces1.html' },
          { id: 'http', title: 'HTTP', issuer: 'OK', year: '', url: 'http://example.com/pieces1.html' },
          { id: 'https', title: 'HTTPS', issuer: 'OK', year: '', url: 'https://example.com/pieces2.html' },
        ],
        prefilledDraft: {
          name: 'Lookup candidate',
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
      value: [file],
      configurable: true,
    })
    await input.trigger('change')

    await wrapper.find('.btn-submit').trigger('click')
    await flushPromises()

    const links = wrapper.findAll('a.numista-link')
    expect(links.map(link => link.attributes('href'))).toEqual([
      'http://example.com/pieces1.html',
      'https://example.com/pieces2.html',
    ])
    expect(wrapper.html()).not.toContain('javascript:alert')
    expect(wrapper.html()).not.toContain('data:text/html')
    expect(wrapper.html()).not.toContain('/catalogue/pieces1.html')
  })
})
