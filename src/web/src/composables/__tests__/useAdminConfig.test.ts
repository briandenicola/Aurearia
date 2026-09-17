import { describe, expect, it, vi, beforeEach } from 'vitest'
import { useAdminConfig } from '../useAdminConfig'

const mocks = vi.hoisted(() => ({
  getAppSettings: vi.fn(),
  getAppSettingDefaults: vi.fn(),
  updateAppSettings: vi.fn(),
  getOllamaStatus: vi.fn(),
  getAnthropicModels: vi.fn(),
  getCoinSearchPrompt: vi.fn(),
  getCoinShowsPrompt: vi.fn(),
  getValuationPrompt: vi.fn(),
  testAnthropicConnection: vi.fn(),
  testSearXNGConnection: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

describe('useAdminConfig', () => {
  beforeEach(() => {
    Object.values(mocks).forEach(mock => mock.mockReset())
    mocks.getAppSettings.mockResolvedValue({
      data: {
        AnthropicAPIKey: 'saved-key',
        PriceAlertCheckEnabled: 'true',
        PriceAlertCheckStartTime: '01:00',
        PriceAlertCheckInterval: '15',
      },
    })
    mocks.getAppSettingDefaults.mockResolvedValue({ data: {} })
    mocks.getAnthropicModels.mockResolvedValue({ data: [] })
    mocks.getCoinSearchPrompt.mockResolvedValue({ data: null })
    mocks.getCoinShowsPrompt.mockResolvedValue({ data: null })
    mocks.getValuationPrompt.mockResolvedValue({ data: null })
    mocks.updateAppSettings.mockResolvedValue({ data: {} })
  })

  it('saves auction alert scheduler keys expected by the backend scheduler', async () => {
    const config = useAdminConfig()

    await config.loadSettings()
    config.settings.value.AuctionAlertsCheckEnabled = 'true'
    config.settings.value.AuctionAlertsCheckStartTime = '09:30'
    config.settings.value.AuctionAlertsCheckInterval = '120'

    await config.saveSettings()

    const entries = mocks.updateAppSettings.mock.calls[0]?.[0] as Array<{ key: string; value: string }>
    expect(entries).toEqual(expect.arrayContaining([
      { key: 'AuctionAlertsCheckEnabled', value: 'true' },
      { key: 'AuctionAlertsCheckStartTime', value: '09:30' },
      { key: 'AuctionAlertsCheckInterval', value: '120' },
    ]))
    expect(entries.some(entry => entry.key.startsWith('PriceAlertCheck'))).toBe(false)

    config.cleanup()
  })

  it('requires an edited Anthropic key to be saved before testing', async () => {
    const config = useAdminConfig()

    await config.loadSettings()
    config.settings.value.AnthropicAPIKey = 'edited-key'

    await config.testAnthropicConn()

    expect(mocks.testAnthropicConnection).not.toHaveBeenCalled()
    expect(config.anthropicTestOk.value).toBe(false)
    expect(config.anthropicTestResult.value).toBe('Save AI settings before testing the Anthropic API.')

    config.cleanup()
  })

  it('tests the Anthropic key after it has been saved', async () => {
    mocks.testAnthropicConnection.mockResolvedValue({
      data: { available: true, message: 'Anthropic key is valid' },
    })
    const config = useAdminConfig()

    await config.loadSettings()
    config.settings.value.AnthropicAPIKey = 'edited-key'
    await config.saveSettings()
    await config.testAnthropicConn()

    expect(mocks.testAnthropicConnection).toHaveBeenCalledOnce()
    expect(config.anthropicTestOk.value).toBe(true)
    expect(config.anthropicTestResult.value).toBe('Anthropic key is valid')

    config.cleanup()
  })
})
