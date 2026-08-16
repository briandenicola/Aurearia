import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminSystemSection from '../AdminSystemSection.vue'

const mocks = vi.hoisted(() => ({
  getAdminNumistaHealth: vi.fn(),
  getAdminOCREHealth: vi.fn(),
  getAdminDeepIdentificationObservability: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

describe('AdminSystemSection OCRE / Deep Analysis configuration and health', () => {
  beforeEach(() => {
    mocks.getAdminNumistaHealth.mockReset()
    mocks.getAdminNumistaHealth.mockResolvedValue({ data: emptyNumistaHealth() })
    mocks.getAdminOCREHealth.mockReset()
    mocks.getAdminOCREHealth.mockResolvedValue({
      data: {
        enabled: true,
        callBudget: 5,
        gateValidated: true,
        lastOutcome: 'contributed',
        lastCheckedAt: '2026-08-11T18:00:00Z',
      },
    })
    mocks.getAdminDeepIdentificationObservability.mockReset()
    mocks.getAdminDeepIdentificationObservability.mockResolvedValue({ data: deepMetrics() })
  })

  it('groups OCRE controls in one labelled boundary distinct from Numista', () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    const group = wrapper.get('[data-testid="ocre-section"]')
    expect(group.attributes('aria-labelledby')).toBe('ocre-section-heading')
    expect(group.get('#ocre-section-heading').text()).toBe('OCRE / Deep Analysis')
    // Distinct from the Numista boundary.
    expect(wrapper.findAll('[data-testid="numista-section"]')).toHaveLength(1)
    expect(wrapper.findAll('[data-testid="ocre-section"]')).toHaveLength(1)
    expect(group.find('[data-testid="numista-section"]').exists()).toBe(false)
  })

  it('renders a default-off toggle and a bounded 1–20 call-budget input', () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    const deepToggle = wrapper.find<HTMLInputElement>('input[name="DeepIdentificationEnabled"]')
    expect(deepToggle.exists()).toBe(true)
    expect(deepToggle.element.checked).toBe(false)

    const toggle = wrapper.find<HTMLInputElement>('input[name="DeepIdentificationOCREEnabled"]')
    expect(toggle.exists()).toBe(true)
    expect(toggle.attributes('type')).toBe('checkbox')
    expect(toggle.element.checked).toBe(false)

    const budget = wrapper.find<HTMLInputElement>('input[name="DeepIdentificationOCRECallBudget"]')
    expect(budget.exists()).toBe(true)
    expect(budget.attributes('type')).toBe('number')
    expect(budget.attributes('min')).toBe('1')
    expect(budget.attributes('max')).toBe('20')
    expect(budget.element.value).toBe('3')
  })

  it('persists a toggle change and a validated call budget on save', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })

    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationEnabled"]').setValue(true)
    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationOCREEnabled"]').setValue(true)
    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationOCRECallBudget"]').setValue('9')
    await wrapper.find('form').trigger('submit')

    const payload = wrapper.emitted('save')?.at(-1)?.[0]
    expect(payload).toMatchObject({
      deepIdentificationEnabled: 'true',
      deepIdentificationOCREEnabled: 'true',
      deepIdentificationOCRECallBudget: '9',
    })
  })

  it('renders and persists every active Deep Analysis operational limit', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    const expectedNames = [
      'DeepIdentificationWorkerCount',
      'DeepIdentificationMaxActivePerUser',
      'DeepIdentificationQueueDepth',
      'DeepIdentificationHardTimeoutSeconds',
      'DeepIdentificationEventRetentionHours',
      'DeepIdentificationResultRetentionDays',
      'DeepIdentificationMaxProviders',
      'DeepIdentificationNumistaCallBudget',
    ]
    for (const name of expectedNames) {
      expect(wrapper.find(`input[name="${name}"]`).exists()).toBe(true)
    }

    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationWorkerCount"]').setValue('99')
    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationHardTimeoutSeconds"]').setValue('0')
    await wrapper.find('form').trigger('submit')

    expect(wrapper.emitted('save')?.at(-1)?.[0]).toMatchObject({
      deepIdentificationWorkerCount: '32',
      deepIdentificationHardTimeoutSeconds: '1',
      deepIdentificationQueueDepth: '32',
      deepIdentificationResultRetentionDays: '90',
    })
  })

  it('clamps an out-of-range call budget to the 1–20 bound on save', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })

    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationOCRECallBudget"]').setValue('99')
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('save')?.at(-1)?.[0]).toMatchObject({ deepIdentificationOCRECallBudget: '20' })

    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationOCRECallBudget"]').setValue('0')
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('save')?.at(-1)?.[0]).toMatchObject({ deepIdentificationOCRECallBudget: '1' })

    await wrapper.find<HTMLInputElement>('input[name="DeepIdentificationOCRECallBudget"]').setValue('1.5')
    await wrapper.find('form').trigger('submit')
    expect(wrapper.emitted('save')?.at(-1)?.[0]).toMatchObject({ deepIdentificationOCRECallBudget: '3' })
  })

  it('renders OCRE health enablement, budget, and last outcome only', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()

    expect(mocks.getAdminOCREHealth).toHaveBeenCalledTimes(1)
    const health = wrapper.get('[data-testid="ocre-health"]')
    const text = health.text()
    expect(text).toContain('Enabled')
    expect(text).toContain('Configuration valid')
    expect(text).toContain('5 per job')
    expect(text).toContain('Contributed')
  })

  it('renders loading, error retry, and explicit empty states without user content', async () => {
    mocks.getAdminOCREHealth
      .mockReset()
      .mockRejectedValueOnce(new Error('unavailable'))
      .mockResolvedValueOnce({
        data: { enabled: false, callBudget: 3, gateValidated: true, lastOutcome: null, lastCheckedAt: null },
      })
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()

    const ocreSection = wrapper.get('[data-testid="ocre-section"]')
    expect(ocreSection.get('[role="alert"]').text()).toContain('temporarily unavailable')

    const refreshButton = ocreSection.get('section[aria-labelledby="ocre-health-heading"] button')
    await refreshButton.trigger('click')
    await flushPromises()
    expect(mocks.getAdminOCREHealth).toHaveBeenCalledTimes(2)

    const healthText = wrapper.get('[data-testid="ocre-health"]').text()
    expect(healthText).toContain('Disabled')
    expect(healthText).toContain('No recent outcome')
  })

  it('never renders raw user notes, legends, or inscriptions from health', async () => {
    // The bounded contract only carries enablement/outcome; even if a buggy
    // upstream leaked extra fields, the component must not surface them.
    mocks.getAdminOCREHealth.mockReset().mockResolvedValue({
      data: {
        enabled: true,
        callBudget: 3,
        gateValidated: true,
        lastOutcome: 'no_match',
        lastCheckedAt: '2026-08-11T18:00:00Z',
        notes: 'must-not-render-note',
        legend: 'must-not-render-legend',
        inscription: 'must-not-render-inscription',
      },
    })
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()
    const text = wrapper.text()
    for (const forbidden of ['must-not-render-note', 'must-not-render-legend', 'must-not-render-inscription']) {
      expect(text).not.toContain(forbidden)
    }
    expect(wrapper.get('[data-testid="ocre-health"]').text()).toContain('No Match')
  })

  it('renders redacted Deep Analysis operational metrics', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await flushPromises()

    const metrics = wrapper.get('[data-testid="deep-observability"]').text()
    expect(metrics).toContain('4')
    expect(metrics).toContain('25.0%')
    expect(metrics).toContain('numista')
    expect(metrics).toContain('Contributed 2')
    for (const forbidden of ['notes', 'query', 'claims', 'report']) {
      expect(metrics.toLowerCase()).not.toContain(forbidden)
    }
  })
})

function baseProps() {
  return {
    numistaApiKey: '',
    numistaSearchTTLHours: '24',
    numistaDetailTTLHours: '168',
    numistaEnrichmentLimit: '5',
    numistaSearchResultLimit: '20',
    numistaSearchTimeoutSeconds: '4',
    numistaDetailTimeoutSeconds: '3',
    deepIdentificationEnabled: 'false',
    deepIdentificationOCREEnabled: 'false',
    deepIdentificationOCRECallBudget: '3',
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

function emptyNumistaHealth() {
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

function deepMetrics() {
  return {
    jobsByTerminalStatus: { completed: 2, partial: 1, failed: 1, cancelled: 0 },
    partialSuccessRate: 0.25,
    duration: { p50Ms: 1200, p95Ms: 4800 },
    providers: {
      numista: { statusCounts: { contributed: 2 }, latency: { p50Ms: 80, p95Ms: 150 } },
    },
    activeSseStreams: 1,
    reconnectCount: 3,
    truncationCount: 1,
    queueDepth: 2,
    hintDeletion: { success: 4, failure: 0 },
    janitor: { recoverySweeps: 2, retentionSweeps: 1, failures: 0 },
  }
}
