import { ref } from 'vue'
import {
  getAppSettings, getAppSettingDefaults, updateAppSettings, getOllamaStatus,
  getAnthropicModels, getCoinSearchPrompt, getCoinShowsPrompt, getValuationPrompt,
  testAnthropicConnection, testSearXNGConnection,
} from '@/api/client'
import type { AnthropicModel } from '@/api/client'
import type { AppSettings } from '@/types'
import { CATEGORIES, COIN_ERAS } from '@/types'
import { formatOptionList } from '@/utils/options'

export function useAdminConfig() {
  const settings = ref<AppSettings>({
    AIProvider: '',
    OllamaURL: 'http://localhost:11434',
    OllamaModel: 'llava',
    ObversePrompt: '',
    ReversePrompt: '',
    TextExtractionPrompt: '',
    OllamaTimeout: '300',
    SearXNGURL: '',
    LogLevel: 'info',
    PublicAppURL: '',
    CoinCategories: '',
    CoinEras: '',
  })
  const settingDefaults = ref<AppSettings>({
    AIProvider: '',
    OllamaURL: '',
    OllamaModel: '',
    ObversePrompt: '',
    ReversePrompt: '',
    TextExtractionPrompt: '',
    OllamaTimeout: '',
    SearXNGURL: '',
    LogLevel: '',
    PublicAppURL: '',
    CoinCategories: '',
    CoinEras: '',
  })
  const settingsMsg = ref('')
  const settingsError = ref(false)
  const settingsSaving = ref(false)

  let saveTimerId: ReturnType<typeof setTimeout> | null = null

  // Ollama
  const ollamaTesting = ref(false)
  const ollamaTestResult = ref('')
  const ollamaTestOk = ref(false)

  // Anthropic
  const anthropicTesting = ref(false)
  const anthropicTestResult = ref('')
  const anthropicTestOk = ref(false)
  const anthropicModels = ref<AnthropicModel[]>([
    { id: 'claude-sonnet-4-20250514', name: 'Claude Sonnet 4' },
    { id: 'claude-haiku-4-20250414', name: 'Claude Haiku 4' },
    { id: 'claude-opus-4-20250514', name: 'Claude Opus 4' },
  ])

  // SearXNG
  const searxngTesting = ref(false)
  const searxngTestResult = ref('')
  const searxngTestOk = ref(false)

  // Prompts
  const coinSearchPromptDefault = ref('')
  const coinShowsPromptDefault = ref('')
  const valuationPromptDefault = ref('')

  // Schedule-tab save messages (cleared alongside main settingsMsg)
  const availSettingsMsg = ref('')
  const availSettingsError = ref(false)
  const auctionSettingsMsg = ref('')
  const auctionSettingsError = ref(false)
  const healthSettingsMsg = ref('')
  const healthSettingsError = ref(false)
  const valSettingsMsg = ref('')
  const valSettingsError = ref(false)

  async function loadSettings() {
    try {
      const [settingsRes, defaultsRes] = await Promise.all([
        getAppSettings(),
        getAppSettingDefaults(),
      ])
      settingDefaults.value = { ...settingDefaults.value, ...defaultsRes.data }
      settings.value = { ...settings.value, ...settingsRes.data }

      // Apply defaults for coin property options if not set
      if (!settings.value.CoinCategories) {
        settings.value.CoinCategories = formatOptionList(CATEGORIES)
      }
      if (!settings.value.CoinEras) {
        settings.value.CoinEras = formatOptionList(COIN_ERAS)
      }

      // Apply defaults for auction settings if not set
      if (!settings.value.AuctionEndingCheckEnabled) {
        settings.value.AuctionEndingCheckEnabled = 'false'
      }
      if (!settings.value.AuctionEndingCheckStartTime) {
        settings.value.AuctionEndingCheckStartTime = '08:00'
      }
      if (!settings.value.AuctionEndingCheckInterval) {
        settings.value.AuctionEndingCheckInterval = '1440'
      }

      // Apply defaults for collection health snapshot settings if not set
      if (!settings.value.CollectionHealthSnapshotsEnabled) {
        settings.value.CollectionHealthSnapshotsEnabled = 'false'
      }
      if (!settings.value.CollectionHealthSnapshotsStartTime) {
        settings.value.CollectionHealthSnapshotsStartTime = '04:30'
      }

      const [modelsRes, coinSearchRes, coinShowsRes, valPromptRes] = await Promise.all([
        getAnthropicModels().catch(() => null),
        getCoinSearchPrompt().catch(() => null),
        getCoinShowsPrompt().catch(() => null),
        getValuationPrompt().catch(() => null),
      ])

      if (modelsRes?.data?.length) {
        anthropicModels.value = modelsRes.data
      }

      if (coinSearchRes?.data) {
        coinSearchPromptDefault.value = coinSearchRes.data.default
        if (!settings.value.CoinSearchPrompt) {
          settings.value.CoinSearchPrompt = coinSearchRes.data.prompt
        }
      }

      if (coinShowsRes?.data) {
        coinShowsPromptDefault.value = coinShowsRes.data.default
        if (!settings.value.CoinShowsPrompt) {
          settings.value.CoinShowsPrompt = coinShowsRes.data.prompt
        }
      }

      if (valPromptRes?.data) {
        valuationPromptDefault.value = valPromptRes.data.default
        if (!settings.value.ValuationPrompt) {
          settings.value.ValuationPrompt = valPromptRes.data.prompt
        }
      }
    } catch { /* use defaults */ }
  }

  async function saveSettings() {
    settingsSaving.value = true
    settingsMsg.value = ''
    settingsError.value = false
    availSettingsMsg.value = ''
    availSettingsError.value = false
    auctionSettingsMsg.value = ''
    auctionSettingsError.value = false
    healthSettingsMsg.value = ''
    healthSettingsError.value = false
    valSettingsMsg.value = ''
    valSettingsError.value = false
    try {
      const entries = Object.entries(settings.value).map(([key, value]) => ({ key, value: String(value) }))
      await updateAppSettings(entries)
      settingsMsg.value = 'Settings saved'
      availSettingsMsg.value = 'Settings saved'
      auctionSettingsMsg.value = 'Settings saved'
      healthSettingsMsg.value = 'Settings saved'
      valSettingsMsg.value = 'Settings saved'
      if (saveTimerId) clearTimeout(saveTimerId)
      saveTimerId = setTimeout(() => { availSettingsMsg.value = ''; auctionSettingsMsg.value = ''; healthSettingsMsg.value = ''; valSettingsMsg.value = '' }, 3000)
    } catch {
      settingsMsg.value = 'Failed to save settings'
      settingsError.value = true
      availSettingsMsg.value = 'Failed to save settings'
      availSettingsError.value = true
      auctionSettingsMsg.value = 'Failed to save settings'
      auctionSettingsError.value = true
      healthSettingsMsg.value = 'Failed to save settings'
      healthSettingsError.value = true
      valSettingsMsg.value = 'Failed to save settings'
      valSettingsError.value = true
    } finally {
      settingsSaving.value = false
    }
  }

  async function testOllamaConnection() {
    ollamaTesting.value = true
    ollamaTestResult.value = ''
    try {
      const res = await getOllamaStatus()
      ollamaTestOk.value = res.data.available
      ollamaTestResult.value = res.data.message
    } catch {
      ollamaTestOk.value = false
      ollamaTestResult.value = 'Failed to check Ollama status'
    } finally {
      ollamaTesting.value = false
    }
  }

  async function testAnthropicConn() {
    anthropicTesting.value = true
    anthropicTestResult.value = ''
    try {
      const res = await testAnthropicConnection()
      anthropicTestOk.value = res.data.available
      anthropicTestResult.value = res.data.message
    } catch {
      anthropicTestOk.value = false
      anthropicTestResult.value = 'Failed to test Anthropic connection'
    } finally {
      anthropicTesting.value = false
    }
  }

  async function testSearxngConn() {
    searxngTesting.value = true
    searxngTestResult.value = ''
    try {
      const res = await testSearXNGConnection()
      searxngTestOk.value = res.data.available
      searxngTestResult.value = res.data.message
    } catch {
      searxngTestOk.value = false
      searxngTestResult.value = 'Failed to test SearXNG connection'
    } finally {
      searxngTesting.value = false
    }
  }

  return {
    settings,
    settingDefaults,
    settingsMsg,
    settingsError,
    settingsSaving,
    // Ollama
    ollamaTesting,
    ollamaTestResult,
    ollamaTestOk,
    // Anthropic
    anthropicTesting,
    anthropicTestResult,
    anthropicTestOk,
    anthropicModels,
    // SearXNG
    searxngTesting,
    searxngTestResult,
    searxngTestOk,
    // Prompts
    coinSearchPromptDefault,
    coinShowsPromptDefault,
    valuationPromptDefault,
    // Schedule messages
    availSettingsMsg,
    availSettingsError,
    auctionSettingsMsg,
    auctionSettingsError,
    healthSettingsMsg,
    healthSettingsError,
    valSettingsMsg,
    valSettingsError,
    // Functions
    loadSettings,
    saveSettings,
    testOllamaConnection,
    testAnthropicConn,
    testSearxngConn,
    cleanup() {
      if (saveTimerId) clearTimeout(saveTimerId)
    },
  }
}
