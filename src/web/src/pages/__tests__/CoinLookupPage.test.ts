import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import CoinLookupPage from '../CoinLookupPage.vue'
import { createDeepIdentificationJob, createQuickCaptureDraft, lookupCoin, lookupNumista } from '@/api/client'
import { makeNumistaCandidate, makeNumistaLookupOutcome } from '@/test/numista-fixtures'
import { normalizeGalleryImage } from '@/utils/galleryImage'

const routerPush = vi.fn()
const routerBack = vi.fn()

vi.mock('vue-router', () => ({
  RouterLink: {
    props: ['to'],
    template: '<a><slot /></a>',
  },
  useRouter: () => ({
    push: routerPush,
    back: routerBack,
  }),
}))

vi.mock('@/api/client', () => ({
  lookupCoin: vi.fn(),
  lookupNumista: vi.fn(),
  createQuickCaptureDraft: vi.fn(),
  createDeepIdentificationJob: vi.fn(),
  getDeepIdentificationCapability: vi.fn().mockResolvedValue({ data: { enabled: true } }),
  getApiErrorMessage: (error: unknown) => {
    if (typeof error !== 'object' || error === null) return ''
    const typed = error as { response?: { data?: { error?: unknown } }; message?: unknown }
    const apiMessage = typed.response?.data?.error
    if (typeof apiMessage === 'string') return apiMessage
    return typeof typed.message === 'string' ? typed.message : ''
  },
  onTokenRefreshed: vi.fn(),
}))

vi.mock('@/utils/galleryImage', () => ({
  normalizeGalleryImage: vi.fn(),
}))

function findAnalyzeButton(wrapper: ReturnType<typeof mount>) {
  return wrapper.findAll('button').find((button) =>
    button.text().includes('Analyze Photos') || button.text().includes('Create Quick AI Draft'),
  )
}

// The results-state action row (Retake Photo / Cancel / Save as Draft) no longer
// carries a `.result-actions` wrapper class after the Tailwind refactor — match by
// the buttons' visible labels instead, preserving DOM order (Retake, Cancel, Save).
const ACTION_BUTTON_LABELS = ['Retake Photo', 'Cancel', 'Save as Draft', 'Saving...']
function findActionButtons(wrapper: ReturnType<typeof mount>) {
  return wrapper
    .findAll('button')
    .filter((button) => ACTION_BUTTON_LABELS.some((label) => button.text().includes(label)))
}

describe('CoinLookupPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(lookupCoin).mockReset()
    vi.mocked(createQuickCaptureDraft).mockReset()
    vi.mocked(lookupNumista).mockReset()
    vi.mocked(createDeepIdentificationJob).mockReset()
    vi.mocked(normalizeGalleryImage).mockReset()
    vi.mocked(normalizeGalleryImage).mockImplementation(async file => file)
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
    Object.defineProperty(navigator, 'mediaDevices', {
      value: undefined,
      configurable: true,
    })
  })

  it('normalizes a gallery image before starting identification', async () => {
    const galleryFile = new File(['heic'], 'IMG_1234.HEIC', { type: 'image/heic' })
    const normalizedFile = new File(['jpeg'], 'IMG_1234.jpg', { type: 'image/jpeg' })
    vi.mocked(normalizeGalleryImage).mockResolvedValue(normalizedFile)
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: { confidence: 'low', rawAnalysis: '' },
        numistaCandidates: [],
        prefilledDraft: { name: 'Unidentified Coin' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)

    const wrapper = mount(CoinLookupPage)
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [galleryFile],
      configurable: true,
    })
    await input.trigger('change')
    await flushPromises()
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(normalizeGalleryImage).toHaveBeenCalledWith(galleryFile)
    expect(lookupCoin).toHaveBeenCalledWith([normalizedFile], '', ['obverse'])
  })

  it('sends and saves typed notes and a supporting image without treating it as the reverse', async () => {
    const obverse = new File(['obverse'], 'obverse.jpg', { type: 'image/jpeg' })
    const notesSource = new File(['label'], 'label.png', { type: 'image/png' })
    const notesImage = new File(['label-jpeg'], 'label.jpg', { type: 'image/jpeg' })
    vi.mocked(normalizeGalleryImage)
      .mockResolvedValueOnce(obverse)
      .mockResolvedValueOnce(notesImage)
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: { confidence: 'low', rawAnalysis: '' },
        numistaCandidates: [],
        prefilledDraft: { name: 'Unidentified Coin' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({
      data: { id: 84 },
    } as Awaited<ReturnType<typeof createQuickCaptureDraft>>)

    const wrapper = mount(CoinLookupPage)
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [obverse], configurable: true })
    await input.trigger('change')
    await flushPromises()

    await wrapper.find('[aria-label="Add reverse image"]').trigger('click')
    await wrapper.find('[aria-label="Add notes"]').trigger('click')
    await wrapper.find('textarea').setValue('Weight 3.2 g; dealer suggested Trajan.')
    Object.defineProperty(input.element, 'files', { value: [notesSource], configurable: true })
    await input.trigger('change')
    await flushPromises()
    await findAnalyzeButton(wrapper)?.trigger('click')
    await flushPromises()

    expect(lookupCoin).toHaveBeenCalledWith(
      [obverse, notesImage],
      'Weight 3.2 g; dealer suggested Trajan.',
      ['obverse', 'notes'],
    )

    const saveButton = findActionButtons(wrapper).find(button => button.text().includes('Save as Draft'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      notes: expect.stringContaining('**Collector notes:** Weight 3.2 g; dealer suggested Trajan.'),
      obverseImage: obverse,
      reverseImage: null,
      detailImages: [notesImage],
    }))
  })

  it('shows the API validation detail instead of a generic 400 message', async () => {
    const file = new File(['jpeg'], 'coin.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockRejectedValue({
      response: { data: { error: 'Invalid image upload' } },
      message: 'Request failed with status code 400',
    })

    const wrapper = mount(CoinLookupPage)
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [file],
      configurable: true,
    })
    await input.trigger('change')
    await flushPromises()
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Invalid image upload')
    expect(wrapper.text()).not.toContain('Request failed with status code 400')
  })

  it('waits for a user tap before requesting camera permission', async () => {
    const getUserMedia = vi.fn().mockResolvedValue({
      getTracks: () => [{ stop: vi.fn() }],
    })
    Object.defineProperty(navigator, 'mediaDevices', {
      value: { getUserMedia },
      configurable: true,
    })
    Object.defineProperty(HTMLMediaElement.prototype, 'play', {
      value: vi.fn().mockResolvedValue(undefined),
      configurable: true,
    })

    const wrapper = mount(CoinLookupPage, {
      global: {
        stubs: {
          Camera: true,
          Images: true,
          Search: true,
          X: true,
          AlertCircle: true,
          ShieldCheck: true,
          ExternalLink: true,
          RotateCcw: true,
          Bookmark: true,
          List: true,
        },
      },
    })

    await flushPromises()
    expect(getUserMedia).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Start Camera')
    expect(wrapper.find('.shutter-btn').attributes('disabled')).toBeDefined()
    expect(wrapper.find('.upload-icon-btn').exists()).toBe(true)

    await wrapper.find('.camera-start-btn').trigger('click')
    await flushPromises()

    expect(getUserMedia).toHaveBeenCalledTimes(1)
    expect(getUserMedia).toHaveBeenCalledWith({
      video: { facingMode: { ideal: 'environment' } },
      audio: false,
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
          List: true,
        },
      },
    })

    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [file],
      configurable: true,
    })

    await input.trigger('change')

    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    // No NGC cert, so should show editable review form
    expect(wrapper.text()).toContain('Review Coin Details')
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('AI Observations')
    expect(wrapper.text()).not.toContain('Obverse Description')
    expect(wrapper.text()).not.toContain('Reverse Description')
    expect(wrapper.findAll('textarea').filter(textarea => textarea.attributes('id') !== 'numista-query')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('Add to Collection')
    expect(wrapper.text()).toContain('Save as Draft')

    const nameInput = wrapper.find('input[type="text"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('Trajan Denarius')

    const actionButtons = findActionButtons(wrapper)
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

  it('links the Identify Coin header to all quick capture drafts', () => {
    const wrapper = mount(CoinLookupPage, {
      global: {
        stubs: {
          Camera: true,
          Images: true,
          Search: true,
          X: true,
          AlertCircle: true,
          ShieldCheck: true,
          ExternalLink: true,
          RotateCcw: true,
          Bookmark: true,
          List: true,
        },
      },
    })

    expect(wrapper.text()).toContain('Identify Coin')
    expect(wrapper.find('[aria-label="All drafts"]').exists()).toBe(true)
    expect(wrapper.find('.pwa-icon-btn').exists()).toBe(true)
  })

  it('labels the photo workflow Analyze Photos and retains Save as Draft', async () => {
    const file = new File(['coin'], 'labels.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: { confidence: 'low', rawAnalysis: '' },
        proposedNumistaQuery: '',
        numistaEvidence: {},
        numistaCandidates: [],
        prefilledDraft: { name: 'Unidentified Coin' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)

    const wrapper = mount(CoinLookupPage, {
      global: { stubs: { RouterLink: true, List: true } },
    })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')

    expect(findAnalyzeButton(wrapper)?.text()).toBe('Analyze Photos')
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(findActionButtons(wrapper).map(button => button.text())).toContain('Save as Draft')
  })

  it('shows Deep Analysis in the shared wizard without altering the fast lookup submit path', async () => {
    const file = new File(['coin'], 'labels.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: { confidence: 'low', rawAnalysis: '' },
        proposedNumistaQuery: '',
        numistaEvidence: {},
        numistaCandidates: [],
        prefilledDraft: { name: 'Unidentified Coin' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)

    const wrapper = mount(CoinLookupPage, {
      global: { stubs: { RouterLink: true, List: true } },
    })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')

    const deepAnalysisButton = wrapper.findAll('button').find((button) => button.text().includes('Deep Analysis'))
    expect(deepAnalysisButton).toBeTruthy()

    // Fast lookup submit continues to call lookupCoin only, never the deep-identification API.
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(lookupCoin).toHaveBeenCalledTimes(1)
    expect(createDeepIdentificationJob).not.toHaveBeenCalled()
  })

  it('reuses wizard evidence for Deep Analysis without rendering duplicate capture inputs', async () => {
    const obverse = new File(['obverse'], 'obverse.jpg', { type: 'image/jpeg' })
    const reverse = new File(['reverse'], 'reverse.jpg', { type: 'image/jpeg' })
    vi.mocked(createDeepIdentificationJob).mockResolvedValue({
      data: {
        job: {
          id: 42,
          source: 'intake',
          status: 'queued',
          partialSuccess: false,
          cancelRequested: false,
          lastSeq: 0,
          eventsAvailable: false,
          expiresAt: '2030-01-01T00:00:00Z',
          createdAt: '2030-01-01T00:00:00Z',
        },
      },
    } as Awaited<ReturnType<typeof createDeepIdentificationJob>>)

    const wrapper = mount(CoinLookupPage, {
      global: { stubs: { RouterLink: true, List: true } },
    })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [obverse], configurable: true })
    await input.trigger('change')
    await flushPromises()
    await wrapper.find('[aria-label="Add reverse image"]').trigger('click')
    Object.defineProperty(input.element, 'files', { value: [reverse], configurable: true })
    await input.trigger('change')
    await flushPromises()

    const deepAnalysisButton = wrapper.findAll('button').find((button) => button.text().includes('Deep Analysis'))
    await deepAnalysisButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Using photos from Identify Coin')
    expect(wrapper.find('[data-testid="reused-capture-summary"]').exists()).toBe(true)
    expect(wrapper.findAll('.fixed input[type="file"]')).toHaveLength(0)

    const startButton = wrapper.findAll('button').find((button) => button.text().includes('Start Deep Analysis'))
    await startButton!.trigger('click')
    await flushPromises()

    expect(createDeepIdentificationJob).toHaveBeenCalledWith(expect.objectContaining({
      obverseImage: obverse,
      reverseImage: reverse,
    }))
    expect(lookupCoin).not.toHaveBeenCalled()
    expect(routerPush).toHaveBeenCalledWith('/deep-analysis/42')
  })

  it('reveals an editable NGC Numista override by keyboard without an eager request', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 375, configurable: true })
    const file = new File(['slab'], 'ngc-override.jpg', { type: 'image/jpeg' })
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
          },
        },
        proposedNumistaQuery: 'Augustus denarius silver',
        numistaEvidence: { title: 'Augustus denarius', issuer: 'Augustus', material: 'Silver' },
        numistaCandidates: [],
        prefilledDraft: { name: 'Augustus Denarius', ruler: 'Augustus', material: 'Silver' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(lookupNumista).mockResolvedValue({
      data: makeNumistaLookupOutcome({ effectiveQuery: 'edited Augustus query' }),
    })

    const wrapper = mount(CoinLookupPage, {
      attachTo: document.body,
      global: {
        stubs: {
          RouterLink: true,
          Camera: true,
          Images: true,
          Search: true,
          X: true,
          AlertCircle: true,
          ShieldCheck: true,
          ExternalLink: true,
          RotateCcw: true,
          Bookmark: true,
          List: true,
        },
      },
    })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('NGC Certification: 1234567001')
    expect(wrapper.text()).toContain('Save as Draft')
    expect(lookupNumista).not.toHaveBeenCalled()

    const disclosure = wrapper.findAll('button').find(button => button.text() === 'Also search Numista')!
    expect(disclosure).toBeDefined()
    expect(disclosure.attributes('aria-expanded')).toBe('false')
    disclosure.element.focus()
    await disclosure.trigger('keydown', { key: 'Enter' })
    await flushPromises()

    expect(disclosure.attributes('aria-expanded')).toBe('true')
    expect(lookupNumista).not.toHaveBeenCalled()
    const query = wrapper.get('#numista-query')
    expect(document.activeElement).toBe(query.element)
    expect((query.element as HTMLTextAreaElement).value).toBe('Augustus denarius silver')
    await query.setValue('edited Augustus query')
    expect(lookupNumista).not.toHaveBeenCalled()

    const search = wrapper.findAll('button').find(button => button.text().includes('Search Numista'))!
    await search.trigger('click')
    await flushPromises()
    expect(lookupNumista).toHaveBeenCalledWith(expect.objectContaining({
      query: 'edited Augustus query',
      path: 'photo',
    }))

    const lookupContainer = query.element.closest('.min-w-0')
    expect(lookupContainer?.classList.contains('overflow-hidden')).toBe(true)
    expect(wrapper.find('.flex.flex-wrap').exists()).toBe(true)

    wrapper.unmount()
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

    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('AI Observations')
    expect(wrapper.find('.markdown-rendered strong').text()).toBe('Observed:')
    expect(wrapper.text()).toContain('AI saw a silver denarius.')
    expect(wrapper.html()).not.toContain('<script>')
    expect(wrapper.text()).toContain('Victory standing left')
    expect(wrapper.text()).not.toContain('Obverse Description')
    expect(wrapper.text()).not.toContain('Reverse Description')
    expect(wrapper.findAll('textarea').filter(textarea => textarea.attributes('id') !== 'numista-query')).toHaveLength(0)

    const inputs = wrapper.findAll('input[type="text"]')
    expect((inputs[0]!.element as HTMLInputElement).value).toBe('Trajan Denarius')
    expect((inputs[1]!.element as HTMLInputElement).value).toBe('Trajan')
    expect((inputs[2]!.element as HTMLInputElement).value).toBe('Denarius')
    expect((inputs[3]!.element as HTMLInputElement).value).toBe('Roman')
    expect((inputs[4]!.element as HTMLInputElement).value).toBe('VF')

    const actionButtons = findActionButtons(wrapper)
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

    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    const nameInput = wrapper.find('input[type="text"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('Julia Domna Denarius')
    expect(wrapper.text()).toContain('Review Coin Details')

    const actionButtons = findActionButtons(wrapper)
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

    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    const nameInput = wrapper.find('input[type="text"]')
    expect((nameInput.element as HTMLInputElement).value).toBe('Julia Domna Denarius')

    const actionButtons = findActionButtons(wrapper)
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

    await findAnalyzeButton(wrapper)!.trigger('click')
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

    await findAnalyzeButton(wrapper)!.trigger('click')
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

    const actionButtons = findActionButtons(wrapper)
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
    expect(lookupNumista).not.toHaveBeenCalled()
  })

  it('shows an editable photo proposal without an eager request and retains explicit selection on retry', async () => {
    const file = new File(['coin'], 'photo.jpg', { type: 'image/jpeg' })
    const selected = makeNumistaCandidate({ id: 12345, title: 'Selected denarius' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: { confidence: 'medium', rawAnalysis: 'photo evidence' },
        proposedNumistaQuery: 'Trajan denarius Rome silver',
        numistaEvidence: { title: 'Trajan denarius', issuer: 'Trajan', material: 'Silver' },
        numistaLookup: null,
        numistaCandidates: [],
        prefilledDraft: { name: 'Trajan denarius', ruler: 'Trajan', material: 'Silver' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({ data: makeNumistaLookupOutcome({ candidates: [selected] }) })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          effectiveQuery: 'edited retry',
          candidates: [makeNumistaCandidate({ id: 999, title: 'Different result' })],
        }),
      })
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 101 } } as never)

    const wrapper = mount(CoinLookupPage, {
      global: {
        stubs: {
          Camera: true, Images: true, Search: true, X: true, AlertCircle: true,
          ShieldCheck: true, ExternalLink: true, RotateCcw: true, Bookmark: true, List: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(lookupNumista).not.toHaveBeenCalled()
    const query = wrapper.find('#numista-query')
    expect((query.element as HTMLTextAreaElement).value).toBe('Trajan denarius Rome silver')
    await query.setValue('edited first')
    await wrapper.findAll('button').find(button => button.text().includes('Search Numista'))!.trigger('click')
    await flushPromises()
    expect(lookupNumista).toHaveBeenNthCalledWith(1, expect.objectContaining({
      query: 'edited first',
      path: 'photo',
    }))

    await wrapper.find('input[type="radio"]').trigger('keydown', { key: ' ' })
    await wrapper.find('input[type="radio"]').setValue(true)
    await query.setValue('edited retry')
    await wrapper.findAll('button').find(button => button.text().includes('Search again'))!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Selection retained from an earlier search')

    await findActionButtons(wrapper)[2]!.trigger('click')
    await flushPromises()
    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      selectedNumistaId: '12345',
      selectedNumistaUrl: 'https://en.numista.com/catalogue/pieces12345.html',
    }))
  })

  it('keeps empty or noisy photo evidence available for manual Numista query entry', async () => {
    const file = new File(['coin'], 'uncertain-photo.jpg', { type: 'image/jpeg' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: {
          confidence: 'low',
          rawAnalysis: 'Legend unclear; issuer cannot be determined.',
        },
        proposedNumistaQuery: '',
        numistaEvidence: {},
        numistaLookup: null,
        numistaCandidates: [],
        prefilledDraft: { name: 'Unidentified Coin' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    vi.mocked(lookupNumista).mockResolvedValue({
      data: makeNumistaLookupOutcome({
        effectiveQuery: 'manual bronze coin',
        candidates: [],
      }),
    })

    const wrapper = mount(CoinLookupPage, {
      global: {
        stubs: {
          Camera: true, Images: true, Search: true, X: true, AlertCircle: true,
          ShieldCheck: true, ExternalLink: true, RotateCcw: true, Bookmark: true, List: true,
          RouterLink: { template: '<a><slot /></a>' },
        },
      },
    })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    await input.trigger('change')
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(lookupNumista).not.toHaveBeenCalled()
    const query = wrapper.find('#numista-query')
    const searchButton = wrapper.findAll('button').find(button => button.text().includes('Search Numista'))!
    expect(query.exists()).toBe(true)
    expect((query.element as HTMLTextAreaElement).value).toBe('')
    expect(wrapper.text()).toContain('Enter at least one search term to enable Numista lookup.')
    expect(searchButton.attributes('disabled')).toBeDefined()

    await query.setValue('manual bronze coin')
    expect(searchButton.attributes('disabled')).toBeUndefined()
    await searchButton.trigger('click')
    await flushPromises()

    expect(lookupNumista).toHaveBeenCalledTimes(1)
    expect(lookupNumista).toHaveBeenCalledWith({
      query: 'manual bronze coin',
      path: 'photo',
      evidence: {},
      querySource: 'manual',
    })
  })

  it('keeps the shared lookup panel contained for narrow mobile layouts', async () => {
    Object.defineProperty(window, 'innerWidth', { value: 375, configurable: true })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: { confidence: 'low', rawAnalysis: '' },
        proposedNumistaQuery: 'editable query',
        numistaEvidence: {},
        numistaCandidates: [],
        prefilledDraft: { name: 'Unknown coin' },
      },
    } as Awaited<ReturnType<typeof lookupCoin>>)
    const wrapper = mount(CoinLookupPage, {
      global: { stubs: { RouterLink: true, List: true } },
    })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [new File(['coin'], 'mobile.jpg', { type: 'image/jpeg' })],
      configurable: true,
    })
    await input.trigger('change')
    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    expect(wrapper.find('.card.min-w-0.overflow-hidden').exists()).toBe(true)
    expect(wrapper.find('#numista-query').attributes('maxlength')).toBe('500')
    expect(wrapper.find('fieldset').exists()).toBe(false)
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

      await findAnalyzeButton(wrapper)!.trigger('click')
      await flushPromises()

      const textInputs = wrapper.findAll('input[type="text"]')
      expect(textInputs).toHaveLength(6)
      expect((textInputs[0]!.element as HTMLInputElement).value).toBe('Constantine I Reduced Nummus')
      expect((textInputs[5]!.element as HTMLInputElement).value).toBe('6828608004')
      await textInputs[0]!.setValue('Constantine I BI Reduced Nummus')
      await textInputs[4]!.setValue('Choice VF')

      expect(wrapper.text()).not.toContain('ROMAN EMPIRE / Constantine I')

      const actionButtons = findActionButtons(wrapper)
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

    await findAnalyzeButton(wrapper)!.trigger('click')
    await flushPromises()

    const links = wrapper.findAll('a').filter((link) => link.text().includes('View on Numista'))
    expect(links.map(link => link.attributes('href'))).toEqual([
      'http://example.com/pieces1.html',
      'https://example.com/pieces2.html',
    ])
    expect(wrapper.html()).not.toContain('javascript:alert')
    expect(wrapper.html()).not.toContain('data:text/html')
    expect(wrapper.html()).not.toContain('/catalogue/pieces1.html')
  })
})
