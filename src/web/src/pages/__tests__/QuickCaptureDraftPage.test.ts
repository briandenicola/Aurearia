import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import QuickCaptureDraftPage from '../QuickCaptureDraftPage.vue'
import {
  getQuickCaptureDraft,
  lookupNumista,
  promoteQuickCaptureDraft,
  updateQuickCaptureDraft,
} from '@/api/client'
import { makeNumistaCandidate, makeNumistaLookupOutcome, makeSelectedNumistaReference } from '@/test/numista-fixtures'
import type { QuickCaptureDraft } from '@/types'

const routerPush = vi.fn()

vi.mock('vue-router', () => ({
  RouterLink: { props: ['to'], template: '<a><slot /></a>' },
  useRoute: () => ({ params: { id: '12' } }),
  useRouter: () => ({ push: routerPush }),
}))

vi.mock('@/api/client', () => ({
  discardQuickCaptureDraft: vi.fn(),
  getApiErrorMessage: vi.fn((error: unknown) => error instanceof Error ? error.message : ''),
  getQuickCaptureDraft: vi.fn(),
  lookupNumista: vi.fn(),
  onTokenRefreshed: vi.fn(),
  promoteQuickCaptureDraft: vi.fn(),
  updateQuickCaptureDraft: vi.fn(),
}))

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const source = fs.readFileSync(path.resolve(__dirname, '../QuickCaptureDraftPage.vue'), 'utf8')

function draft(overrides: Partial<QuickCaptureDraft> = {}): QuickCaptureDraft {
  return {
    id: 12,
    userId: 1,
    workingTitle: 'Trajan denarius',
    dateRange: '98-117',
    era: 'Roman',
    acquisitionSource: 'Coin show',
    purchasePrice: 125,
    notes: 'Draft notes',
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
    ...overrides,
  }
}

function mountPage() {
  return mount(QuickCaptureDraftPage, {
    global: {
      stubs: {
        AuthenticatedImage: true,
        QuickCaptureImageSlots: true,
        RouterLink: { template: '<a><slot /></a>' },
        List: true,
      },
    },
  })
}

describe('QuickCaptureDraftPage', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.resetAllMocks()
    routerPush.mockReset()
    vi.mocked(getQuickCaptureDraft).mockResolvedValue({ data: draft() } as never)
    vi.mocked(updateQuickCaptureDraft).mockResolvedValue({ data: draft() } as never)
    vi.mocked(lookupNumista).mockResolvedValue({
      data: makeNumistaLookupOutcome({
        candidates: [makeNumistaCandidate({
          id: 67890,
          canonicalUrl: 'https://en.numista.com/catalogue/pieces67890.html',
          title: 'Replacement reference',
        })],
      }),
    })
    vi.mocked(promoteQuickCaptureDraft).mockResolvedValue({
      data: { draftId: 12, status: 'promoted', coinId: 77, alreadyPromoted: false, target: 'collection' },
    })
  })

  it('loads a draft into an editable resume form and persists changed fields/images', () => {
    expect(source).toContain('getQuickCaptureDraft')
    expect(source).toContain('populateForm(res.data)')
    expect(source).toContain('updateQuickCaptureDraft')
    expect(source).toContain('workingTitle: workingTitle.value')
    expect(source).toContain('name: workingTitle.value.trim()')
    expect(source).toContain('purchaseLocation: acquisitionSource.value.trim()')
    expect(source).toContain('purchasePrice: currentPurchasePrice()')
    expect(source).toContain("return typeof purchasePrice.value === 'number' ? purchasePrice.value : null")
    expect(source).toContain('notes: notes.value.trim()')
    expect(source).not.toContain('name: workingTitle.value.trim() || undefined')
    expect(source).not.toContain('purchaseLocation: acquisitionSource.value.trim() || undefined')
    expect(source).not.toContain('purchasePrice.value ?? undefined')
    expect(source).not.toContain('notes: notes.value.trim() || undefined')
    expect(source).toContain('removeImageIds.value.size > 0')
    expect(source).toContain('QuickCaptureImageSlots')
    expect(source).toContain('Draft saved.')
  })

  it('surfaces validation/load errors and uses an explicit discard confirmation flow', () => {
    expect(source).toContain('Unable to load quick capture draft.')
    expect(source).toContain('Failed to save draft. Please try again.')
    expect(source).toContain('discardQuickCaptureDraft')
    expect(source).toContain('Discard this draft?')
    expect(source).toContain('Yes, discard')
  })

  it('preserves promotion integration and links terminal states without broad page coupling', () => {
    expect(source).toContain('PromotionReadinessPanel')
    expect(source).toContain(':promotion-overrides="promotionOverrides"')
    expect(source).toContain('This draft was promoted to a coin.')
    expect(source).toContain('View Coin')
    expect(source).toContain('This draft has been discarded.')
    expect(source).toContain('router.push(`/coin/${coinId}`)')
  })

  it('uses a compact drafts header action and concise page title', () => {
    expect(source).toContain('<h1>Draft</h1>')
    expect(source).toContain('aria-label="All drafts"')
    expect(source).toContain('pwa-icon-btn')
    expect(source).not.toContain('<h1>Quick Capture Draft</h1>')
    expect(source).not.toContain('>All drafts</RouterLink>')
  })

  it('resumes with the saved selection outside results and preserves it on unrelated edits', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('Numista #12345')
    expect(wrapper.text()).toContain('Selection retained from an earlier search')
    await wrapper.find('input[placeholder="Unattributed denarius"]').setValue('Edited title')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(updateQuickCaptureDraft).toHaveBeenCalledWith(12, expect.objectContaining({
      selectedNumistaId: undefined,
      selectedNumistaUrl: undefined,
      clearSelectedNumista: undefined,
    }))
  })

  it('replaces and explicitly clears the saved reference with keyboard-operable controls', async () => {
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.findAll('button').find(button => button.text().includes('Search Numista'))!.trigger('click')
    await flushPromises()
    const radio = wrapper.find('input[type="radio"]')
    expect(radio.attributes('name')).toBe('numista-candidate')
    await radio.trigger('keydown', { key: ' ' })
    await radio.setValue(true)
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(updateQuickCaptureDraft).toHaveBeenLastCalledWith(12, expect.objectContaining({
      selectedNumistaId: '67890',
      selectedNumistaUrl: 'https://en.numista.com/catalogue/pieces67890.html',
    }))

    vi.mocked(updateQuickCaptureDraft).mockResolvedValueOnce({
      data: draft({ selectedNumistaReference: null }),
    } as never)
    await wrapper.find('[aria-label="Remove selected Numista reference"]').trigger('click')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(updateQuickCaptureDraft).toHaveBeenLastCalledWith(12, expect.objectContaining({
      clearSelectedNumista: true,
    }))
  })

  it('retains the pending selection after validation failure and through a failed promotion retry', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('Search Numista'))!.trigger('click')
    await flushPromises()
    await wrapper.find('input[type="radio"]').setValue(true)

    vi.mocked(updateQuickCaptureDraft).mockRejectedValueOnce(new Error('Validation failed'))
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(wrapper.text()).toContain('Validation failed')
    expect(wrapper.text()).toContain('Replacement reference')
    expect(wrapper.text()).toContain('Save the reference change before promotion')
    expect(wrapper.findAll('button').find(button => button.text().includes('Promote to Collection'))!
      .attributes('disabled')).toBeDefined()

    vi.mocked(updateQuickCaptureDraft).mockResolvedValueOnce({
      data: draft({ selectedNumistaReference: makeSelectedNumistaReference({
        number: '67890',
        uri: 'https://en.numista.com/catalogue/pieces67890.html',
      }) }),
    } as never)
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    vi.mocked(promoteQuickCaptureDraft)
      .mockRejectedValueOnce(new Error('Promotion failed'))
      .mockResolvedValueOnce({
        data: { draftId: 12, status: 'promoted', coinId: 77, alreadyPromoted: true, target: 'collection' },
      })
    const confirmation = wrapper.find('input[type="checkbox"]')
    await confirmation.setValue(true)
    const promoteButton = wrapper.findAll('button').find(button => button.text().includes('Promote to Collection'))!
    await promoteButton.trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Promotion failed')
    expect(wrapper.text()).toContain('Replacement reference')

    await promoteButton.trigger('click')
    await flushPromises()
    expect(promoteQuickCaptureDraft).toHaveBeenCalledTimes(2)
    expect(routerPush).toHaveBeenCalledWith('/coin/77')
  })

  it('states that readiness treats the selected reference as optional', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.text()).toContain('Numista #12345 selected')
    expect(wrapper.text()).toContain('A Numista reference is optional')
  })

  it('displays the retained safe reference after promotion', async () => {
    vi.mocked(getQuickCaptureDraft).mockResolvedValueOnce({
      data: draft({ status: 'promoted', promotedCoinId: 77 }),
    } as never)
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.text()).toContain('Promoted with Numista #12345')
    expect(wrapper.find('a[href="https://en.numista.com/catalogue/pieces12345.html"]').exists()).toBe(true)
  })
})
