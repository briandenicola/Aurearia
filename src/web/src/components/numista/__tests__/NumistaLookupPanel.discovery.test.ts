import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import NumistaLookupPanel from '../NumistaLookupPanel.vue'
import { makeNumistaCandidate, makeNumistaLookupOutcome } from '@/test/numista-fixtures'
import type { NumistaEvidence, NumistaLookupStatus } from '@/types'

const apiMocks = vi.hoisted(() => ({
  lookupNumista: vi.fn(),
  enrichNumista: vi.fn(),
}))

vi.mock('@/api/client', () => apiMocks)

const evidence: NumistaEvidence = {
  title: 'Honorius AE3 GLORIA ROMANORVM RIC IX 46 LRBC 2424',
  issuer: 'Honorius',
  denomination: 'AE3',
  mint: 'SMNT',
  dateText: '393–423 CE',
  material: 'Bronze',
  obverseInscription: 'DN HONORIVS PF AVG',
  reverseInscription: 'GLORIA ROMANORVM',
  visibleText: 'NGC Ancients Honorius AE3 RIC IX 46',
}

const wrappers: Array<ReturnType<typeof mount>> = []

function mountPanel(initialQuery = 'Honorius GLORIA ROMANORVM Nicomedia') {
  const wrapper = mount(NumistaLookupPanel, {
    attachTo: document.body,
    props: {
      initialQuery,
      evidence,
      path: 'direct',
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

describe('NumistaLookupPanel query-source and fallback disclosure', () => {
  beforeEach(() => {
    apiMocks.lookupNumista.mockReset()
    apiMocks.enrichNumista.mockReset()
  })

  afterEach(() => {
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  })

  it('submits an untouched server proposal as generated and displays a server-relaxed effective query separately', async () => {
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: {
        ...makeNumistaLookupOutcome({
          status: 'success',
          effectiveQuery: 'Honorius Nicomedia',
          candidates: [makeNumistaCandidate({ enrichmentState: 'enriched' })],
        }),
        querySource: 'generated',
        searchAttempt: 'relaxed',
        searchAttemptCount: 2,
      },
    })

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).toHaveBeenCalledTimes(1)
    expect(apiMocks.lookupNumista).toHaveBeenCalledWith({
      query: 'Honorius GLORIA ROMANORVM Nicomedia',
      path: 'direct',
      evidence,
      querySource: 'generated',
      generationVersion: 'numista-query-v2',
    })
    expect(wrapper.find<HTMLTextAreaElement>('#numista-query').element.value)
      .toBe('Honorius GLORIA ROMANORVM Nicomedia')
    expect(wrapper.text()).toContain('Honorius Nicomedia')
  })

  it('keeps user-edited sticky even when the collector restores the original proposal text', async () => {
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: {
        ...makeNumistaLookupOutcome({
          status: 'empty',
          effectiveQuery: 'Honorius GLORIA ROMANORVM Nicomedia',
          candidates: [],
        }),
        querySource: 'user-edited',
        searchAttempt: 'primary',
        searchAttemptCount: 1,
      },
    })

    const wrapper = mountPanel()
    const query = wrapper.find('#numista-query')
    await query.setValue('Honorius edited')
    await query.setValue('Honorius GLORIA ROMANORVM Nicomedia')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).toHaveBeenCalledTimes(1)
    expect(apiMocks.lookupNumista).toHaveBeenCalledWith(expect.objectContaining({
      query: 'Honorius GLORIA ROMANORVM Nicomedia',
      querySource: 'user-edited',
      generationVersion: 'numista-query-v2',
    }))
  })

  it('submits a query entered into an uninitialized panel as exact manual text', async () => {
    const manual = '  Honorius RIC IX 46 / LRBC 2424  '
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: {
        ...makeNumistaLookupOutcome({
          status: 'empty',
          effectiveQuery: manual,
          candidates: [],
        }),
        querySource: 'manual',
        searchAttempt: 'primary',
        searchAttemptCount: 1,
      },
    })

    const wrapper = mountPanel('')
    await wrapper.find('#numista-query').setValue(manual)
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).toHaveBeenCalledTimes(1)
    expect(apiMocks.lookupNumista).toHaveBeenCalledWith({
      query: manual,
      path: 'direct',
      evidence,
      querySource: 'manual',
    })
    expect(wrapper.find<HTMLTextAreaElement>('#numista-query').element.value).toBe(manual)
  })

  it('accepts a parent proposal refresh without marking the query as edited', async () => {
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: {
        ...makeNumistaLookupOutcome(),
        querySource: 'generated',
      },
    })
    const wrapper = mountPanel('Honorius old proposal')

    await wrapper.setProps({ initialQuery: 'Honorius refreshed proposal' })
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).toHaveBeenCalledWith(expect.objectContaining({
      query: 'Honorius refreshed proposal',
      querySource: 'generated',
      generationVersion: 'numista-query-v2',
    }))
  })

  it('keeps a server-downgraded stale proposal user-edited on retry', async () => {
    apiMocks.lookupNumista
      .mockResolvedValueOnce({
        data: {
          ...makeNumistaLookupOutcome({ status: 'empty', candidates: [] }),
          querySource: 'user-edited',
        },
      })
      .mockResolvedValueOnce({
        data: {
          ...makeNumistaLookupOutcome({ status: 'empty', candidates: [] }),
          querySource: 'user-edited',
        },
      })
    const wrapper = mountPanel('stale generated proposal')

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).toHaveBeenNthCalledWith(1, expect.objectContaining({
      querySource: 'generated',
    }))
    expect(apiMocks.lookupNumista).toHaveBeenNthCalledWith(2, expect.objectContaining({
      querySource: 'user-edited',
    }))
  })

  it.each<NumistaLookupStatus>([
    'unconfigured',
    'quota-limited',
    'timeout',
    'unavailable',
  ])('does not make a second frontend request after %s', async (status) => {
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({ status, candidates: [] }),
    })

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(apiMocks.lookupNumista).toHaveBeenCalledTimes(1)
  })

  it('does not make a second frontend request after a non-empty result or cancellation', async () => {
    apiMocks.lookupNumista.mockResolvedValueOnce({
      data: makeNumistaLookupOutcome({
        status: 'success',
        candidates: [makeNumistaCandidate({ enrichmentState: 'enriched' })],
      }),
    })
    let wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(apiMocks.lookupNumista).toHaveBeenCalledTimes(1)

    wrapper.unmount()
    apiMocks.lookupNumista.mockReset()
    apiMocks.lookupNumista.mockRejectedValueOnce(new DOMException('cancelled', 'AbortError'))
    wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(apiMocks.lookupNumista).toHaveBeenCalledTimes(1)
  })
})
