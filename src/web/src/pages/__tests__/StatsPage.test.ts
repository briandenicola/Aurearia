import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import StatsPage from '@/pages/StatsPage.vue'

const routerLinkStub = {
  props: ['to'],
  template: '<a :href="to"><slot /></a>',
}

describe('StatsPage', () => {
  it('renders landing cards for stats subviews', () => {
    const wrapper = mount(StatsPage, {
      global: { stubs: { RouterLink: routerLinkStub } },
    })

    expect(wrapper.text()).toContain('Mint Map')
    expect(wrapper.text()).toContain('Timeline')
    expect(wrapper.text()).toContain('Collection Distribution')
    expect(wrapper.text()).toContain('Collection Health')
    expect(wrapper.text()).toContain('Value Over Time')
    expect(wrapper.html()).toContain('href="/stats/mint-map"')
    expect(wrapper.html()).toContain('href="/stats/timeline"')
    expect(wrapper.html()).toContain('href="/stats/distribution"')
  })
})
