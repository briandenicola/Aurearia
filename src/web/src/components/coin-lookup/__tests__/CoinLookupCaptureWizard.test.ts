import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import CoinLookupCaptureWizard from '../CoinLookupCaptureWizard.vue'

function image(name: string) {
  return {
    file: new File(['image'], name, { type: 'image/jpeg' }),
    preview: `blob:${name}`,
  }
}

function mountWizard(overrides: Record<string, unknown> = {}) {
  return mount(CoinLookupCaptureWizard, {
    props: {
      obverse: null,
      reverse: null,
      notesImage: null,
      notes: '',
      submitting: false,
      preparingImage: false,
      uploadError: '',
      ...overrides,
    },
    global: {
      stubs: {
        InlineCameraCapturePanel: {
          template: '<div class="camera-stub"></div>',
          methods: { stopCamera() {} },
        },
      },
    },
  })
}

describe('CoinLookupCaptureWizard', () => {
  it('shows only the current step guidance before the first image', () => {
    const wrapper = mountWizard()

    expect(wrapper.text()).toContain('Step 1 of 3')
    expect(wrapper.text()).toContain('Add the obverse')
    expect(wrapper.find('ol').exists()).toBe(false)
    expect(wrapper.find('[aria-label="Add reverse image"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).not.toContain('Analyze Photos')
  })

  it('offers immediate analysis and compact progression after the obverse', async () => {
    const wrapper = mountWizard({ obverse: image('obverse.jpg') })

    expect(wrapper.find('.camera-stub').exists()).toBe(false)
    const analyze = wrapper.findAll('button').find(button => button.text().includes('Analyze Photos'))
    await analyze?.trigger('click')
    expect(wrapper.emitted('analyze')).toHaveLength(1)

    await wrapper.find('[aria-label="Add reverse image"]').trigger('click')
    expect(wrapper.text()).toContain('Step 2 of 3')
    expect(wrapper.text()).toContain('Add the reverse')
    expect(wrapper.find('.camera-stub').exists()).toBe(true)
    expect(wrapper.find('[aria-label="Previous step"]').exists()).toBe(true)

    await wrapper.find('[aria-label="Add notes"]').trigger('click')
    expect(wrapper.text()).toContain('Step 3 of 3')
    expect(wrapper.text()).toContain('Add supporting evidence')
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('textarea').attributes('maxlength')).toBe('2000')
  })

  it('hides each camera after that step has an image', async () => {
    const wrapper = mountWizard({
      obverse: image('obverse.jpg'),
      reverse: image('reverse.jpg'),
    })

    await wrapper.find('[aria-label="Add reverse image"]').trigger('click')
    expect(wrapper.find('.camera-stub').exists()).toBe(false)
    expect(wrapper.find('[aria-label="Remove reverse image"]').exists()).toBe(true)
  })

  it('uses the shared wizard for Deep Analysis and guides users to a missing reverse', async () => {
    const wrapper = mountWizard({
      obverse: image('obverse.jpg'),
      deepAnalysisEnabled: true,
    })

    await wrapper.findAll('button').find(button => button.text().includes('Deep Analysis'))!.trigger('click')
    expect(wrapper.text()).toContain('Step 2 of 3')
    expect(wrapper.text()).toContain('Add a reverse image before starting Deep Analysis.')
    expect(wrapper.emitted('deepAnalyze')).toBeUndefined()

    await wrapper.setProps({ reverse: image('reverse.jpg') })
    await wrapper.findAll('button').find(button => button.text().includes('Deep Analysis'))!.trigger('click')
    expect(wrapper.emitted('deepAnalyze')).toHaveLength(1)
  })

  it('does not analyze while a gallery image is still being prepared', () => {
    const wrapper = mountWizard({
      obverse: image('obverse.jpg'),
      preparingImage: true,
    })

    const analyze = wrapper.findAll('button').find(button => button.text().includes('Preparing image...'))
    expect(analyze?.attributes('disabled')).toBeDefined()
  })
})
