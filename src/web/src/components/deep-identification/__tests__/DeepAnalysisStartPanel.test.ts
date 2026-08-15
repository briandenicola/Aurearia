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
})
