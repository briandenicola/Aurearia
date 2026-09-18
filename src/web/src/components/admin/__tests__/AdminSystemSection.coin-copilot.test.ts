import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminSystemSection from '../AdminSystemSection.vue'

const mocks = vi.hoisted(() => ({
  getAdminNumistaHealth: vi.fn(),
  getAdminOCREHealth: vi.fn(),
  getAdminDeepIdentificationObservability: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

describe('AdminSystemSection Coin Copilot settings', () => {
  beforeEach(() => {
    mocks.getAdminNumistaHealth.mockResolvedValue({ data: null })
    mocks.getAdminOCREHealth.mockResolvedValue({ data: null })
    mocks.getAdminDeepIdentificationObservability.mockResolvedValue({ data: null })
  })

  it('is default-off and renders every bounded Coin Copilot limit', () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    const section = wrapper.get('[data-testid="coin-copilot-section"]')
    expect(section.get<HTMLInputElement>('input[name="CoinCopilotEnabled"]').element.checked).toBe(false)

    const bounds = [
      ['CoinCopilotWorkerCount', '1', '4', '1'],
      ['CoinCopilotMaxActivePerUser', '1', '3', '1'],
      ['CoinCopilotQueueDepth', '1', '100', '16'],
      ['CoinCopilotMaxReasoningIterations', '1', '20', '8'],
      ['CoinCopilotMaxToolCalls', '1', '40', '12'],
      ['CoinCopilotHardTimeoutSeconds', '15', '600', '120'],
      ['CoinCopilotMaxPersistedToolResultBytes', '4096', '131072', '32768'],
      ['CoinCopilotEventRetentionHours', '1', '720', '168'],
      ['CoinCopilotCheckpointRetentionDays', '1', '365', '30'],
      ['CoinCopilotResumeWindowHours', '1', '720', '168'],
    ] as const
    for (const [name, min, max, value] of bounds) {
      const input = section.get<HTMLInputElement>(`input[name="${name}"]`)
      expect(input.attributes('min')).toBe(min)
      expect(input.attributes('max')).toBe(max)
      expect(input.element.value).toBe(value)
    }
    expect(section.find('input[name="CoinCopilotMaxEstimatedCostMicros"]').exists()).toBe(false)
  })

  it('clamps invalid limits and emits the complete settings payload', async () => {
    const wrapper = mount(AdminSystemSection, { props: baseProps() })
    await wrapper.get<HTMLInputElement>('input[name="CoinCopilotEnabled"]').setValue(true)
    await wrapper.get<HTMLInputElement>('input[name="CoinCopilotWorkerCount"]').setValue('99')
    await wrapper.get<HTMLInputElement>('input[name="CoinCopilotHardTimeoutSeconds"]').setValue('2')
    await wrapper.get<HTMLInputElement>('input[name="CoinCopilotMaxToolCalls"]').setValue('not-a-number')
    await wrapper.get('form').trigger('submit')

    expect(wrapper.emitted('save')?.at(-1)?.[0]).toMatchObject({
      coinCopilotEnabled: 'true',
      coinCopilotWorkerCount: '4',
      coinCopilotHardTimeoutSeconds: '15',
      coinCopilotMaxToolCalls: '12',
      coinCopilotMaxPersistedToolResultBytes: '32768',
      coinCopilotResumeWindowHours: '168',
    })
    expect(wrapper.emitted('save')?.at(-1)?.[0]).not.toHaveProperty('coinCopilotMaxEstimatedCostMicros')
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
