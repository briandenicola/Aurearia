import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CoinNumistaPanel from '../CoinNumistaPanel.vue'
import { createCoinReference, lookupNumista } from '@/api/client'
import { makeNumistaCandidate, makeNumistaLookupOutcome } from '@/test/numista-fixtures'

vi.mock('@/api/client', () => ({
  createCoinReference: vi.fn(),
  getApiErrorMessage: vi.fn(() => ''),
  lookupNumista: vi.fn(),
  onTokenRefreshed: vi.fn(),
}))

const baseProps = {
  coinId: 42,
  coinName: 'Antoninus Pius denarius',
  coinRuler: 'Antoninus Pius',
  coinDenomination: 'Denarius',
  coinMint: 'Rome',
  coinDateRange: '138–161 CE',
  coinMaterial: 'Silver' as const,
  coinObverseInscription: 'ANTONINVS AVG PIVS',
  coinReverseInscription: 'TR POT COS III',
}

function mountPanel() {
  return mount(CoinNumistaPanel, {
    props: baseProps,
    global: {
      plugins: [createPinia()],
      stubs: { RouterLink: { template: '<a><slot /></a>' } },
    },
  })
}

describe('CoinNumistaPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(lookupNumista).mockReset()
    vi.mocked(createCoinReference).mockReset()
    vi.mocked(createCoinReference).mockResolvedValue({ data: {} } as never)
  })

  it('builds an editable rich query and submits the exact edited value on first search and retry', async () => {
    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({ data: makeNumistaLookupOutcome({ effectiveQuery: 'edited first' }) })
      .mockResolvedValueOnce({ data: makeNumistaLookupOutcome({ effectiveQuery: 'edited retry' }) })

    const wrapper = mountPanel()
    expect(wrapper.find('textarea').element.value).toContain('Rome 138–161 CE Silver')
    expect(wrapper.find('textarea').element.value).toContain('ANTONINVS AVG PIVS')

    await wrapper.find('textarea').setValue('edited first')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(lookupNumista).toHaveBeenNthCalledWith(1, expect.objectContaining({ query: 'edited first', path: 'direct' }))

    await wrapper.find('textarea').setValue('edited retry')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(lookupNumista).toHaveBeenNthCalledWith(2, expect.objectContaining({ query: 'edited retry', path: 'direct' }))
  })

  it('renders explained ranking and persists only after explicit confirmation', async () => {
    const candidate = makeNumistaCandidate({
      assessment: {
        scoringVersion: 'numista-v1',
        score: 88,
        band: 'strong',
        reasons: [
          { field: 'mint', kind: 'match', code: 'mint_match', label: 'Mint matches' },
          { field: 'date', kind: 'conflict', code: 'date_conflict', label: 'Date range conflicts' },
        ],
      },
    })
    vi.mocked(lookupNumista).mockResolvedValue({
      data: makeNumistaLookupOutcome({ candidates: [candidate] }),
    })
    const wrapper = mountPanel()

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('88 · strong')
    expect(wrapper.text()).toContain('Mint matches')
    expect(wrapper.text()).toContain('Conflict: Date range conflicts')
    expect(createCoinReference).not.toHaveBeenCalled()

    await wrapper.find('input[type="radio"]').setValue(true)
    expect(createCoinReference).not.toHaveBeenCalled()
    await wrapper.findAll('button.btn-primary').at(-1)?.trigger('click')
    await flushPromises()

    expect(createCoinReference).toHaveBeenCalledTimes(1)
    expect(createCoinReference).toHaveBeenCalledWith(42, {
      catalog: 'Numista',
      number: String(candidate.id),
      uri: candidate.canonicalUrl,
    })
  })

  it('supports replace and remove without persisting either interaction', async () => {
    const first = makeNumistaCandidate({ id: 1, title: 'First' })
    const second = makeNumistaCandidate({ id: 2, title: 'Second' })
    vi.mocked(lookupNumista).mockResolvedValue({
      data: makeNumistaLookupOutcome({ candidates: [first, second] }),
    })
    const wrapper = mountPanel()

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    const radios = wrapper.findAll('input[type="radio"]')
    await radios[0]!.setValue(true)
    expect(wrapper.text()).toContain('Selected: First')
    await radios[1]!.setValue(true)
    expect(wrapper.text()).toContain('Selected: Second')
    await wrapper.find('button.btn-ghost').trigger('click')
    expect(wrapper.text()).not.toContain('Selected: Second')
    expect(createCoinReference).not.toHaveBeenCalled()
  })

  it('retains a selected candidate outside the latest retry results', async () => {
    const selected = makeNumistaCandidate({ id: 1, title: 'Retained candidate' })
    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({ data: makeNumistaLookupOutcome({ candidates: [selected] }) })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({ candidates: [makeNumistaCandidate({ id: 2, title: 'New result' })] }),
      })
    const wrapper = mountPanel()

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    await wrapper.find('input[type="radio"]').setValue(true)
    await wrapper.find('textarea').setValue('retry query')
    await wrapper.findAll('button.btn-primary')[0]!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Selection retained from an earlier search')
    expect(wrapper.text()).toContain('Retained candidate')
    expect(createCoinReference).not.toHaveBeenCalled()
  })

  it.each([
    ['empty', 'No matches found'],
    ['unconfigured', 'Numista lookup is not configured'],
    ['quota-limited', 'Numista lookup limit reached'],
    ['timeout', 'Numista lookup timed out'],
    ['unavailable', 'Numista lookup is unavailable'],
  ] as const)('renders the %s outcome distinctly', async (status, text) => {
    vi.mocked(lookupNumista).mockResolvedValue({
      data: makeNumistaLookupOutcome({ status, candidates: [], retryAfterSeconds: 60 }),
    })
    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain(text)
  })
})
