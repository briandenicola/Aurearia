import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseToggle from '../BaseToggle.vue'

describe('BaseToggle', () => {
  it('renders an accessible checkbox reflecting the model value', () => {
    const wrapper = mount(BaseToggle, { props: { modelValue: true, label: 'Enable thing' } })
    const input = wrapper.find('input[type="checkbox"]')
    expect((input.element as HTMLInputElement).checked).toBe(true)
    expect(input.attributes('aria-label')).toBe('Enable thing')
  })

  it('emits the new value when toggled', async () => {
    const wrapper = mount(BaseToggle, { props: { modelValue: false, label: 'Thing' } })
    await wrapper.find('input').setValue(true)
    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])
  })

  it('marks the input disabled so it cannot be toggled', () => {
    const wrapper = mount(BaseToggle, { props: { modelValue: false, disabled: true, label: 'Thing' } })
    expect(wrapper.find('input').attributes('disabled')).toBeDefined()
    expect(wrapper.find('span').classes()).not.toContain('cursor-pointer')
  })

  // The two sizes replace the three that had drifted apart across 15 files (#784).
  it('offers exactly the two standard sizes', () => {
    const md = mount(BaseToggle, { props: { modelValue: false, label: 'Thing' } })
    const sm = mount(BaseToggle, { props: { modelValue: false, size: 'sm', label: 'Thing' } })
    expect(md.find('label').classes()).toContain('h-7')
    expect(md.find('span').classes()).toContain('after:h-[22px]')
    expect(sm.find('label').classes()).toContain('h-[22px]')
    expect(sm.find('span').classes()).toContain('after:h-4')
  })

  it('keeps a visible focus ring on the track', () => {
    const wrapper = mount(BaseToggle, { props: { modelValue: false, label: 'Thing' } })
    expect(wrapper.find('label').classes()).toContain('focus-within:outline-gold')
  })

  it('forwards attributes to the input rather than the wrapper', () => {
    const wrapper = mount(BaseToggle, { props: { modelValue: false, label: 'Thing' }, attrs: { id: 'thing-toggle' } })
    expect(wrapper.find('input').attributes('id')).toBe('thing-toggle')
    expect(wrapper.find('label').attributes('id')).toBeUndefined()
  })

  it('puts class and style on the wrapper, not the hidden input', () => {
    const wrapper = mount(BaseToggle, {
      props: { modelValue: false, label: 'Thing' },
      attrs: { class: 'ml-auto', style: 'margin-top: 4px' },
    })
    expect(wrapper.find('label').classes()).toContain('ml-auto')
    expect(wrapper.find('input').classes()).not.toContain('ml-auto')
    expect(wrapper.find('label').attributes('style')).toContain('margin-top')
  })

  // A call site may keep its own native @change handler (SettingsAccountSection
  // reverts an unchecked public-collection toggle). Both it and the internal
  // emit must fire, and the emitted value must follow the DOM after the handler.
  it('still emits when a call site adds its own change listener', async () => {
    const seen: boolean[] = []
    const wrapper = mount(BaseToggle, {
      props: { modelValue: false, label: 'Thing' },
      attrs: {
        onChange: (event: Event) => {
          const input = event.target as HTMLInputElement
          seen.push(input.checked)
          input.checked = false
        },
      },
    })
    await wrapper.find('input').setValue(true)
    expect(seen).toEqual([true])
    expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
  })
})
