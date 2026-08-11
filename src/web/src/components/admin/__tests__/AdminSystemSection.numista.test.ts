import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminSystemSection from '../AdminSystemSection.vue'

const mocks = vi.hoisted(() => ({
  getAdminNumistaHealth: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

describe('AdminSystemSection Numista configuration and health', () => {
  beforeEach(() => {
    mocks.getAdminNumistaHealth.mockReset()
    mocks.getAdminNumistaHealth.mockResolvedValue({
      data: {
        configured: true,
        configurationValid: true,
        lastOutcome: 'quota-limited',
        lastCheckedAt: '2026-08-11T18:00:00Z',
        statusCounts: {
          success: 12,
          empty: 3,
          unconfigured: 2,
          'quota-limited': 1,
          timeout: 4,
          unavailable: 5,
        },
        broadRequestCount: 20,
        detailRequestCount: 7,
        freshCacheHitCount: 4,
        coalescedRequestCount: 6,
        providerLoadCount: 8,
        providerFailureCount: 2,
        cancelledRequestCount: 1,
        freshCacheHitRate: 1 / 3,
        p50ElapsedMs: 125,
        p95ElapsedMs: 980,
        enrichmentAttempted: 5,
        enrichmentSucceeded: 4,
        enrichmentFailed: 1,
        lastQuotaLimitedAt: '2026-08-11T17:59:00Z',
        lastRetryAfterSeconds: 120,
        apiKey: 'must-not-render-key',
        query: 'must-not-render-query',
        obverseInscription: 'must-not-render-inscription',
        visibleLabel: 'must-not-render-label',
        rawError: 'must-not-render-error',
      },
    })
  })

  it('renders bounded Numista settings and emits exact validated setting values', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    const bounds = [
      ['NumistaSearchTTLHours', '1', '720', '24'],
      ['NumistaDetailTTLHours', '1', '2160', '168'],
      ['NumistaEnrichmentLimit', '1', '10', '5'],
      ['NumistaSearchResultLimit', '1', '50', '20'],
      ['NumistaSearchTimeoutSeconds', '1', '10', '4'],
      ['NumistaDetailTimeoutSeconds', '1', '10', '3'],
    ] as const

    for (const [name, min, max, value] of bounds) {
      const input = wrapper.find<HTMLInputElement>(`input[name="${name}"]`)
      expect(input.exists(), `${name} control`).toBe(true)
      expect(input.attributes('type')).toBe('number')
      expect(input.attributes('min')).toBe(min)
      expect(input.attributes('max')).toBe(max)
      expect(input.element.value).toBe(value)
    }

    await wrapper.find<HTMLInputElement>('input[name="NumistaSearchTTLHours"]').setValue('48')
    await wrapper.find<HTMLInputElement>('input[name="NumistaDetailTTLHours"]').setValue('336')
    await wrapper.find('form').trigger('submit')

    const payload = wrapper.emitted('save')?.at(-1)?.[0]
    expect(payload).toMatchObject({
      numistaSearchTTLHours: '48',
      numistaDetailTTLHours: '336',
      numistaEnrichmentLimit: '5',
      numistaSearchResultLimit: '20',
      numistaSearchTimeoutSeconds: '4',
      numistaDetailTimeoutSeconds: '3',
    })
  })

  it('shows complete redacted status, latency, cache, enrichment, and quota signals', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()

    expect(mocks.getAdminNumistaHealth).toHaveBeenCalledTimes(1)
    const text = wrapper.text()
    expect(text).toContain('Numista Health')
    expect(text).toContain('Configured')
    expect(text).toContain('Configuration valid')
    for (const [label, count] of [
      ['Success', '12'],
      ['Empty', '3'],
      ['Unconfigured', '2'],
      ['Quota limited', '1'],
      ['Timeout', '4'],
      ['Unavailable', '5'],
    ]) {
      expect(text).toContain(label)
      expect(text).toContain(count)
    }
    expect(text).toMatch(/p50[^0-9]*125\s*ms/i)
    expect(text).toMatch(/p95[^0-9]*980\s*ms/i)
    expect(text).toMatch(/cache[^]*4[^]*fresh cache hit/i)
    expect(text).toMatch(/6[^]*coalesced request/i)
    expect(text).toMatch(/33\.3\s*%/)
    expect(text).toMatch(/provider loads[^]*8[^]*load/i)
    expect(text).toMatch(/2[^]*failed[^]*1[^]*cancelled/i)
    expect(text).toMatch(/enrichment[^]*5[^]*4[^]*1/i)
    expect(text).toMatch(/retry[^]*120\s*(seconds|sec|s)/i)
  })

  it('renders loading, error, retry, and explicit empty states', async () => {
    mocks.getAdminNumistaHealth
      .mockReset()
      .mockRejectedValueOnce(new Error('unavailable'))
      .mockResolvedValueOnce({
        data: {
          configured: true,
          configurationValid: true,
          statusCounts: {},
          broadRequestCount: 0,
          detailRequestCount: 0,
          freshCacheHitCount: 0,
          coalescedRequestCount: 0,
          providerLoadCount: 0,
          providerFailureCount: 0,
          cancelledRequestCount: 0,
          freshCacheHitRate: 0,
          p50ElapsedMs: 0,
          p95ElapsedMs: 0,
          enrichmentAttempted: 0,
          enrichmentSucceeded: 0,
          enrichmentFailed: 0,
        },
      })
    const wrapper = mount(AdminSystemSection, { props: baseProps() })

    expect(wrapper.get('[role="status"]').text()).toContain('Loading Numista health')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('temporarily unavailable')

    await wrapper.get('button.btn-secondary').trigger('click')
    await flushPromises()
    expect(mocks.getAdminNumistaHealth).toHaveBeenCalledTimes(2)

    const empty = wrapper.get('[role="status"]')
    expect(empty.text()).toContain('No Numista lookup health events have been recorded yet')
    expect(empty.text()).toContain('Numista is configured')
    expect(wrapper.text()).not.toContain('Status Counts')
    expect(wrapper.text()).not.toMatch(/p50\s+0\s+ms/i)
  })

  it('renders partial sparse status counts with absent statuses normalized to zero', async () => {
    mocks.getAdminNumistaHealth.mockReset().mockResolvedValue({
      data: {
        configured: true,
        configurationValid: true,
        lastOutcome: 'success',
        statusCounts: { success: 2, timeout: 1 },
        broadRequestCount: 3,
        detailRequestCount: 0,
        freshCacheHitCount: 0,
        coalescedRequestCount: 0,
        providerLoadCount: 3,
        providerFailureCount: 0,
        cancelledRequestCount: 0,
        freshCacheHitRate: 0,
        p50ElapsedMs: 20,
        p95ElapsedMs: 40,
        enrichmentAttempted: 0,
        enrichmentSucceeded: 0,
        enrichmentFailed: 0,
      },
    })
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()

    const statusText = wrapper.findAll('h4')
      .find((heading) => heading.text() === 'Status Counts')
      ?.element.parentElement?.textContent
    expect(statusText).toContain('Success2')
    expect(statusText).toContain('Timeout1')
    expect(statusText).toContain('Empty0')
    expect(statusText).toContain('Unavailable0')
  })

  it.each([
    ['unconfigured status', { statusCounts: { unconfigured: 1 }, lastOutcome: 'unconfigured' }, /Unconfigured1/],
    ['fresh cache hit', { freshCacheHitCount: 1 }, /1 fresh cache hit/],
    ['coalesced request', { coalescedRequestCount: 1 }, /1 coalesced request/],
    ['caller cancellation', { cancelledRequestCount: 1 }, /1 cancelled/],
  ])('renders health metrics for %s-only activity', async (_, activity, expected) => {
    mocks.getAdminNumistaHealth.mockReset().mockResolvedValue({
      data: { ...emptyHealth(), ...activity },
    })
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()

    expect(wrapper.text()).toContain('Status Counts')
    expect(wrapper.text()).toMatch(expected)
    expect(wrapper.text()).not.toContain('No Recent Activity')
  })

  it('keeps a genuinely zero-event snapshot in the empty state', async () => {
    mocks.getAdminNumistaHealth.mockReset().mockResolvedValue({ data: emptyHealth() })
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()

    expect(wrapper.get('[role="status"]').text()).toContain('No Numista lookup health events have been recorded yet')
    expect(wrapper.text()).not.toContain('Status Counts')
  })

  it('does not claim remaining quota or render sensitive response fields', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()
    const text = wrapper.text()

    expect(text).not.toMatch(/estimated\s+(remaining\s+)?quota/i)
    expect(text).not.toMatch(/quota\s+remaining/i)
    for (const forbidden of [
      'must-not-render-key',
      'must-not-render-query',
      'must-not-render-inscription',
      'must-not-render-label',
      'must-not-render-error',
    ]) {
      expect(text).not.toContain(forbidden)
    }
  })
})

function baseProps() {
  return {
    numistaApiKey: 'password-field-only',
    numistaSearchTTLHours: '24',
    numistaDetailTTLHours: '168',
    numistaEnrichmentLimit: '5',
    numistaSearchResultLimit: '20',
    numistaSearchTimeoutSeconds: '4',
    numistaDetailTimeoutSeconds: '3',
    pushoverAppToken: '',
    publicAppUrl: '',
    uspsApiBaseUrl: '',
    uspsApiKey: '',
    uspsApiKeyHeader: '',
    upsApiBaseUrl: '',
    upsTokenUrl: '',
    upsClientId: '',
    upsClientSecret: '',
    upsScope: '',
    fedexApiBaseUrl: '',
    fedexTokenUrl: '',
    fedexClientId: '',
    fedexClientSecret: '',
    fedexScope: '',
    logLevel: 'info',
    logLevels: ['debug', 'info', 'warn', 'error'] as const,
    saving: false,
    msg: '',
    error: false,
    appVersion: 'test',
    buildDate: '',
  }
}

function emptyHealth() {
  return {
    configured: true,
    configurationValid: true,
    statusCounts: {},
    broadRequestCount: 0,
    detailRequestCount: 0,
    freshCacheHitCount: 0,
    coalescedRequestCount: 0,
    providerLoadCount: 0,
    providerFailureCount: 0,
    cancelledRequestCount: 0,
    freshCacheHitRate: 0,
    p50ElapsedMs: 0,
    p95ElapsedMs: 0,
    enrichmentAttempted: 0,
    enrichmentSucceeded: 0,
    enrichmentFailed: 0,
  }
}
