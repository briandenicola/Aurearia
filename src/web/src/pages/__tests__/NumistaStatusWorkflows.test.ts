import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CoinNumistaPanel from '@/components/coin/CoinNumistaPanel.vue'
import CoinLookupPage from '../CoinLookupPage.vue'
import QuickCaptureDraftPage from '../QuickCaptureDraftPage.vue'
import {
  createQuickCaptureDraft,
  getQuickCaptureDraft,
  lookupCoin,
  lookupNumista,
  updateQuickCaptureDraft,
} from '@/api/client'
import {
  makeNumistaCandidate,
  makeNumistaLookupOutcome,
  makeSelectedNumistaReference,
} from '@/test/numista-fixtures'
import type { QuickCaptureDraft } from '@/types'

const routerPush = vi.fn()

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' },
  useRoute: () => ({ params: { id: '12' } }),
  useRouter: () => ({ push: routerPush, back: vi.fn(), replace: vi.fn() }),
}))

vi.mock('@/api/client', () => ({
  createCoinReference: vi.fn(),
  createQuickCaptureDraft: vi.fn(),
  discardQuickCaptureDraft: vi.fn(),
  getApiErrorMessage: vi.fn((error: unknown) => error instanceof Error ? error.message : ''),
  getQuickCaptureDraft: vi.fn(),
  lookupCoin: vi.fn(),
  lookupNumista: vi.fn(),
  onTokenRefreshed: vi.fn(),
  promoteQuickCaptureDraft: vi.fn(),
  updateQuickCaptureDraft: vi.fn(),
}))

const directProps = {
  coinId: 42,
  coinName: 'Trajan denarius',
  coinRuler: 'Trajan',
  coinDenomination: 'Denarius',
  coinMint: 'Rome',
  coinDateRange: '98-117',
  coinMaterial: 'Silver' as const,
  coinObverseInscription: 'IMP TRAIANO',
  coinReverseInscription: 'P M TR P',
}

function draft(): QuickCaptureDraft {
  return {
    id: 12,
    userId: 1,
    workingTitle: 'Saved draft title',
    dateRange: '98-117',
    era: 'Roman',
    acquisitionSource: '',
    purchasePrice: null,
    notes: '',
    source: 'find_coin_ai',
    ngcCertNumber: '',
    ngcLookupUrl: '',
    ngcGrade: '',
    labelText: 'TRAIANO AVG',
    aiConfidence: 'medium',
    selectedNumistaReference: makeSelectedNumistaReference(),
    status: 'active',
    promotedCoinId: null,
    promotedAt: null,
    discardedAt: null,
    images: [],
    createdAt: '2026-08-11T12:00:00Z',
    updatedAt: '2026-08-11T12:00:00Z',
  }
}

describe('Numista status workflow retention', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.resetAllMocks()
    routerPush.mockReset()
  })

  it('keeps the direct edited query and selection through timeout and unavailable transitions', async () => {
    const selected = makeNumistaCandidate({ id: 5, title: 'Direct retained reference' })
    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({ data: makeNumistaLookupOutcome({ candidates: [selected] }) })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          status: 'timeout',
          effectiveQuery: 'direct edited retry',
          candidates: [],
          cache: undefined,
        }),
      })
      .mockRejectedValueOnce(new Error('provider failure'))

    const wrapper = mount(CoinNumistaPanel, {
      props: directProps,
      global: { plugins: [createPinia()], stubs: { RouterLink: true } },
    })
    const query = wrapper.find('#numista-query')
    await query.setValue('direct edited first')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    await wrapper.find('input[type="radio"]').setValue(true)
    await query.setValue('direct edited retry')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect((query.element as HTMLTextAreaElement).value).toBe('direct edited retry')
    expect(wrapper.text()).toContain('Direct retained reference')
    expect(wrapper.text()).toContain('Selection retained from an earlier search')
  })

  it('keeps the photo edited query and explicit selection through quota status before draft save', async () => {
    const file = new File(['coin'], 'coin.jpg', { type: 'image/jpeg' })
    const selected = makeNumistaCandidate({ id: 9, title: 'Photo retained reference' })
    vi.mocked(lookupCoin).mockResolvedValue({
      data: {
        extractedData: { confidence: 'medium', rawAnalysis: 'photo evidence' },
        proposedNumistaQuery: 'photo proposal',
        numistaEvidence: { title: 'Trajan' },
        numistaLookup: null,
        numistaCandidates: [],
        prefilledDraft: { name: 'Trajan denarius' },
      },
    } as never)
    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({ data: makeNumistaLookupOutcome({ candidates: [selected] }) })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          status: 'quota-limited',
          effectiveQuery: 'photo edited retry',
          candidates: [],
          cache: undefined,
        }),
      })
    vi.mocked(createQuickCaptureDraft).mockResolvedValue({ data: { id: 33 } } as never)

    const wrapper = mount(CoinLookupPage, {
      global: { plugins: [createPinia()], stubs: { RouterLink: true, List: true } },
    })
    const fileInput = wrapper.find('input[type="file"]')
    Object.defineProperty(fileInput.element, 'files', { value: [file], configurable: true })
    await fileInput.trigger('change')
    await wrapper.findAll('button').find(button => button.text().includes('Create Quick AI Draft'))!.trigger('click')
    await flushPromises()

    const query = wrapper.find('#numista-query')
    await query.setValue('photo edited first')
    await wrapper.findAll('button').find(button => button.text().includes('Search Numista'))!.trigger('click')
    await flushPromises()
    await wrapper.find('input[type="radio"]').setValue(true)
    await query.setValue('photo edited retry')
    await wrapper.findAll('button').find(button => button.text().includes('Search again'))!.trigger('click')
    await flushPromises()

    expect((query.element as HTMLTextAreaElement).value).toBe('photo edited retry')
    expect(wrapper.text()).toContain('Photo retained reference')
    await wrapper.findAll('button').find(button => button.text().includes('Save as Draft'))!.trigger('click')
    await flushPromises()
    expect(createQuickCaptureDraft).toHaveBeenCalledWith(expect.objectContaining({
      selectedNumistaId: '9',
      selectedNumistaUrl: selected.canonicalUrl,
    }))
  })

  it('keeps a persisted draft selection and manually edited query through form edits and errors', async () => {
    vi.mocked(getQuickCaptureDraft).mockResolvedValue({ data: draft() } as never)
    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          status: 'empty',
          effectiveQuery: 'manual draft query',
          candidates: [],
          cache: undefined,
        }),
      })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          status: 'unavailable',
          effectiveQuery: 'manual draft query',
          candidates: [],
          cache: undefined,
        }),
      })
    vi.mocked(updateQuickCaptureDraft).mockResolvedValue({ data: draft() } as never)

    const wrapper = mount(QuickCaptureDraftPage, {
      global: {
        plugins: [createPinia()],
        stubs: {
          AuthenticatedImage: true,
          QuickCaptureImageSlots: true,
          RouterLink: true,
          List: true,
        },
      },
    })
    await flushPromises()

    const query = wrapper.find('#numista-query')
    await query.setValue('manual draft query')
    await wrapper.find('input[placeholder="Unattributed denarius"]').setValue('Changed draft title')
    expect((query.element as HTMLTextAreaElement).value).toBe('manual draft query')

    await wrapper.findAll('button').find(button => button.text().includes('Search Numista'))!.trigger('click')
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('Retry lookup'))!.trigger('click')
    await flushPromises()
    expect((query.element as HTMLTextAreaElement).value).toBe('manual draft query')
    expect(wrapper.text()).toContain('Numista #12345')

    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(updateQuickCaptureDraft).toHaveBeenCalledWith(12, expect.objectContaining({
      selectedNumistaId: undefined,
      selectedNumistaUrl: undefined,
      clearSelectedNumista: undefined,
    }))
  })
})
