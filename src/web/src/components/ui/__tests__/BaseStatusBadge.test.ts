import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseStatusBadge from '../BaseStatusBadge.vue'

describe('BaseStatusBadge', () => {
  it('renders slot content', () => {
    const wrapper = mount(BaseStatusBadge, { slots: { default: 'success' } })
    expect(wrapper.text()).toBe('success')
  })

  it('defaults to the neutral tone', () => {
    const wrapper = mount(BaseStatusBadge)
    expect(wrapper.attributes('style')).toContain('var(--status-neutral-bg)')
    expect(wrapper.attributes('style')).toContain('var(--status-neutral-fg)')
  })

  it('drives colour from status tokens for every tone', () => {
    for (const tone of ['success', 'error', 'warning', 'info', 'neutral'] as const) {
      const wrapper = mount(BaseStatusBadge, { props: { tone } })
      const style = wrapper.attributes('style') ?? ''
      expect(style).toContain(`var(--status-${tone}-bg)`)
      expect(style).toContain(`var(--status-${tone}-fg)`)
      expect(style).not.toContain('rgba(')
    }
  })

  it('keeps one shared badge shape', () => {
    const wrapper = mount(BaseStatusBadge)
    expect(wrapper.classes()).toEqual([
      'inline-block',
      'rounded-full',
      'px-2',
      'py-[0.15rem]',
      'text-label',
      'font-semibold',
    ])
  })
})
