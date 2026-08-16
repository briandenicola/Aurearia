import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import OCREAttribution from '../OCREAttribution.vue'
import NomismaAttribution from '@/components/mint/NomismaAttribution.vue'

const OCRE_TYPE_URI = 'https://numismatics.org/ocre/id/ric.1(2).aug.1'
const ODBL_LICENSE = 'https://opendatacommons.org/licenses/odbl/1-0/'

describe('OCREAttribution', () => {
  it('renders nothing when no canonical type link is supplied', () => {
    const wrapper = mount(OCREAttribution, { props: { uri: '' } })
    expect(wrapper.find('.ocre-attribution').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('renders the exact ODbL / ANS attribution text', () => {
    const wrapper = mount(OCREAttribution, { props: { uri: OCRE_TYPE_URI } })
    // Normalize whitespace/newlines from the template for an exact compare.
    const text = wrapper.text().replace(/\s+/g, ' ').trim()
    expect(text).toBe(
      'Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society \u2014 ODbL 1.0.',
    )
  })

  it('links the canonical OCRE type and the ODbL license, both correctly targeted', () => {
    const wrapper = mount(OCREAttribution, { props: { uri: OCRE_TYPE_URI } })
    const hrefs = wrapper.findAll('a').map(a => a.attributes('href'))
    expect(hrefs).toContain(OCRE_TYPE_URI)
    expect(hrefs).toContain(ODBL_LICENSE)
    for (const a of wrapper.findAll('a')) {
      expect(a.attributes('target')).toBe('_blank')
      expect(a.attributes('rel')).toContain('noopener')
      expect(a.attributes('rel')).toContain('noreferrer')
    }
  })

  it('is visually/structurally distinct from NomismaAttribution', () => {
    const ocre = mount(OCREAttribution, { props: { uri: OCRE_TYPE_URI } })
    const nomisma = mount(NomismaAttribution, { props: { uri: 'https://nomisma.org/id/augustus' } })
    expect(ocre.find('.ocre-attribution').exists()).toBe(true)
    expect(ocre.find('.nomisma-attribution').exists()).toBe(false)
    expect(nomisma.find('.nomisma-attribution').exists()).toBe(true)
    expect(nomisma.find('.ocre-attribution').exists()).toBe(false)
    // OCRE attribution is unique in naming ODbL / ANS; Nomisma is CC BY.
    expect(ocre.text()).toContain('ODbL 1.0')
    expect(nomisma.text()).not.toContain('ODbL')
  })
})
