import { expect, test } from '@playwright/test'
import { installAuthenticatedSession, installWorkflowApiMocks } from '../fixtures/workflow'

const tinyPng = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=',
  'base64',
)

test('mobile Identify Coin keeps voice notes editable and explicitly submitted', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  await installAuthenticatedSession(page)
  await installWorkflowApiMocks(page)
  await page.addInitScript(() => {
    type RecognitionResultHandler = (event: {
      resultIndex: number
      results: ArrayLike<{ isFinal: boolean; 0: { transcript: string }; length: number }>
    }) => void
    type RecognitionErrorHandler = (event: { error: string }) => void
    type VoiceTestWindow = Window & typeof globalThis & {
      __voiceEmitFinal: (text: string) => void
      __voiceEmitError: (code: string) => void
    }

    class MockSpeechRecognition {
      static latest: MockSpeechRecognition | null = null
      lang = ''
      continuous = false
      interimResults = false
      maxAlternatives = 1
      onstart: (() => void) | null = null
      onend: (() => void) | null = null
      onresult: RecognitionResultHandler | null = null
      onerror: RecognitionErrorHandler | null = null

      constructor() {
        MockSpeechRecognition.latest = this
      }

      start() {
        this.onstart?.()
      }

      stop() {
        this.onend?.()
      }

      abort() {
        const count = Number(document.documentElement.dataset.voiceAbortCount ?? '0')
        document.documentElement.dataset.voiceAbortCount = String(count + 1)
        this.onend?.()
      }
    }

    Object.defineProperty(window, 'SpeechRecognition', { value: MockSpeechRecognition })
    Object.defineProperty(navigator, 'standalone', { value: true })
    const testWindow = window as VoiceTestWindow
    testWindow.__voiceEmitFinal = (text) => MockSpeechRecognition.latest?.onresult?.({
      resultIndex: 0,
      results: [{ isFinal: true, 0: { transcript: text }, length: 1 }],
    })
    testWindow.__voiceEmitError = (code) => MockSpeechRecognition.latest?.onerror?.({ error: code })
  })

  const lookupRequests: string[] = []
  page.on('request', (request) => {
    if (new URL(request.url()).pathname.endsWith('/api/coins/lookup')) {
      lookupRequests.push(request.postDataBuffer()?.toString('utf8') ?? '')
    }
  })

  await page.goto('/lookup')
  const chooser = page.waitForEvent('filechooser')
  await page.getByRole('button', { name: 'Choose from library', exact: true }).click()
  await (await chooser).setFiles({ name: 'obverse.png', mimeType: 'image/png', buffer: tinyPng })
  await page.getByRole('button', { name: 'Add notes' }).click()

  await page.getByRole('button', { name: 'Start voice dictation' }).click()
  await page.evaluate(() => {
    (window as Window & typeof globalThis & { __voiceEmitFinal: (text: string) => void })
      .__voiceEmitFinal('Laureate bust right.')
  })
  await expect(page.getByRole('textbox', { name: 'Identification notes (optional)' }))
    .toHaveValue('Laureate bust right.')
  expect(lookupRequests).toHaveLength(0)

  await page.getByRole('button', { name: 'Stop voice dictation' }).click()
  await expect(page.getByRole('button', { name: 'Start voice dictation' })).toBeVisible()
  await page.getByRole('button', { name: 'Start voice dictation' }).click()
  await page.evaluate(() => {
    (window as Window & typeof globalThis & { __voiceEmitError: (code: string) => void })
      .__voiceEmitError('no-speech')
  })
  await expect(page.getByRole('alert')).toContainText('No speech was detected')

  await page.getByRole('button', { name: 'Start voice dictation' }).click()
  await page.getByRole('button', { name: 'Add reverse image' }).click()
  await expect(page.locator('html')).toHaveAttribute('data-voice-abort-count', '1')

  await page.getByRole('button', { name: 'Add notes' }).click()
  await page.getByRole('textbox', { name: 'Identification notes (optional)' })
    .fill('Laureate bust right; legend incomplete.')
  await page.getByRole('button', { name: 'Analyze Photos' }).click()

  await expect.poll(() => lookupRequests.length).toBe(1)
  expect(lookupRequests[0]).toContain('Laureate bust right; legend incomplete.')
})
