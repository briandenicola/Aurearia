import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

export type VoiceDictationStatus = 'idle' | 'requesting' | 'listening' | 'stopping'

export interface VoiceRecognitionAlternative {
  transcript: string
}

export interface VoiceRecognitionResult {
  isFinal: boolean
  length: number
  [index: number]: VoiceRecognitionAlternative
}

export interface VoiceRecognitionEvent {
  resultIndex: number
  results: ArrayLike<VoiceRecognitionResult>
}

export interface VoiceRecognitionErrorEvent {
  error: string
}

export interface VoiceRecognition {
  lang: string
  continuous: boolean
  interimResults: boolean
  maxAlternatives: number
  onstart: (() => void) | null
  onend: (() => void) | null
  onresult: ((event: VoiceRecognitionEvent) => void) | null
  onerror: ((event: VoiceRecognitionErrorEvent) => void) | null
  start(): void
  stop(): void
  abort(): void
}

export type VoiceRecognitionFactory = () => VoiceRecognition
export type VoiceMicrophonePermissionRequester = () => Promise<void>

type VoiceRecognitionConstructor = new () => VoiceRecognition
type RecognitionGlobal = typeof globalThis & {
  SpeechRecognition?: VoiceRecognitionConstructor
  webkitSpeechRecognition?: VoiceRecognitionConstructor
}

type UseVoiceDictationOptions = {
  disabled?: Ref<boolean>
  factory?: VoiceRecognitionFactory
  requestPermission?: VoiceMicrophonePermissionRequester | null
  onFinalText: (text: string) => void
}

function browserFactory(): VoiceRecognitionFactory | null {
  const target = globalThis as RecognitionGlobal
  const Recognition = target.SpeechRecognition ?? target.webkitSpeechRecognition
  return Recognition ? () => new Recognition() : null
}

function browserMicrophonePermissionRequester(): VoiceMicrophonePermissionRequester | null {
  const getUserMedia = globalThis.navigator?.mediaDevices?.getUserMedia
  if (!getUserMedia) return null

  return async () => {
    const stream = await getUserMedia.call(globalThis.navigator.mediaDevices, { audio: true })
    for (const track of stream.getTracks()) track.stop()
  }
}

function errorMessage(code: string): string {
  switch (code) {
    case 'not-allowed':
    case 'service-not-allowed':
      return 'Microphone access is blocked. Allow it in your browser site permissions, then try again.'
    case 'no-speech':
      return 'No speech was detected. Try again or continue typing.'
    case 'audio-capture':
      return 'No microphone is available. You can continue typing your notes.'
    case 'network':
      return 'Speech recognition is unavailable. Check your connection or continue typing.'
    default:
      return 'Voice dictation stopped unexpectedly. You can continue typing your notes.'
  }
}

function permissionErrorMessage(error: unknown): string {
  const name = typeof error === 'object' && error !== null && 'name' in error
    ? String(error.name)
    : ''

  switch (name) {
    case 'NotAllowedError':
    case 'SecurityError':
      return 'Microphone access is blocked. Allow it in your browser site permissions, then try again.'
    case 'NotFoundError':
    case 'DevicesNotFoundError':
      return 'No microphone is available. You can continue typing your notes.'
    case 'NotReadableError':
    case 'TrackStartError':
      return 'The microphone is being used by another app or could not be started.'
    default:
      return 'Microphone access could not be requested. You can continue typing your notes.'
  }
}

export function appendVoiceTranscript(existing: string, transcript: string, maxLength = 2000) {
  const spoken = transcript.trim()
  if (!spoken) return { value: existing.slice(0, maxLength), truncated: existing.length > maxLength }

  const separator = existing.length === 0 || /\s$/.test(existing) ? '' : ' '
  const combined = `${existing}${separator}${spoken}`
  return {
    value: combined.slice(0, maxLength),
    truncated: combined.length > maxLength,
  }
}

export function useVoiceDictation(options: UseVoiceDictationOptions) {
  const factory = options.factory ?? browserFactory()
  const requestPermission = options.requestPermission === undefined
    ? browserMicrophonePermissionRequester()
    : options.requestPermission
  const supported = ref(factory !== null)
  const status = ref<VoiceDictationStatus>('idle')
  const error = ref('')

  let recognition: VoiceRecognition | null = null
  let generation = 0
  let permissionConfirmed = false

  function abort() {
    const active = recognition
    generation += 1
    recognition = null
    status.value = 'idle'
    active?.abort()
  }

  async function start() {
    if (!factory || options.disabled?.value || status.value !== 'idle') return

    error.value = ''
    const session = ++generation
    const committedResults = new Set<number>()
    status.value = requestPermission && !permissionConfirmed ? 'requesting' : 'listening'

    try {
      if (requestPermission && !permissionConfirmed) {
        await requestPermission()
        if (session !== generation) return
        permissionConfirmed = true
      }

      const active = factory()
      recognition = active
      active.lang = 'en-US'
      active.continuous = false
      active.interimResults = true
      active.maxAlternatives = 1
      active.onstart = () => {
        if (session === generation) status.value = 'listening'
      }
      active.onend = () => {
        if (session !== generation) return
        recognition = null
        generation += 1
        status.value = 'idle'
      }
      active.onresult = (event) => {
        if (session !== generation) return
        const finalSegments: string[] = []
        for (let index = event.resultIndex; index < event.results.length; index += 1) {
          const result = event.results[index]
          if (!result?.isFinal || committedResults.has(index)) continue
          committedResults.add(index)
          const transcript = result[0]?.transcript.trim()
          if (transcript) finalSegments.push(transcript)
        }
        if (finalSegments.length > 0) options.onFinalText(finalSegments.join(' '))
      }
      active.onerror = (event) => {
        if (session !== generation) return
        recognition = null
        generation += 1
        status.value = 'idle'
        if (event.error !== 'aborted') error.value = errorMessage(event.error)
      }
      status.value = 'listening'
      active.start()
    } catch (caught) {
      if (session !== generation) return
      recognition = null
      generation += 1
      status.value = 'idle'
      error.value = permissionConfirmed
        ? 'Voice dictation could not start. You can continue typing your notes.'
        : permissionErrorMessage(caught)
    }
  }

  function stop() {
    if (!recognition || status.value !== 'listening') return
    status.value = 'stopping'
    recognition.stop()
  }

  if (options.disabled) {
    watch(options.disabled, (disabled) => {
      if (disabled && recognition) abort()
    })
  }
  onBeforeUnmount(abort)

  return {
    supported,
    status,
    error,
    start,
    stop,
    abort,
  }
}
