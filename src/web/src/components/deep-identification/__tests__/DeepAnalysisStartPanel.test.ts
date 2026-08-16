import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepAnalysisStartPanel from '../DeepAnalysisStartPanel.vue'

function makeFile(name: string) {
  return new File(['data'], name, { type: 'image/jpeg' })
}

describe('DeepAnalysisStartPanel', () => {
  function mountPanel() {
    Object.defineProperty(URL, 'createObjectURL', {
      value: () => 'blob:deep-analysis',
      configurable: true,
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      value: () => {},
      configurable: true,
    })
    return mount(DeepAnalysisStartPanel)
  }

  function setFile(wrapper: ReturnType<typeof mountPanel>, index: number, file: File) {
    const input = wrapper.findAll('input[type="file"]')[index]
    Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
    return input.trigger('change')
  }

  it('blocks submit with a specific message when obverse or reverse is missing', async () => {
    const wrapper = mountPanel()
    const startButton = wrapper.findAll('button').find((b) => b.text().includes('Start Deep Analysis'))
    await startButton!.trigger('click')

    expect(wrapper.text()).toContain('Obverse and reverse photos are both required to start Deep Analysis.')
    expect(wrapper.emitted('submit')).toBeFalsy()
  })

  it('enforces a maximum of three hint images', async () => {
    const wrapper = mountPanel()
    await setFile(wrapper, 0, makeFile('obverse.jpg'))
    await setFile(wrapper, 1, makeFile('reverse.jpg'))

    const hintInput = wrapper.findAll('input[type="file"]').find((i) => i.attributes('multiple') !== undefined)
    const files = [makeFile('h1.jpg'), makeFile('h2.jpg'), makeFile('h3.jpg'), makeFile('h4.jpg')]
    Object.defineProperty(hintInput!.element, 'files', { value: files, configurable: true })
    await hintInput!.trigger('change')

    expect(wrapper.text()).toContain('3 of 3 selected')

    const startButton = wrapper.findAll('button').find((b) => b.text().includes('Start Deep Analysis'))
    await startButton!.trigger('click')

    expect(wrapper.emitted('submit')).toBeTruthy()
    const emitted = wrapper.emitted('submit')![0][0] as { hintImages: File[] }
    expect(emitted.hintImages).toHaveLength(3)
  })

  it('emits submit with obverse, reverse, notes and provider override once valid', async () => {
    const wrapper = mountPanel()
    await setFile(wrapper, 0, makeFile('obverse.jpg'))
    await setFile(wrapper, 1, makeFile('reverse.jpg'))
    await wrapper.find('textarea').setValue('Mint mark visible under magnification.')

    const nomismaCheckbox = wrapper.findAll('input[type="checkbox"]').find((c) => c.element.value === 'nomisma')
    await nomismaCheckbox!.setValue(true)

    const startButton = wrapper.findAll('button').find((b) => b.text().includes('Start Deep Analysis'))
    await startButton!.trigger('click')

    expect(wrapper.emitted('submit')).toBeTruthy()
    const payload = wrapper.emitted('submit')![0][0] as {
      obverseImage: File | null
      reverseImage: File | null
      notes?: string
      providers?: string[]
    }
    expect(payload.obverseImage?.name).toBe('obverse.jpg')
    expect(payload.reverseImage?.name).toBe('reverse.jpg')
    expect(payload.notes).toBe('Mint mark visible under magnification.')
    expect(payload.providers).toEqual(['nomisma'])
  })

  it('emits cancel when the Cancel button is clicked', async () => {
    const wrapper = mountPanel()
    const cancelButton = wrapper.findAll('button').find((b) => b.text().includes('Cancel'))
    await cancelButton!.trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })

  it('reuses Identify Coin evidence without rendering another set of image or notes inputs', async () => {
    const obverse = makeFile('obverse.jpg')
    const reverse = makeFile('reverse.jpg')
    const hint = makeFile('label.jpg')
    const wrapper = mount(DeepAnalysisStartPanel, {
      props: {
        reuseCapturedEvidence: true,
        initialObverseImage: obverse,
        initialReverseImage: reverse,
        initialHintImages: [hint],
        initialNotes: 'Weight 3.2 g',
      },
    })

    expect(wrapper.get('[data-testid="reused-capture-summary"]').text())
      .toContain('obverse, reverse, 1 supporting image, notes')
    expect(wrapper.findAll('input[type="file"]')).toHaveLength(0)
    expect(wrapper.find('textarea').exists()).toBe(false)

    await wrapper.findAll('button').find((button) => button.text().includes('Start Deep Analysis'))!.trigger('click')
    const payload = wrapper.emitted('submit')?.[0]?.[0] as {
      obverseImage: File
      reverseImage: File
      hintImages: File[]
      notes: string
    }
    expect(payload.obverseImage).toBe(obverse)
    expect(payload.reverseImage).toBe(reverse)
    expect(payload.hintImages).toEqual([hint])
    expect(payload.notes).toBe('Weight 3.2 g')
  })

  it('saved-coin mode: hides the obverse upload slot and only requires the missing reverse photo', async () => {
    Object.defineProperty(URL, 'createObjectURL', { value: () => 'blob:deep-analysis', configurable: true })
    Object.defineProperty(URL, 'revokeObjectURL', { value: () => {}, configurable: true })
    const wrapper = mount(DeepAnalysisStartPanel, {
      props: { coinId: 42, hasExistingObverse: true, hasExistingReverse: false },
    })

    expect(wrapper.text()).toContain("Using this coin's existing obverse photo")
    expect(wrapper.findAll('input[type="file"][capture="environment"]')).toHaveLength(1)

    const startButton = wrapper.findAll('button').find((b) => b.text().includes('Start Deep Analysis'))
    await startButton!.trigger('click')
    expect(wrapper.text()).toContain('Obverse and reverse photos are both required to start Deep Analysis.')
    expect(wrapper.emitted('submit')).toBeFalsy()

    const reverseInput = wrapper.findAll('input[type="file"]').find((i) => i.attributes('capture') === 'environment')
    Object.defineProperty(reverseInput!.element, 'files', { value: [makeFile('reverse.jpg')], configurable: true })
    await reverseInput!.trigger('change')
    await startButton!.trigger('click')

    expect(wrapper.emitted('submit')).toBeTruthy()
    const payload = wrapper.emitted('submit')![0][0] as { coinId?: number; obverseImage: File | null; reverseImage: File | null }
    expect(payload.coinId).toBe(42)
    expect(payload.obverseImage).toBeNull()
    expect(payload.reverseImage?.name).toBe('reverse.jpg')
  })

  it('saved-coin mode: submits immediately with no images when both roles already exist', async () => {
    const wrapper = mount(DeepAnalysisStartPanel, {
      props: { coinId: 42, hasExistingObverse: true, hasExistingReverse: true },
    })

    expect(wrapper.findAll('input[type="file"][capture="environment"]')).toHaveLength(0)

    const startButton = wrapper.findAll('button').find((b) => b.text().includes('Start Deep Analysis'))
    await startButton!.trigger('click')

    expect(wrapper.emitted('submit')).toBeTruthy()
    const payload = wrapper.emitted('submit')![0][0] as { coinId?: number; obverseImage: File | null; reverseImage: File | null }
    expect(payload.coinId).toBe(42)
    expect(payload.obverseImage).toBeNull()
    expect(payload.reverseImage).toBeNull()
  })
})
