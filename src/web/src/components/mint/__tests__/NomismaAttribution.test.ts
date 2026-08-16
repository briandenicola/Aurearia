import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import NomismaAttribution from '@/components/mint/NomismaAttribution.vue'

describe('NomismaAttribution', () => {
  it('renders nothing when no uri is linked', () => {
    const wrapper = mount(NomismaAttribution, { props: { uri: null } })
    expect(wrapper.find('p').exists()).toBe(false)
  })

  it('renders the exact attribution string with the specific concept link and the CC BY 4.0 license link', () => {
    const wrapper = mount(NomismaAttribution, { props: { uri: 'http://nomisma.org/id/roma' } })

    expect(wrapper.text().replace(/\s+/g, ' ').trim()).toBe('Source: Nomisma.org · CC BY 4.0')

    const links = wrapper.findAll('a')
    expect(links).toHaveLength(2)
    expect(links[0].text()).toBe('Nomisma.org')
    expect(links[0].attributes('href')).toBe('http://nomisma.org/id/roma')
    expect(links[1].text()).toBe('CC BY 4.0')
    expect(links[1].attributes('href')).toBe('https://creativecommons.org/licenses/by/4.0/')
  })
})
