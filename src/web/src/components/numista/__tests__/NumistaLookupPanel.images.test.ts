import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import NumistaLookupPanel from '../NumistaLookupPanel.vue'
import {
  makeNumistaCandidate,
  makeNumistaEvidence,
  makeNumistaLookupOutcome,
} from '@/test/numista-fixtures'

const apiMocks = vi.hoisted(() => ({
  lookupNumista: vi.fn(),
  enrichNumista: vi.fn(),
}))

vi.mock('@/api/client', () => apiMocks)

const wrappers: Array<ReturnType<typeof mount>> = []

function mountPanel() {
  const wrapper = mount(NumistaLookupPanel, {
    attachTo: document.body,
    props: {
      initialQuery: 'Antoninus Pius denarius',
      evidence: makeNumistaEvidence(),
    },
  })
  wrappers.push(wrapper)
  return wrapper
}

async function renderCandidates(overrides = {}) {
  apiMocks.lookupNumista.mockResolvedValueOnce({
    data: makeNumistaLookupOutcome({
      candidates: [makeNumistaCandidate({
        title: 'Denarius - Antoninus Pius',
        obverseThumbnail: 'https://images.numista.test/obverse.jpg',
        reverseThumbnail: 'https://images.numista.test/reverse.jpg',
        enrichmentState: 'enriched',
        ...overrides,
      })],
      stage: 'enriched',
    }),
  })
  const wrapper = mountPanel()
  await wrapper.find('button.btn-primary').trigger('click')
  await flushPromises()
  return wrapper
}

function imageButton(side: 'obverse' | 'reverse'): HTMLButtonElement {
  const button = document.body.querySelector<HTMLButtonElement>(
    `[aria-label^="Enlarge ${side} image"]`,
  )
  if (!button) throw new Error(`Missing ${side} image button`)
  return button
}

describe('NumistaLookupPanel candidate image zoom', () => {
  beforeEach(() => {
    apiMocks.lookupNumista.mockReset()
    apiMocks.enrichNumista.mockReset()
  })

  afterEach(() => {
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
    document.body.innerHTML = ''
  })

  it('opens the selected image with meaningful text and returns focus after close', async () => {
    await renderCandidates()
    const trigger = imageButton('obverse')
    trigger.focus()
    trigger.click()
    await flushPromises()

    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]')
    const image = dialog?.querySelector<HTMLImageElement>('img')
    const close = dialog?.querySelector<HTMLButtonElement>(
      '[aria-label="Close enlarged candidate image"]',
    )

    expect(dialog?.getAttribute('aria-label'))
      .toBe('Obverse image for Numista candidate Denarius - Antoninus Pius')
    expect(image?.src).toBe('https://images.numista.test/obverse.jpg')
    expect(image?.alt).toBe('Obverse image for Numista candidate Denarius - Antoninus Pius')
    expect(document.activeElement).toBe(close)

    close?.click()
    await flushPromises()
    expect(document.body.querySelector('[role="dialog"]')).toBeNull()
    expect(document.activeElement).toBe(trigger)
  })

  it('opens either side with Enter or Space and closes with Escape or backdrop click', async () => {
    const wrapper = await renderCandidates()
    const obverse = wrapper.find('[aria-label^="Enlarge obverse image"]')
    const reverse = wrapper.find('[aria-label^="Enlarge reverse image"]')

    await obverse.trigger('keydown', { key: 'Enter' })
    await flushPromises()
    expect(document.body.querySelector<HTMLImageElement>('[role="dialog"] img')?.src)
      .toBe('https://images.numista.test/obverse.jpg')

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()
    expect(document.body.querySelector('[role="dialog"]')).toBeNull()

    await reverse.trigger('keydown', { key: ' ' })
    await flushPromises()
    expect(document.body.querySelector<HTMLImageElement>('[role="dialog"] img')?.alt)
      .toBe('Reverse image for Numista candidate Denarius - Antoninus Pius')

    const overlay = document.body.querySelector<HTMLElement>('[data-testid="numista-image-overlay"]')
    overlay?.click()
    await flushPromises()
    expect(document.body.querySelector('[role="dialog"]')).toBeNull()
  })

  it('does not render controls for missing or non-HTTPS images', async () => {
    await renderCandidates({
      obverseThumbnail: 'http://images.numista.test/obverse.jpg',
      reverseThumbnail: undefined,
    })

    expect(document.body.querySelector('[aria-label^="Enlarge obverse image"]')).toBeNull()
    expect(document.body.querySelector('[aria-label^="Enlarge reverse image"]')).toBeNull()
    expect(document.body.querySelector('[role="dialog"]')).toBeNull()
  })

  it('keeps the overlay and enlarged image constrained for narrow viewports', async () => {
    await renderCandidates()
    imageButton('reverse').click()
    await flushPromises()

    const overlay = document.body.querySelector<HTMLElement>('[data-testid="numista-image-overlay"]')
    const dialog = document.body.querySelector<HTMLElement>('[role="dialog"]')
    const image = dialog?.querySelector<HTMLImageElement>('img')

    expect(overlay?.className).toContain('p-3')
    expect(dialog?.className).toContain('max-h-[90dvh]')
    expect(dialog?.className).toContain('w-full')
    expect(image?.className).toContain('max-w-full')
    expect(image?.className).toContain('object-contain')
  })
})
