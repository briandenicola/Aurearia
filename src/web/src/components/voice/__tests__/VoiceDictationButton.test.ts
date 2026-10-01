import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import VoiceDictationButton from '../VoiceDictationButton.vue'
import type {
  VoiceRecognition,
  VoiceRecognitionErrorEvent,
  VoiceRecognitionEvent,
} from '@/composables/useVoiceDictation'

class FakeRecognition implements VoiceRecognition {
  static latest: FakeRecognition | null = null
  lang = ''
  continuous = true
  interimResults = false
  maxAlternatives = 0
  onstart: (() => void) | null = null
  onend: (() => void) | null = null
  onresult: ((event: VoiceRecognitionEvent) => void) | null = null
  onerror: ((event: VoiceRecognitionErrorEvent) => void) | null = null
  start = vi.fn(() => this.onstart?.())
  stop = vi.fn()
  abort = vi.fn()

  constructor() {
    FakeRecognition.latest = this
  }
}

afterEach(() => {
  vi.unstubAllGlobals()
  FakeRecognition.latest = null
})

describe('VoiceDictationButton', () => {
  it('shows disclosure and toggles accessible listening state', async () => {
    vi.stubGlobal('SpeechRecognition', FakeRecognition)
    const wrapper = mount(VoiceDictationButton)

    expect(wrapper.text()).toContain('browser or device provider')
    const button = wrapper.get('button')
    expect(button.attributes('aria-label')).toBe('Start voice dictation')

    await button.trigger('click')
    expect(FakeRecognition.latest?.start).toHaveBeenCalledTimes(1)
    expect(button.attributes('aria-label')).toBe('Stop voice dictation')
    expect(wrapper.text()).toContain('Listening')

    await button.trigger('click')
    expect(FakeRecognition.latest?.stop).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Stopping')
  })

  it('emits final transcript text and renders actionable errors', async () => {
    vi.stubGlobal('webkitSpeechRecognition', FakeRecognition)
    const wrapper = mount(VoiceDictationButton)

    await wrapper.get('button').trigger('click')
    FakeRecognition.latest?.onresult?.({
      resultIndex: 0,
      results: [{ isFinal: true, 0: { transcript: 'Victory reverse' }, length: 1 }],
    })
    expect(wrapper.emitted('transcript')).toEqual([['Victory reverse']])

    FakeRecognition.latest?.onerror?.({ error: 'no-speech' })
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="alert"]').text()).toContain('No speech was detected')
  })

  it('does not render a broken action when recognition is unavailable', () => {
    vi.stubGlobal('SpeechRecognition', undefined)
    vi.stubGlobal('webkitSpeechRecognition', undefined)
    const wrapper = mount(VoiceDictationButton)

    expect(wrapper.find('button').exists()).toBe(false)
  })

  it('aborts active recognition when disabled', async () => {
    vi.stubGlobal('SpeechRecognition', FakeRecognition)
    const wrapper = mount(VoiceDictationButton, { props: { disabled: false } })

    await wrapper.get('button').trigger('click')
    await wrapper.setProps({ disabled: true })

    expect(FakeRecognition.latest?.abort).toHaveBeenCalledTimes(1)
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
  })
})
