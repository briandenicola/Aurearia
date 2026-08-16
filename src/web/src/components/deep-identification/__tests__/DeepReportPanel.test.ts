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
    expect(text).toContain('Not automated')
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
})
