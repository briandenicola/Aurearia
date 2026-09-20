import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import AdminLogsSection from '../AdminLogsSection.vue'

describe('AdminLogsSection', () => {
  it('keeps the desktop log controls on one row without wrapping button labels', () => {
    const wrapper = mount(AdminLogsSection, {
      props: {
        logs: [],
        loading: false,
        filter: '',
        autoRefresh: false,
      },
    })

    const toolbar = wrapper.get('section > div')
    expect(toolbar.classes()).toContain('sm:flex-nowrap')

    const filter = wrapper.get('select')
    expect(filter.classes()).toContain('sm:max-w-56')

    for (const button of wrapper.findAll('button')) {
      expect(button.classes()).toContain('shrink-0')
      expect(button.classes()).toContain('whitespace-nowrap')
    }
  })
})
