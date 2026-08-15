import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepAnalysisEntryButton from '../DeepAnalysisEntryButton.vue'

describe('DeepAnalysisEntryButton', () => {
  it('emits click and exposes an accessible label', async () => {
    const wrapper = mount(DeepAnalysisEntryButton)
    expect(wrapper.find('button').attributes('aria-label')).toBe('Deep Analysis')

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('click')).toBeTruthy()
  })

  it('disables the button when disabled prop is set', () => {
    const wrapper = mount(DeepAnalysisEntryButton, { props: { disabled: true } })
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })

  it('supports a custom label', () => {
    const wrapper = mount(DeepAnalysisEntryButton, { props: { label: 'Run Deep Analysis' } })
    expect(wrapper.text()).toContain('Run Deep Analysis')
  })

  it('renders a lucide icon and contains no emoji (constitution: no emojis in UI)', () => {
    const wrapper = mount(DeepAnalysisEntryButton)
    // The microscope is an accessible-hidden SVG icon, never an emoji glyph.
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.html()).not.toMatch(/\p{Extended_Pictographic}/u)
  })
})
