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
  it('shows required and optional steps before the first image', () => {
    const wrapper = mountWizard()

    expect(wrapper.text()).toContain('Step 1 of 3')
    expect(wrapper.text()).toContain('Add the obverse')
    expect(wrapper.text()).toContain('Required')
    expect(wrapper.text().match(/Optional/g)).toHaveLength(2)
    expect(wrapper.findAll('button').find(button => button.text().includes('Add Reverse'))?.attributes('disabled')).toBeDefined()
    expect(wrapper.text()).not.toContain('Analyze Photos')
  })

  it('offers immediate analysis or progression after the obverse', async () => {
    const wrapper = mountWizard({ obverse: image('obverse.jpg') })

    const analyze = wrapper.findAll('button').find(button => button.text().includes('Analyze Photos'))
    await analyze?.trigger('click')
    expect(wrapper.emitted('analyze')).toHaveLength(1)

    await wrapper.findAll('button').find(button => button.text().includes('Add Reverse'))?.trigger('click')
    expect(wrapper.text()).toContain('Step 2 of 3')
    expect(wrapper.text()).toContain('Add the reverse')

    await wrapper.findAll('button').find(button => button.text().includes('Add Notes'))?.trigger('click')
    expect(wrapper.text()).toContain('Step 3 of 3')
    expect(wrapper.text()).toContain('Add supporting evidence')
    expect(wrapper.find('textarea').exists()).toBe(true)
    expect(wrapper.find('textarea').attributes('maxlength')).toBe('2000')
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
