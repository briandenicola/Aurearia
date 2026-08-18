import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepReportPanel from '../DeepReportPanel.vue'
import type { DeepReport } from '@/types'

function baseReport(overrides: Partial<DeepReport> = {}): DeepReport {
  return {
    schemaVersion: 1,
    narrative: 'This denarius likely depicts Trajan based on obverse legend evidence.',
    coverage: [
      { provider: 'nomisma', status: 'contributed' },
      { provider: 'numista', status: 'contributed' },
      { provider: 'ngc', status: 'not_automated', note: 'Link out only', linkOut: 'https://ngccoin.com/x' },
      { provider: 'ocre', status: 'not_automated' },
      { provider: 'rpc', status: 'unavailable' },
    ],
    partialSuccess: false,
    generatedAt: '2030-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('DeepReportPanel', () => {
  it('renders the narrative text', () => {
    const wrapper = mount(DeepReportPanel, { props: { report: baseReport() } })
    expect(wrapper.text()).toContain('This denarius likely depicts Trajan')
  })

  it('does not render model thinking, signatures, or encoded transport content', () => {
    const wrapper = mount(DeepReportPanel, {
      props: {
        report: baseReport({
          narrative: "[{'type': 'thinking', 'signature': 'secret'}, {'type': 'text', 'text': 'result'}]",
        }),
      },
    })
    expect(wrapper.text()).toContain('could not be displayed safely')
    expect(wrapper.text()).not.toContain('signature')
    expect(wrapper.text()).not.toContain('secret')
  })

  it('shows a partial-success banner only when the report is partial', () => {
    const partialWrapper = mount(DeepReportPanel, { props: { report: baseReport({ partialSuccess: true }) } })
    expect(partialWrapper.text()).toContain('Partial results')

    const completeWrapper = mount(DeepReportPanel, { props: { report: baseReport({ partialSuccess: false }) } })
    expect(completeWrapper.text()).not.toContain('Partial results')
  })

  it('renders every provider coverage status distinctly, never collapsing not_automated/unavailable into no_match', () => {
    const wrapper = mount(DeepReportPanel, { props: { report: baseReport() } })
    const text = wrapper.text()
    expect(text).toContain('Manual verification')
    expect(text).toContain('Unavailable')
    expect(text).not.toContain('No match')
  })

  it('links out to NGC without implying an automated result', () => {
    const wrapper = mount(DeepReportPanel, { props: { report: baseReport() } })
    const link = wrapper.find('a[href="https://ngccoin.com/x"]')
    expect(link.exists()).toBe(true)
  })

  it('renders disagreements with each provider claim and citation, without hiding provider status', () => {
    const wrapper = mount(DeepReportPanel, {
      props: {
        report: baseReport({
          disagreements: [
            {
              field: 'ruler',
              resolution: 'unresolved',
              claims: [
                { field: 'ruler', value: 'Trajan', citation: 'https://nomisma.org/a' },
                { field: 'ruler', value: 'Hadrian', citation: 'https://numista.com/b' },
              ],
            },
          ],
        }),
      },
    })
    expect(wrapper.text()).toContain('ruler')
    expect(wrapper.text()).toContain('Trajan')
    expect(wrapper.text()).toContain('Hadrian')
    expect(wrapper.findAll('a[href]').length).toBeGreaterThanOrEqual(2)
  })

  it('renders unresolved questions when present', () => {
    const wrapper = mount(DeepReportPanel, {
      props: { report: baseReport({ unresolvedQuestions: ['Is the mint mark legible?'] }) },
    })
    expect(wrapper.text()).toContain('Is the mint mark legible?')
  })

  it('renders nothing extra when quickLookupOutcome is "ok"', () => {
    const wrapper = mount(DeepReportPanel, { props: { report: baseReport({ quickLookupOutcome: 'ok' }) } })
    expect(wrapper.text()).not.toContain('Quick lookup')
  })

  it('renders a quiet note when quickLookupOutcome is "no_data"', () => {
    const wrapper = mount(DeepReportPanel, { props: { report: baseReport({ quickLookupOutcome: 'no_data' }) } })
    expect(wrapper.text()).toContain('Quick lookup completed but found no supporting cert or catalog data')
    expect(wrapper.text()).not.toContain('did not finish')
  })

  it('renders a prominent, legible warning when quickLookupOutcome is "unavailable"', () => {
    const wrapper = mount(DeepReportPanel, { props: { report: baseReport({ quickLookupOutcome: 'unavailable' }) } })
    expect(wrapper.text()).toContain('Quick lookup incomplete')
    expect(wrapper.text()).toContain('did not finish before this report was generated')
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  it('renders exactly as today when quickLookupOutcome is absent (pre-change report)', () => {
    const report = baseReport()
    delete (report as Partial<DeepReport>).quickLookupOutcome
    const wrapper = mount(DeepReportPanel, { props: { report } })
    expect(wrapper.text()).not.toContain('Quick lookup')
    expect(wrapper.text()).not.toContain('undefined')
  })

  it('renders no attribution section when attributions are absent/empty', () => {
    const wrapper = mount(DeepReportPanel, { props: { report: baseReport() } })
    expect(wrapper.text()).not.toContain('Attribution & licensing')
    expect(wrapper.find('.ocre-attribution').exists()).toBe(false)
  })

  it('renders the OCRE attribution when the report includes an ocre attribution entry', () => {
    const wrapper = mount(DeepReportPanel, {
      props: {
        report: baseReport({
          attributions: [
            {
              provider: 'ocre',
              text: 'Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society \u2014 ODbL 1.0.',
              identifier: 'https://numismatics.org/ocre/id/ric.1(2).aug.1',
            },
          ],
        }),
      },
    })
    expect(wrapper.text()).toContain('Attribution & licensing')
    expect(wrapper.find('.ocre-attribution').exists()).toBe(true)
    expect(wrapper.find('a[href="https://opendatacommons.org/licenses/odbl/1-0/"]').exists()).toBe(true)
    expect(wrapper.find('a[href="https://numismatics.org/ocre/id/ric.1(2).aug.1"]').exists()).toBe(true)
  })

  it('keeps OCRE attribution distinct from a co-present Nomisma attribution entry', () => {
    const wrapper = mount(DeepReportPanel, {
      props: {
        report: baseReport({
          attributions: [
            {
              provider: 'ocre',
              text: 'Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society \u2014 ODbL 1.0.',
              identifier: 'https://numismatics.org/ocre/id/ric.1(2).aug.1',
            },
            {
              provider: 'nomisma',
              text: 'Data: Nomisma.org (CC BY)',
              identifier: 'https://nomisma.org/id/augustus',
            },
          ],
        }),
      },
    })
    // OCRE renders via its dedicated distinct component; Nomisma via generic text.
    expect(wrapper.find('.ocre-attribution').exists()).toBe(true)
    expect(wrapper.text()).toContain('ODbL 1.0')
    expect(wrapper.text()).toContain('Data: Nomisma.org (CC BY)')
  })

  describe('image hypothesis panel (T091, RD-6)', () => {
    it('hides the section entirely for a pre-351 report with no image_hypothesis key', () => {
      const wrapper = mount(DeepReportPanel, { props: { report: baseReport() } })
      expect(wrapper.find('details').exists()).toBe(false)
      expect(wrapper.text()).not.toContain('What the images alone said')
    })

    it('renders collapsed by default when image_hypothesis is present', () => {
      const wrapper = mount(DeepReportPanel, {
        props: {
          report: baseReport({
            image_hypothesis: {
              ruler: { value: 'Maximinus I (Thrax)', confidence: 0.86 },
              legible: true,
              observations: 'Silvered surfaces, high relief portrait.',
            },
          }),
        },
      })
      const details = wrapper.find('details')
      expect(details.exists()).toBe(true)
      expect(details.attributes('open')).toBeUndefined()
      expect(details.text()).toContain('What the images alone said')
    })

    it('renders each typed hypothesis field with its own confidence, plus observations', () => {
      const wrapper = mount(DeepReportPanel, {
        props: {
          report: baseReport({
            image_hypothesis: {
              ruler: { value: 'Maximinus I (Thrax)', confidence: 0.86 },
              denomination: { value: 'Denarius', confidence: 0.9 },
              legible: true,
              observations: 'Silvered surfaces, high relief portrait.',
            },
          }),
        },
      })
      const text = wrapper.text()
      expect(text).toContain('Maximinus I (Thrax)')
      expect(text).toContain('86% confidence')
      expect(text).toContain('Denarius')
      expect(text).toContain('90% confidence')
      expect(text).toContain('Silvered surfaces, high relief portrait.')
    })

    it('states plainly when legible is false, visibly distinct from an empty/dropped result', () => {
      const wrapper = mount(DeepReportPanel, {
        props: {
          report: baseReport({
            image_hypothesis: { legible: false },
          }),
        },
      })
      const text = wrapper.text()
      expect(text).toContain('not legible enough')
      expect(text).not.toContain('did not find any coin details')
    })

    it('distinguishes a legible-but-empty hypothesis from an unreadable one', () => {
      const wrapper = mount(DeepReportPanel, {
        props: {
          report: baseReport({
            image_hypothesis: { legible: true, observations: '' },
          }),
        },
      })
      const text = wrapper.text()
      expect(text).toContain('did not find any coin details')
      expect(text).not.toContain('not legible enough')
    })
  })

  describe('disagreement claim source marking (T089, T092)', () => {
    it('marks a citation-less disagreement claim as image-derived, distinct from a cited provider claim', () => {
      const wrapper = mount(DeepReportPanel, {
        props: {
          report: baseReport({
            disagreements: [
              {
                field: 'mint',
                resolution: 'unresolved',
                claims: [
                  { field: 'mint', value: 'Rome', citation: 'https://en.numista.com/catalogue/pieces1.html' },
                  { field: 'mint', value: 'Antioch' },
                ],
              },
            ],
          }),
        },
      })
      const text = wrapper.text()
      expect(text).toContain('Rome')
      expect(text).toContain('Antioch')
      expect(text).toContain('From images')
      const disagreementLinks = wrapper.findAll('.chip-sm').filter((el) => el.text() === 'From images')
      expect(disagreementLinks).toHaveLength(1)
      const romeLink = wrapper.find('a[href="https://en.numista.com/catalogue/pieces1.html"]')
      expect(romeLink.exists()).toBe(true)
    })
  })
})
