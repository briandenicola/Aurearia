import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AdminAISection from '@/components/admin/AdminAISection.vue'
import type { AppSettings } from '@/types'

describe('AdminAISection', () => {
  it('loads and saves the default-off external tool server setting', async () => {
    const props = baseProps()
    const wrapper = mount(AdminAISection, { props })
    const toggle = wrapper.get<HTMLInputElement>('input[name="ExternalToolServerEnabled"]')

    expect(toggle.element.checked).toBe(false)

    await toggle.setValue(true)
    expect(props.settings.ExternalToolServerEnabled).toBe('true')

    await wrapper.get('form').trigger('submit')
    expect(wrapper.emitted('save')).toHaveLength(1)
  })

  it('clarifies provider tests do not validate the internal agent service', () => {
    const wrapper = mount(AdminAISection, {
      props: baseProps(),
    })

    expect(wrapper.text()).toContain('Provider tests validate the selected AI provider only.')
    expect(wrapper.text()).toContain('Agent chat and image analysis also require the internal agent service to be configured and running.')
  })
})

function baseProps() {
  const settings: AppSettings = {
    AIProvider: 'anthropic',
    AnthropicAPIKey: '',
    AnthropicModel: 'claude-sonnet-4',
    OllamaURL: '',
    OllamaModel: '',
    ObversePrompt: '',
    ReversePrompt: '',
    TextExtractionPrompt: '',
    OllamaTimeout: '300',
    SearXNGURL: '',
    LogLevel: 'info',
    ExternalToolServerEnabled: 'false',
    CoinSearchPrompt: '',
    CoinShowsPrompt: '',
    ValuationPrompt: '',
  }

  return {
    settings,
    settingDefaults: { ...settings },
    settingsMsg: '',
    settingsError: false,
    settingsSaving: false,
    anthropicModels: [{ id: 'claude-sonnet-4', name: 'Claude Sonnet 4' }],
    anthropicTesting: false,
    anthropicTestResult: 'Anthropic key is valid',
    anthropicTestOk: true,
    ollamaTesting: false,
    ollamaTestResult: '',
    ollamaTestOk: false,
    searxngTesting: false,
    searxngTestResult: '',
    searxngTestOk: false,
    coinSearchPromptDefault: '',
    coinShowsPromptDefault: '',
    valuationPromptDefault: '',
  }
}
