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
    expect(wrapper.text()).toContain('AI Observations')
    expect(wrapper.text()).not.toContain('Obverse Description')
    expect(wrapper.text()).not.toContain('Reverse Description')
    expect(wrapper.findAll('textarea')).toHaveLength(0)
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

  it('renders safe AI observations narrative instead of editable side description boxes', async () => {
    const file = new File(['obverse'], 'observations.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'medium',
          rawAnalysis: 'AI saw a silver denarius.',
        },
        numistaCandidates: [],
        prefilledDraft: {
          name: 'Trajan Denarius',
          ruler: 'Trajan',
          denomination: 'Denarius',
          category: 'Roman',
          grade: 'VF',
          obverseDescription: 'Laureate bust of Trajan right',
          reverseDescription: 'Victory standing left',
          notes: '**Observed:** Laureate bust of Trajan right\n\n<script>alert("x")</script>',
        },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 45 } } as Awaited<ReturnType<typeof createQuickCaptureDraft>>)

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

    expect(wrapper.text()).toContain('AI Observations')
    expect(wrapper.find('.markdown-rendered strong').text()).toBe('Observed:')
    expect(wrapper.text()).toContain('AI saw a silver denarius.')
    expect(wrapper.html()).not.toContain('<script>')
    expect(wrapper.text()).toContain('Victory standing left')
    expect(wrapper.text()).not.toContain('Obverse Description')
    expect(wrapper.text()).not.toContain('Reverse Description')
    expect(wrapper.findAll('textarea')).toHaveLength(0)

    const inputs = wrapper.findAll('input[type="text"]')
    expect((inputs[0]!.element as HTMLInputElement).value).toBe('Trajan Denarius')
    expect((inputs[1]!.element as HTMLInputElement).value).toBe('Trajan')
    expect((inputs[2]!.element as HTMLInputElement).value).toBe('Denarius')
    expect((inputs[3]!.element as HTMLInputElement).value).toBe('Roman')
    expect((inputs[4]!.element as HTMLInputElement).value).toBe('VF')

    const actionButtons = wrapper.findAll('.result-actions button')
    await actionButtons[2]!.trigger('click')
    await flushPromises()

    const payload = vi.mocked(createQuickCaptureDraft).mock.calls[0]?.[0]
    expect(payload?.notes).toContain('**Extracted fields**')
    expect(payload?.notes).toContain('Ruler: Trajan')
    expect(payload?.notes).toContain('Denomination: Denarius')
    expect(payload?.notes).toContain('Category: Roman')
    expect(payload?.notes).toContain('Grade: VF')
    expect(payload?.notes).toContain('**Observed:** Laureate bust of Trajan right')
    expect(payload?.notes).toContain('**Reverse:** Victory standing left')
    expect(payload?.notes?.match(/Laureate bust of Trajan right/g)).toHaveLength(1)
    expect(payload?.source).toBe('find_coin_ai')
  })

  it('promotes missing review fields from extracted coin fields before saving', async () => {
    const file = new File(['obverse'], 'coin-fields.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'medium',
          rawAnalysis: 'coin fields extracted',
          coinFields: {
            Name: 'Julia Domna Denarius',
            Ruler: 'Julia Domna',
            Denomination: 'Denarius',
            Category: 'Roman',
          },
        },
        numistaCandidates: [],
        prefilledDraft: {
          notes: 'Backend returned notes but no title.',
        },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 43 } } as Awaited<ReturnType<typeof createQuickCaptureDraft>>)

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

    const nameInput = wrapper.find('input[type="text"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('Julia Domna Denarius')
    expect(wrapper.text()).toContain('Review Coin Details')

    const actionButtons = wrapper.findAll('.result-actions button')
    await actionButtons[2]!.trigger('click')
    await flushPromises()

    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      workingTitle: 'Julia Domna Denarius',
      source: 'find_coin_ai',
      obverseImage: file,
    }))
  })

  it('promotes a clear name from raw analysis lines when the draft title is missing', async () => {
    const file = new File(['obverse'], 'raw-analysis.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'medium',
          rawAnalysis: [
            'Name: Julia Domna Denarius',
            'Ruler: Julia Domna',
            'Denomination: Denarius',
            'Category: Roman',
          ].join('\n'),
        },
        numistaCandidates: [],
        prefilledDraft: {
          name: 'Unidentified Coin',
          notes: 'Name: Julia Domna Denarius\nRuler: Julia Domna',
        },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 44 } } as Awaited<ReturnType<typeof createQuickCaptureDraft>>)

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

    const nameInput = wrapper.find('input[type="text"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('Julia Domna Denarius')

    const actionButtons = wrapper.findAll('.result-actions button')
    await actionButtons[2]!.trigger('click')
    await flushPromises()

    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      workingTitle: 'Julia Domna Denarius',
      source: 'find_coin_ai',
    }))
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

  it('shows editable review details when NGC cert is detected', async () => {
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

    // NGC path should keep the review form editable while preserving certification details.
    expect(wrapper.text()).toContain('Review Coin Details')
    expect(wrapper.text()).toContain('NGC Certification: 1234567001')
    expect(wrapper.text()).toContain('NGC Grade')
    expect(wrapper.text()).toContain('Verify on NGC')

    const textInputs = wrapper.findAll('input[type="text"]')
    expect(textInputs).toHaveLength(6)
    expect((textInputs[0]!.element as HTMLInputElement).value).toBe('Augustus Denarius')
    expect((textInputs[1]!.element as HTMLInputElement).value).toBe('Augustus')
    expect((textInputs[2]!.element as HTMLInputElement).value).toBe('Denarius')
    expect((textInputs[3]!.element as HTMLInputElement).value).toBe('Roman')
    expect((textInputs[4]!.element as HTMLInputElement).value).toBe('Ch VF')
    expect((textInputs[5]!.element as HTMLInputElement).value).toBe('1234567001')

    await textInputs[0]!.setValue('Augustus AR Denarius')
    await textInputs[1]!.setValue('Octavian Augustus')
    await textInputs[2]!.setValue('AR Denarius')
    await textInputs[3]!.setValue('Roman Imperial')
    await textInputs[4]!.setValue('Choice VF')

    const actionButtons = wrapper.findAll('.result-actions button')
    await actionButtons[2]!.trigger('click')
    await flushPromises()

    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      workingTitle: 'Augustus AR Denarius',
      source: 'find_coin_ai',
      ngcCertNumber: '1234567001',
      ngcLookupUrl: 'https://www.ngccoin.com/certlookup/1234567001/NGCAncients/',
      ngcGrade: 'Ch VF',
      notes: expect.stringContaining('Ruler: Octavian Augustus'),
      obverseImage: file,
    }))
    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      notes: expect.stringContaining('Denomination: AR Denarius'),
    }))
    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      notes: expect.stringContaining('Category: Roman Imperial'),
    }))
    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      notes: expect.stringContaining('Grade: Choice VF'),
    }))
  })

  it.each([undefined, 'Unidentified Coin'] as Array<string | undefined>)(
    'derives Constantine title from slash-delimited NGC label when draft name is %s',
    async (draftName) => {
      const file = new File(['slab'], 'constantine-slab.jpg', { type: 'image/jpeg' })
      vi.mocked(lookupCoin).mockResolvedValue({
        data: {
          extractedData: {
            confidence: 'high',
            rawAnalysis: 'NGC cert detected',
            labelText: 'ROMAN EMPIRE / Constantine I, AD 307-337 / BI Reduced Nummus / LONDON MINT',
            ngc: {
              certNumber: '6828608-004',
              normalizedCert: '6828608004',
              lookupURL: 'https://www.ngccoin.com/certlookup/6828608004/NGCAncients/',
              grade: 'Ch VF',
            },
          },
          numistaCandidates: [],
          prefilledDraft: draftName === undefined
            ? { notes: 'Backend returned no title.' }
            : { name: draftName, notes: 'Backend returned placeholder title.' },
        },
      } as Awaited<ReturnType<typeof lookupCoin>>)
      vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 100 } } as Awaited<ReturnType<typeof createQuickCaptureDraft>>)

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

      const textInputs = wrapper.findAll('input[type="text"]')
      expect(textInputs).toHaveLength(6)
      expect((textInputs[0]!.element as HTMLInputElement).value).toBe('Constantine I Reduced Nummus')
      expect((textInputs[5]!.element as HTMLInputElement).value).toBe('6828608004')
      await textInputs[0]!.setValue('Constantine I BI Reduced Nummus')
      await textInputs[4]!.setValue('Choice VF')

      expect(wrapper.text()).not.toContain('ROMAN EMPIRE / Constantine I')

      const actionButtons = wrapper.findAll('.result-actions button')
      await actionButtons[2]!.trigger('click')
      await flushPromises()

      expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
        workingTitle: 'Constantine I BI Reduced Nummus',
        source: 'find_coin_ai',
        ngcCertNumber: '6828608004',
        notes: expect.stringContaining('Grade: Choice VF'),
        labelText: 'ROMAN EMPIRE / Constantine I, AD 307-337 / BI Reduced Nummus / LONDON MINT',
        obverseImage: file,
      }))
    }
  )

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
