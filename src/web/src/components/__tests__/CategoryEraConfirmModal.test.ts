import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import CategoryEraConfirmModal from '../chat/CategoryEraConfirmModal.vue'

const CHAT_SIDEBAR_Z = 1400

describe('CategoryEraConfirmModal', () => {
  it('renders nothing when request is null', () => {
    const wrapper = mount(CategoryEraConfirmModal, {
      props: { request: null },
      global: { stubs: { Teleport: true } },
    })
    expect(wrapper.find('[class*="fixed"]').exists()).toBe(false)
  })

  it('overlay z-index exceeds CoinSearchChat sidebar z-index so the dialog is selectable', () => {
    // The CoinSearchChat sidebar uses z-[1400]. The CategoryEraConfirmModal is
    // teleported to <body> via <Teleport>; its overlay must have a higher z-index
    // so it appears above the sidebar. We assert the class token directly because
    // jsdom cannot measure visual stacking.
    const wrapper = mount(CategoryEraConfirmModal, {
      props: {
        request: {
          fieldLabel: 'Era',
          suggestedValue: 'AD190',
          options: ['Roman Imperial', 'Roman Republic'],
        },
      },
      global: { stubs: { Teleport: true } },
    })

    const overlay = wrapper.find('.fixed.inset-0[class*="z-["]')
    expect(overlay.exists()).toBe(true)
    const zMatch = overlay.classes().join(' ').match(/z-\[(\d+)\]/)
    expect(zMatch).not.toBeNull()
    const zValue = parseInt(zMatch![1], 10)
    expect(zValue).toBeGreaterThan(CHAT_SIDEBAR_Z)
  })

  it('emits choose with the selected option', async () => {
    const wrapper = mount(CategoryEraConfirmModal, {
      props: {
        request: {
          fieldLabel: 'Era',
          suggestedValue: 'AD190',
          options: ['Roman Imperial', 'Roman Republic'],
        },
      },
      global: { stubs: { Teleport: true } },
    })

    const buttons = wrapper.findAll('button')
    const romanImperial = buttons.find((b) => b.text() === 'Roman Imperial')
    expect(romanImperial).toBeTruthy()
    await romanImperial!.trigger('click')
    expect(wrapper.emitted('choose')).toEqual([['Roman Imperial']])
  })

  it('emits cancel when clicking the Cancel button', async () => {
    const wrapper = mount(CategoryEraConfirmModal, {
      props: {
        request: {
          fieldLabel: 'Era',
          suggestedValue: 'AD190',
          options: ['Roman Imperial'],
        },
      },
      global: { stubs: { Teleport: true } },
    })

    const cancelBtn = wrapper.findAll('button').find((b) => b.text() === 'Cancel')
    expect(cancelBtn).toBeTruthy()
    await cancelBtn!.trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })
})
