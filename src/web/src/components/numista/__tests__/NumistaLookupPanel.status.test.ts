import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import NumistaLookupPanel from '../NumistaLookupPanel.vue'
import { lookupNumista } from '@/api/client'
import {
  getNumistaCacheFreshnessText,
  getNumistaStatusGuidance,
} from '@/utils/numistaLookup'
import {
  makeNumistaCandidate,
  makeNumistaEvidence,
  makeNumistaLookupOutcome,
} from '@/test/numista-fixtures'
import type { NumistaLookupOutcome, NumistaLookupStatus } from '@/types'

vi.mock('@/api/client', () => ({
  lookupNumista: vi.fn(),
}))

const wrappers: Array<ReturnType<typeof mount>> = []

function mountPanel(props: Record<string, unknown> = {}) {
  const wrapper = mount(NumistaLookupPanel, {
    attachTo: document.body,
    props: {
      initialQuery: 'Trajan denarius',
      evidence: makeNumistaEvidence(),
      ...props,
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

describe('NumistaLookupPanel status behavior', () => {
  beforeEach(() => {
    vi.mocked(lookupNumista).mockReset()
  })

  afterEach(() => {
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  })

  it('provides typed labels and guidance for every explicit panel state', () => {
    expect(getNumistaStatusGuidance('idle', false).label).toBe('Ready')
    expect(getNumistaStatusGuidance('loading', false).label).toBe('Searching')
    expect(getNumistaStatusGuidance('success', false).label).toBe('Results ready')
    expect(getNumistaStatusGuidance('empty', false).canRetry).toBe(true)
    expect(getNumistaStatusGuidance('quota-limited', false).canRetry).toBe(true)
    expect(getNumistaStatusGuidance('timeout', false).canRetry).toBe(true)
    expect(getNumistaStatusGuidance('unavailable', false).canRetry).toBe(true)

    const userSetup = getNumistaStatusGuidance('unconfigured', false)
    expect(userSetup.canRetry).toBe(false)
    expect(userSetup.settingsHref).toBeUndefined()
    expect(userSetup.message).not.toContain('API key')

    const adminSetup = getNumistaStatusGuidance('unconfigured', true)
    expect(adminSetup.canRetry).toBe(true)
    expect(adminSetup.settingsHref).toBe('/admin?tab=system')
    expect(adminSetup.message).toContain('Numista API key')
  })

  it('renders idle, loading, success, and every terminal state with non-color text', async () => {
    let resolveLookup: ((value: { data: NumistaLookupOutcome }) => void) | undefined
    vi.mocked(lookupNumista).mockImplementationOnce(() => new Promise(resolve => {
      resolveLookup = resolve as (value: { data: NumistaLookupOutcome }) => void
    }) as ReturnType<typeof lookupNumista>)

    const wrapper = mountPanel({ isAdmin: true })
    expect(wrapper.find('[data-lookup-state="idle"]').text()).toContain('Ready to search')

    await wrapper.find('button.btn-primary').trigger('click')
    expect(wrapper.find('[data-lookup-state="loading"]').text()).toContain('Looking for ranked catalog candidates')

    resolveLookup?.({ data: makeNumistaLookupOutcome() })
    await flushPromises()
    expect(wrapper.find('[data-lookup-state="success"]').text()).toContain('Matches ready for review')

    for (const [status, title] of [
      ['empty', 'No matches found'],
      ['unconfigured', 'Numista lookup is not configured'],
      ['quota-limited', 'Numista lookup limit reached'],
      ['timeout', 'Numista lookup timed out'],
      ['unavailable', 'Numista lookup is unavailable'],
    ] as Array<[NumistaLookupStatus, string]>) {
      vi.mocked(lookupNumista).mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({ status, candidates: [], cache: undefined }),
      })
      await wrapper.find('button.btn-primary').trigger('click')
      await flushPromises()
      expect(wrapper.find(`[data-lookup-state="${status}"]`).text()).toContain(title)
    }
  })

  it('shows admin-only setup actions and safe ordinary-user guidance', async () => {
    vi.mocked(lookupNumista).mockResolvedValue({
      data: makeNumistaLookupOutcome({ status: 'unconfigured', candidates: [], cache: undefined }),
    })

    const userPanel = mountPanel()
    await userPanel.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(userPanel.text()).toContain('Ask an administrator for help.')
    expect(userPanel.find('a[href="/admin?tab=system"]').exists()).toBe(false)
    expect(userPanel.find('button.btn-primary').attributes('disabled')).toBeDefined()

    const adminPanel = mountPanel({ isAdmin: true })
    await adminPanel.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(adminPanel.find('a[href="/admin?tab=system"]').exists()).toBe(true)
    expect(adminPanel.find('button.btn-primary').text()).toBe('Retry after configuration')
    expect(adminPanel.find('button.btn-primary').attributes('disabled')).toBeUndefined()
  })

  it('only displays supplied retry timing and appropriate cache freshness', async () => {
    expect(getNumistaCacheFreshnessText('success', undefined)).toBeNull()
    expect(getNumistaCacheFreshnessText('unavailable', {
      hit: true,
      createdAt: '',
      expiresAt: '',
      ageSeconds: 120,
    })).toBeNull()
    expect(getNumistaCacheFreshnessText('success', {
      hit: true,
      createdAt: '',
      expiresAt: '',
      ageSeconds: 120,
    })).toBe('Cached results · 2 min old')

    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          status: 'quota-limited',
          candidates: [],
          cache: undefined,
          retryAfterSeconds: 120,
        }),
      })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          status: 'quota-limited',
          candidates: [],
          cache: undefined,
          retryAfterSeconds: undefined,
        }),
      })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          cache: { hit: true, createdAt: '', expiresAt: '', ageSeconds: 120 },
        }),
      })

    const wrapper = mountPanel()
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('wait 2 minutes before retrying')
    expect(wrapper.text()).not.toContain('remaining quota')

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Wait before trying again.')
    expect(wrapper.text()).not.toContain('2 minutes')

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Cached results · 2 min old')
  })

  it('retains edited input and selection across errors and focuses terminal status', async () => {
    const selected = makeNumistaCandidate({ id: 77, title: 'Retained selection' })
    vi.mocked(lookupNumista)
      .mockResolvedValueOnce({ data: makeNumistaLookupOutcome({ candidates: [selected] }) })
      .mockResolvedValueOnce({
        data: makeNumistaLookupOutcome({
          status: 'timeout',
          effectiveQuery: 'edited retry query',
          candidates: [],
          cache: undefined,
        }),
      })
      .mockRejectedValueOnce(new Error('transport detail must stay hidden'))

    const wrapper = mountPanel()
    const query = wrapper.find('#numista-query')
    await query.setValue('edited first query')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    await wrapper.find('input[type="radio"]').setValue(true)

    await query.setValue('edited retry query')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect((query.element as HTMLTextAreaElement).value).toBe('edited retry query')
    expect(wrapper.text()).toContain('Retained selection')
    expect(wrapper.text()).toContain('Selection retained from an earlier search')
    expect(document.activeElement).toBe(wrapper.find('[aria-live="polite"]').element)

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    expect((query.element as HTMLTextAreaElement).value).toBe('edited retry query')
    expect(wrapper.text()).toContain('Retained selection')
    expect(wrapper.text()).not.toContain('transport detail')
  })
})
