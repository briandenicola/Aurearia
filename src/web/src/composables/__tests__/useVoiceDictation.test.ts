import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  appendVoiceTranscript,
  useVoiceDictation,
  type VoiceMicrophonePermissionRequester,
  type VoiceRecognition,
  type VoiceRecognitionErrorEvent,
  type VoiceRecognitionEvent,
} from '../useVoiceDictation'

class FakeRecognition implements VoiceRecognition {
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
}

function result(transcript: string, isFinal: boolean) {
  return {
    isFinal,
    0: { transcript },
    length: 1,
  }
}

function mountComposable(
  disabled = ref(false),
  requestPermission: VoiceMicrophonePermissionRequester | null = null,
) {
  const recognition = new FakeRecognition()
  const onFinalText = vi.fn()
  let api!: ReturnType<typeof useVoiceDictation>
  const wrapper = mount(defineComponent({
    setup() {
      api = useVoiceDictation({
        disabled,
        factory: () => recognition,
        requestPermission,
        onFinalText,
      })
      return () => h('div')
    },
  }))
  return { wrapper, api, recognition, onFinalText, disabled }
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useVoiceDictation', () => {
  it('does not create or start recognition before the user action', () => {
    const { api, recognition } = mountComposable()

    expect(api.supported.value).toBe(true)
    expect(recognition.start).not.toHaveBeenCalled()
    expect(api.status.value).toBe('idle')
  })

  it('requests microphone permission before starting recognition', async () => {
    const requestPermission = vi.fn(async () => undefined)
    const { api, recognition } = mountComposable(ref(false), requestPermission)

    await api.start()

    expect(requestPermission).toHaveBeenCalledTimes(1)
    expect(recognition.start).toHaveBeenCalledTimes(1)
    expect(requestPermission.mock.invocationCallOrder[0]).toBeLessThan(recognition.start.mock.invocationCallOrder[0]!)
  })

  it('does not start recognition when microphone permission is denied', async () => {
    const requestPermission = vi.fn(async () => {
      throw new DOMException('denied', 'NotAllowedError')
    })
    const { api, recognition } = mountComposable(ref(false), requestPermission)

    await api.start()

    expect(recognition.start).not.toHaveBeenCalled()
    expect(api.status.value).toBe('idle')
    expect(api.error.value).toContain('browser site permissions')
  })

  it('prevents overlapping starts and allows Stop while browser startup is pending', () => {
    const { api, recognition } = mountComposable()
    recognition.start.mockImplementation(() => undefined)

    api.start()
    api.start()

    expect(recognition.start).toHaveBeenCalledTimes(1)
    expect(api.status.value).toBe('listening')
    api.stop()
    expect(recognition.stop).toHaveBeenCalledTimes(1)
  })

  it('configures English recognition and emits only new final results', () => {
    const { api, recognition, onFinalText } = mountComposable()

    api.start()
    expect(recognition.lang).toBe('en-US')
    expect(recognition.continuous).toBe(false)
    expect(recognition.interimResults).toBe(true)
    expect(recognition.maxAlternatives).toBe(1)
    expect(api.status.value).toBe('listening')

    const event = {
      resultIndex: 0,
      results: [
        result('Marcus', false),
        result('Marcus Aurelius', true),
      ],
    }
    recognition.onresult?.(event)
    recognition.onresult?.(event)

    expect(onFinalText).toHaveBeenCalledTimes(1)
    expect(onFinalText).toHaveBeenCalledWith('Marcus Aurelius')
  })

  it('stops explicitly and aborts when disabled or unmounted', async () => {
    const { api, recognition, disabled, wrapper } = mountComposable()

    api.start()
    api.stop()
    expect(api.status.value).toBe('stopping')
    expect(recognition.stop).toHaveBeenCalledTimes(1)

    disabled.value = true
    await wrapper.vm.$nextTick()
    expect(recognition.abort).toHaveBeenCalledTimes(1)
    expect(api.status.value).toBe('idle')

    disabled.value = false
    await wrapper.vm.$nextTick()
    api.start()
    wrapper.unmount()
    expect(recognition.abort).toHaveBeenCalledTimes(2)
  })

  it('maps recognition failures to typed-fallback messages', async () => {
    const { api, recognition, onFinalText } = mountComposable()

    await api.start()
    recognition.onerror?.({ error: 'not-allowed' })

    expect(api.error.value).toContain('browser site permissions')
    expect(api.status.value).toBe('idle')
    expect(onFinalText).not.toHaveBeenCalled()
  })

  it.each([
    ['no-speech', 'No speech was detected'],
    ['audio-capture', 'No microphone is available'],
    ['network', 'Check your connection'],
    ['vendor-secret-code', 'stopped unexpectedly'],
  ])('maps %s errors without exposing browser event details', (code, message) => {
    const { api, recognition } = mountComposable()

    api.start()
    recognition.onerror?.({ error: code })

    expect(api.error.value).toContain(message)
    expect(api.error.value).not.toContain(code)
  })

  it('ignores final callbacks delivered after an abort', () => {
    const { api, recognition, onFinalText } = mountComposable()

    api.start()
    const staleHandler = recognition.onresult
    api.abort()
    staleHandler?.({
      resultIndex: 0,
      results: [result('late transcript', true)],
    })

    expect(onFinalText).not.toHaveBeenCalled()
  })

  it('ignores retained callbacks after explicit Stop completes', () => {
    const { api, recognition, onFinalText } = mountComposable()

    api.start()
    const staleResult = recognition.onresult
    const staleError = recognition.onerror
    api.stop()
    recognition.onend?.()
    staleResult?.({
      resultIndex: 0,
      results: [result('late stopped transcript', true)],
    })
    staleError?.({ error: 'network' })

    expect(api.status.value).toBe('idle')
    expect(api.error.value).toBe('')
    expect(onFinalText).not.toHaveBeenCalled()
  })

  it('ignores retained callbacks after natural recognition completion', () => {
    const { api, recognition, onFinalText } = mountComposable()

    api.start()
    const staleResult = recognition.onresult
    const staleError = recognition.onerror
    recognition.onend?.()
    staleResult?.({
      resultIndex: 0,
      results: [result('late natural transcript', true)],
    })
    staleError?.({ error: 'no-speech' })

    expect(api.status.value).toBe('idle')
    expect(api.error.value).toBe('')
    expect(onFinalText).not.toHaveBeenCalled()
  })

  it('reports unsupported browsers without constructing recognition', () => {
    vi.stubGlobal('SpeechRecognition', undefined)
    vi.stubGlobal('webkitSpeechRecognition', undefined)
    let api!: ReturnType<typeof useVoiceDictation>
    const wrapper = mount(defineComponent({
      setup() {
        api = useVoiceDictation({ onFinalText: vi.fn() })
        return () => h('div')
      },
    }))

    expect(api.supported.value).toBe(false)
    api.start()
    expect(api.status.value).toBe('idle')
    wrapper.unmount()
  })
})

describe('appendVoiceTranscript', () => {
  it('preserves typed notes and appends readable dictated text', () => {
    expect(appendVoiceTranscript('Weight 3.2 g.', '  Mint mark below bust.  ')).toEqual({
      value: 'Weight 3.2 g. Mint mark below bust.',
      truncated: false,
    })
  })

  it('caps the combined notes without replacing their beginning', () => {
    expect(appendVoiceTranscript('abcd', 'efgh', 7)).toEqual({
      value: 'abcd ef',
      truncated: true,
    })
  })
})
