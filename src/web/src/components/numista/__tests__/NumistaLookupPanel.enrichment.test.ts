import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import NumistaLookupPanel from '../NumistaLookupPanel.vue'
import {
  makeNumistaCandidate,
  makeNumistaEvidence,
  makeNumistaLookupOutcome,
} from '@/test/numista-fixtures'
import type { NumistaCandidate, NumistaLookupOutcome } from '@/types'

const apiMocks = vi.hoisted(() => ({
  lookupNumista: vi.fn(),
  enrichNumista: vi.fn(),
}))

vi.mock('@/api/client', () => apiMocks)

const wrappers: Array<ReturnType<typeof mount>> = []

function broadCandidate(id: number, title: string, position: number): NumistaCandidate {
  return makeNumistaCandidate({
    id,
    canonicalUrl: `https://en.numista.com/catalogue/pieces${id}.html`,
    title,
    providerPosition: position,
    enrichmentState: 'not_requested',
    assessment: {
      scoringVersion: 'numista-v1',
      score: 50,
      band: 'weak',
      reasons: [
        {
          field: 'mint',
          kind: 'unavailable',
          code: 'candidate_value_missing',
          label: 'Mint detail is not available yet',
        },
      ],
    },
    denomination: undefined,
    mint: undefined,
    material: undefined,
    obverseThumbnail: undefined,
    reverseThumbnail: undefined,
  })
}

function mountPanel(initialQuery = 'Trajan denarius Rome silver') {
  const wrapper = mount(NumistaLookupPanel, {
    attachTo: document.body,
    props: {
      initialQuery,
      evidence: makeNumistaEvidence({
        title: 'Trajan denarius',
        mint: 'Rome',
        material: 'Silver',
      }),
    },
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

function deferredOutcome() {
  let resolve!: (value: { data: NumistaLookupOutcome }) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<{ data: NumistaLookupOutcome }>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe('NumistaLookupPanel progressive enrichment', () => {
  beforeEach(() => {
    apiMocks.lookupNumista.mockReset()
    apiMocks.enrichNumista.mockReset()
  })

  afterEach(() => {
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  })

  it('paints the complete broad set before enrichment resolves', async () => {
    const broad = [
      broadCandidate(1, 'Broad first', 0),
      broadCandidate(2, 'Broad second', 1),
      broadCandidate(3, 'Broad third', 2),
    ]
    const enrichment = deferredOutcome()
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: 'Trajan denarius Rome silver',
        candidates: broad,
        stage: 'broad',
      }),
    })
    apiMocks.enrichNumista.mockReturnValueOnce(enrichment.promise)

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Broad first')
    expect(wrapper.text()).toContain('Broad second')
    expect(wrapper.text()).toContain('Broad third')
    expect(wrapper.text().match(/Broad result/g)).toHaveLength(3)
    expect(wrapper.text()).toContain('Enriching')
    expect(apiMocks.enrichNumista).toHaveBeenCalledTimes(1)
    expect(apiMocks.enrichNumista.mock.calls[0]?.[0]).toMatchObject({
      query: 'Trajan denarius Rome silver',
      path: 'direct',
      candidates: broad,
    })

    enrichment.resolve({
      data: makeNumistaLookupOutcome({
        candidates: broad,
        stage: 'enriched',
      }),
    })
    await flushPromises()
  })

  it('trims only enrichment submission whitespace while retaining the editable direct query', async () => {
    const query = ' \tTrajan   denarius Rome silver\n'
    const broad = [broadCandidate(1, 'Broad result', 0)]
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: query,
        candidates: broad,
        stage: 'broad',
      }),
    })
    apiMocks.enrichNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: 'Trajan   denarius Rome silver',
        candidates: broad,
        stage: 'enriched',
      }),
    })

    const wrapper = mountPanel(query)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).toHaveBeenCalledWith(expect.objectContaining({ query }))
    expect(apiMocks.enrichNumista).toHaveBeenCalledWith(
      expect.objectContaining({ query: 'Trajan   denarius Rome silver' }),
      expect.any(AbortSignal),
    )
    expect(wrapper.find<HTMLTextAreaElement>('#numista-query').element.value).toBe(query)
  })

  it('preserves relaxed fallback attribution after enrichment completes', async () => {
    const primaryQuery = 'Honorius GLORIA ROMANORVM Nicomedia'
    const relaxedQuery = 'Honorius Nicomedia'
    const broad = [broadCandidate(1, 'Relaxed broad result', 0)]
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: relaxedQuery,
        candidates: broad,
        stage: 'broad',
        querySource: 'generated',
        searchAttempt: 'relaxed',
        searchAttemptCount: 2,
      }),
    })
    apiMocks.enrichNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: relaxedQuery,
        candidates: [makeNumistaCandidate({
          ...broad[0],
          enrichmentState: 'enriched',
        })],
        stage: 'enriched',
        querySource: 'generated',
        searchAttempt: 'relaxed',
        searchAttemptCount: 2,
      }),
    })

    const wrapper = mountPanel(primaryQuery)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.enrichNumista).toHaveBeenCalledWith(
      expect.objectContaining({
        query: relaxedQuery,
        querySource: 'generated',
        generationVersion: 'numista-query-v2',
      }),
      expect.any(AbortSignal),
    )
    expect(wrapper.text()).toContain(`Numista retried once with the relaxed query “${relaxedQuery}”.`)
    expect(wrapper.find<HTMLTextAreaElement>('#numista-query').element.value).toBe(primaryQuery)
  })

  it('preserves relaxed fallback attribution after partial enrichment failure', async () => {
    const primaryQuery = 'Honorius GLORIA ROMANORVM Nicomedia'
    const relaxedQuery = 'Honorius Nicomedia'
    const broad = [
      broadCandidate(1, 'Enriched result', 0),
      broadCandidate(2, 'Failed detail result', 1),
    ]
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: relaxedQuery,
        candidates: broad,
        stage: 'broad',
        querySource: 'generated',
        searchAttempt: 'relaxed',
        searchAttemptCount: 2,
      }),
    })
    apiMocks.enrichNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: relaxedQuery,
        candidates: [
          makeNumistaCandidate({ ...broad[0], enrichmentState: 'enriched' }),
          makeNumistaCandidate({ ...broad[1], enrichmentState: 'failed' }),
        ],
        stage: 'enriched',
        querySource: 'generated',
        searchAttempt: 'relaxed',
        searchAttemptCount: 2,
      }),
    })

    const wrapper = mountPanel(primaryQuery)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain(`Numista retried once with the relaxed query “${relaxedQuery}”.`)
    expect(wrapper.text()).toContain('Detail unavailable')
    expect(wrapper.find<HTMLTextAreaElement>('#numista-query').element.value).toBe(primaryQuery)
  })

  it('uses the backend enrichment attribution instead of masking it with broad metadata', async () => {
    const primaryQuery = 'Honorius GLORIA ROMANORVM Nicomedia'
    const relaxedQuery = 'Honorius Nicomedia'
    const broad = [broadCandidate(1, 'Broad result', 0)]
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: primaryQuery,
        candidates: broad,
        stage: 'broad',
        querySource: 'generated',
        searchAttempt: 'primary',
        searchAttemptCount: 1,
      }),
    })
    apiMocks.enrichNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        effectiveQuery: relaxedQuery,
        candidates: [makeNumistaCandidate({ ...broad[0], enrichmentState: 'enriched' })],
        stage: 'enriched',
        querySource: 'generated',
        searchAttempt: 'relaxed',
        searchAttemptCount: 2,
      }),
    })

    const wrapper = mountPanel(primaryQuery)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain(`Numista retried once with the relaxed query “${relaxedQuery}”.`)
  })

  it('suppresses whitespace-only lookup and enrichment submissions', async () => {
    const wrapper = mountPanel(' \t\r\n ')
    const button = wrapper.find<HTMLButtonElement>('button.btn-primary')

    expect(button.element.disabled).toBe(true)
    await button.trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).not.toHaveBeenCalled()
    expect(apiMocks.enrichNumista).not.toHaveBeenCalled()
  })

  it('updates order, reasons, and cached/enriched/failed labels while retaining an explicit selection', async () => {
    const broad = [
      broadCandidate(1, 'Initially first', 0),
      broadCandidate(2, 'Collector selection', 1),
      broadCandidate(3, 'Detail will fail', 2),
    ]
    const enrichment = deferredOutcome()
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({ candidates: broad, stage: 'broad' }),
    })
    apiMocks.enrichNumista.mockReturnValueOnce(enrichment.promise)

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    const radios = wrapper.findAll<HTMLInputElement>('input[type="radio"]')
    await radios[1]?.setValue(true)
    expect(wrapper.text()).toContain('Selected: Collector selection')

    const enriched = [
      makeNumistaCandidate({
        id: 3,
        canonicalUrl: 'https://en.numista.com/catalogue/pieces3.html',
        title: 'Detail will fail',
        providerPosition: 2,
        enrichmentState: 'failed',
        assessment: {
          scoringVersion: 'numista-v1',
          score: 40,
          band: 'weak',
          reasons: [{ field: 'mint', kind: 'unavailable', code: 'detail_failed', label: 'Detail could not be loaded' }],
        },
      }),
      makeNumistaCandidate({
        id: 1,
        canonicalUrl: 'https://en.numista.com/catalogue/pieces1.html',
        title: 'Enriched winner',
        providerPosition: 0,
        enrichmentState: 'enriched',
        assessment: {
          scoringVersion: 'numista-v1',
          score: 95,
          band: 'strong',
          reasons: [
            { field: 'mint', kind: 'match', code: 'mint_match', label: 'Rome mint matches' },
            { field: 'material', kind: 'match', code: 'material_match', label: 'Silver material matches' },
          ],
        },
      }),
      makeNumistaCandidate({
        id: 2,
        canonicalUrl: 'https://en.numista.com/catalogue/pieces2.html',
        title: 'Collector selection enriched',
        providerPosition: 1,
        enrichmentState: 'cached',
        assessment: {
          scoringVersion: 'numista-v1',
          score: 88,
          band: 'strong',
          reasons: [{ field: 'denomination', kind: 'match', code: 'denomination_match', label: 'Denomination matches' }],
        },
      }),
    ]
    enrichment.resolve({
      data: makeNumistaLookupOutcome({ candidates: enriched, stage: 'enriched' }),
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Details enriched')
    expect(wrapper.text()).toContain('Cached details')
    expect(wrapper.text()).toContain('Detail unavailable')
    expect(wrapper.text()).toContain('Rome mint matches')
    expect(wrapper.text()).toContain('Silver material matches')
    expect(wrapper.findAll('label')[0]?.text()).toContain('Detail will fail')
    expect(wrapper.find<HTMLInputElement>('input[type="radio"]:checked').element.value).toBe('2')
    expect(wrapper.text()).toContain('Selected: Collector selection enriched')
  })

  it('retains every broad candidate and marks attempted details failed when enrichment request fails', async () => {
    const broad = [
      broadCandidate(1, 'Still selectable one', 0),
      broadCandidate(2, 'Still selectable two', 1),
    ]
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({ candidates: broad, stage: 'broad' }),
    })
    apiMocks.enrichNumista.mockRejectedValueOnce(new Error('private upstream failure'))

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.findAll('input[type="radio"]')).toHaveLength(2)
    expect(wrapper.text()).toContain('Still selectable one')
    expect(wrapper.text()).toContain('Still selectable two')
    expect(wrapper.text().match(/Detail unavailable/g)).toHaveLength(2)
    expect(wrapper.text()).not.toContain('private upstream failure')
  })

  it('uses native keyboard-operable radios and preserves focusable selection controls', async () => {
    const broad = [
      broadCandidate(1, 'Keyboard one', 0),
      broadCandidate(2, 'Keyboard two', 1),
    ]
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({ candidates: broad, stage: 'broad' }),
    })
    apiMocks.enrichNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({ candidates: broad, stage: 'enriched' }),
    })

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    const radios = wrapper.findAll<HTMLInputElement>('input[type="radio"][name="numista-candidate"]')
    expect(radios).toHaveLength(2)
    radios[0]?.element.focus()
    expect(document.activeElement).toBe(radios[0]?.element)
    await radios[1]?.setValue(true)
    expect(wrapper.find<HTMLInputElement>('input[type="radio"]:checked').element.value).toBe('2')
    expect(wrapper.emitted('selectionChanged')?.at(-1)?.[0]).toMatchObject({ id: 2 })
  })

  it('renders only HTTPS images with descriptive alt text and keeps cards stacked before the small breakpoint', async () => {
    const candidates = [
      makeNumistaCandidate({
        id: 1,
        canonicalUrl: 'https://en.numista.com/catalogue/pieces1.html',
        title: 'Safe image',
        obverseThumbnail: 'https://images.numista.test/coin.jpg',
        enrichmentState: 'enriched',
      }),
      makeNumistaCandidate({
        id: 2,
        canonicalUrl: 'https://en.numista.com/catalogue/pieces2.html',
        title: 'Unsafe image',
        obverseThumbnail: 'http://images.numista.test/coin.jpg',
        enrichmentState: 'failed',
      }),
    ]
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({ candidates, stage: 'broad' }),
    })
    apiMocks.enrichNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({ candidates, stage: 'enriched' }),
    })

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    const images = wrapper.findAll('img')
    expect(images).toHaveLength(1)
    expect(images[0]?.attributes('src')).toBe('https://images.numista.test/coin.jpg')
    expect(images[0]?.attributes('alt')).toBe('Obverse thumbnail for Safe image')
    expect(wrapper.findAll('[aria-label^="Enlarge obverse image"]')).toHaveLength(1)

    for (const card of wrapper.findAll('fieldset > label')) {
      expect(card.classes()).toContain('grid')
      expect(card.classes()).toContain('sm:grid-cols-[auto_11rem_1fr]')
      expect(card.classes().some(className => className.startsWith('min-w-['))).toBe(false)
    }
  })
})
