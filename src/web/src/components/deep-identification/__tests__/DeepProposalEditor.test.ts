import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import DeepProposalEditor from '../DeepProposalEditor.vue'
import type { DeepProposal } from '@/types'

function baseProposal(): DeepProposal {
  return {
    schemaVersion: 1,
    fields: {
      denomination: {
        proposed: 'Denarius',
        ownerEdited: false,
        ownerValue: null,
        accepted: null,
      },
      ruler: {
        proposed: 'Trajan',
        ownerEdited: false,
        ownerValue: null,
        accepted: null,
      },
    },
  }
}

describe('DeepProposalEditor', () => {
  it('renders each proposed field with the AI value visibly distinct from an owner edit', () => {
    const wrapper = mount(DeepProposalEditor, { props: { proposal: baseProposal() } })
    expect(wrapper.text()).toContain('Denarius')
    expect(wrapper.text()).toContain('AI proposed')
    expect(wrapper.text()).not.toContain('Edited by you')
  })

  it('marks a field as owner-edited once its input changes, distinct from the AI-proposed label', async () => {
    const wrapper = mount(DeepProposalEditor, { props: { proposal: baseProposal() } })
    const input = wrapper.find('#deep-proposal-field-denomination')
    await input.setValue('Antoninianus')
    expect(wrapper.emitted('update-field')?.[0]).toEqual(['denomination', { ownerValue: 'Antoninianus' }])
  })

  it('emits accept/reject decisions per field without writing anything itself', async () => {
    const wrapper = mount(DeepProposalEditor, { props: { proposal: baseProposal() } })
    const acceptButtons = wrapper.findAll('button[aria-pressed]').filter((b) => b.text() === 'Accept')
    await acceptButtons[0].trigger('click')
    expect(wrapper.emitted('update-field')?.[0]).toEqual(['denomination', { accepted: true }])
    expect(wrapper.emitted('confirm')).toBeUndefined()
  })

  it('disables confirm until at least one field has been explicitly accepted', async () => {
    const wrapper = mount(DeepProposalEditor, { props: { proposal: baseProposal() } })
    const confirmButton = wrapper.find('button:not([aria-pressed])')
    expect(confirmButton.attributes('disabled')).toBeDefined()

    await wrapper.setProps({
      proposal: {
        ...baseProposal(),
        fields: {
          ...baseProposal().fields,
          denomination: { ...baseProposal().fields.denomination, accepted: true },
        },
      },
    })
    expect(wrapper.find('button:not([aria-pressed])').attributes('disabled')).toBeUndefined()
  })

  it('does not emit confirm merely from rendering or field edits', async () => {
    const wrapper = mount(DeepProposalEditor, { props: { proposal: baseProposal() } })
    await wrapper.find('#deep-proposal-field-ruler').setValue('Hadrian')
    expect(wrapper.emitted('confirm')).toBeUndefined()
  })

  it('emits confirm only on explicit button click once enabled', async () => {
    const proposal = baseProposal()
    proposal.fields.denomination.accepted = true
    const wrapper = mount(DeepProposalEditor, { props: { proposal } })
    const confirmButton = wrapper.find('button:not([aria-pressed])')
    await confirmButton.trigger('click')
    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('disables confirm and shows an applying label while a submission is in flight', () => {
    const proposal = baseProposal()
    proposal.fields.denomination.accepted = true
    const wrapper = mount(DeepProposalEditor, { props: { proposal, applying: true } })
    const confirmButton = wrapper.find('button:not([aria-pressed])')
    expect(confirmButton.attributes('disabled')).toBeDefined()
    expect(confirmButton.text()).toContain('Applying')
  })

  it('uses context-specific save labels and a wrapping notes editor', () => {
    const proposal = baseProposal()
    proposal.fields.notes = {
      proposed: 'Detailed report notes',
      ownerEdited: false,
      ownerValue: null,
      accepted: true,
    }
    const wrapper = mount(DeepProposalEditor, {
      props: {
        proposal,
        actionLabel: 'Save as Draft',
        applyingLabel: 'Saving...',
      },
    })
    expect(wrapper.text()).toContain('Save as Draft')
    expect(wrapper.find('textarea#deep-proposal-field-notes').exists()).toBe(true)
  })

  it('surfaces the OCRE attribution beside a coin_type field whose evidence cites an OCRE type', () => {
    const proposal: DeepProposal = {
      schemaVersion: 1,
      fields: {
        coin_type: {
          proposed: 'RIC I Augustus 1',
          confidence: 0.82,
          evidence: [
            {
              field: 'coin_type',
              value: 'RIC I Augustus 1',
              confidence: 0.82,
              citation: 'https://numismatics.org/ocre/id/ric.1(2).aug.1',
            },
          ],
          ownerEdited: false,
          ownerValue: null,
          accepted: null,
        },
      },
    }
    const wrapper = mount(DeepProposalEditor, { props: { proposal } })
    expect(wrapper.find('.ocre-attribution').exists()).toBe(true)
    expect(wrapper.find('a[href="https://opendatacommons.org/licenses/odbl/1-0/"]').exists()).toBe(true)
    expect(wrapper.find('a[href="https://numismatics.org/ocre/id/ric.1(2).aug.1"]').exists()).toBe(true)
  })

  it('does not surface an OCRE attribution for a field without OCRE-cited evidence', () => {
    const wrapper = mount(DeepProposalEditor, { props: { proposal: baseProposal() } })
    expect(wrapper.find('.ocre-attribution').exists()).toBe(false)
  })

  it('marks a field with an empty evidence array as image-derived, distinct from provider-cited fields', () => {
    const proposal: DeepProposal = {
      schemaVersion: 1,
      fields: {
        ruler: {
          proposed: 'Maximinus I (Thrax)',
          confidence: 0.86,
          evidence: [],
          ownerEdited: false,
          ownerValue: null,
          accepted: null,
        },
        denomination: {
          proposed: 'Denarius',
          confidence: 0.9,
          evidence: [{ field: 'denomination', value: 'Denarius', citation: 'https://en.numista.com/catalogue/pieces1.html' }],
          ownerEdited: false,
          ownerValue: null,
          accepted: null,
        },
      },
    }
    const wrapper = mount(DeepProposalEditor, { props: { proposal } })
    const items = wrapper.findAll('li')
    const rulerItem = items.find((item) => item.text().includes('Maximinus'))
    const denominationItem = items.find((item) => item.text().includes('Denarius'))
    expect(rulerItem?.text()).toContain('Image only')
    expect(denominationItem?.text()).not.toContain('Image only')
  })

  it('does not mark a field with no evidence array at all (e.g. the narrative-fallback notes field) as image-derived', () => {
    const proposal = baseProposal()
    proposal.fields.notes = { proposed: 'A silver denarius.', ownerEdited: false, ownerValue: null, accepted: null }
    const wrapper = mount(DeepProposalEditor, { props: { proposal } })
    const notesItem = wrapper.findAll('li').find((item) => item.text().includes('A silver denarius.'))
    expect(notesItem?.text()).not.toContain('Image only')
  })

  describe('RD-3 confidence-driven acceptance default (T120)', () => {
    function fieldProposal(overrides: Partial<DeepProposal['fields']['field']>): DeepProposal {
      return {
        schemaVersion: 1,
        fields: {
          field: {
            proposed: 'Some value',
            ownerEdited: false,
            ownerValue: null,
            accepted: null,
            ...overrides,
          },
        },
      }
    }

    it('renders an image-only field at confidence 0.85 as accepted by default (source does not gate acceptance)', () => {
      const wrapper = mount(DeepProposalEditor, {
        props: { proposal: fieldProposal({ confidence: 0.85, evidence: [] }) },
      })
      const acceptButton = wrapper.findAll('button[aria-pressed]').find((b) => b.text() === 'Accept')
      expect(acceptButton?.attributes('aria-pressed')).toBe('true')
    })

    it('renders a provider-corroborated field at confidence 0.40 as unaccepted', () => {
      const wrapper = mount(DeepProposalEditor, {
        props: {
          proposal: fieldProposal({
            confidence: 0.4,
            evidence: [{ field: 'field', value: 'Some value', citation: 'https://en.numista.com/catalogue/pieces1.html' }],
          }),
        },
      })
      const acceptButton = wrapper.findAll('button[aria-pressed]').find((b) => b.text() === 'Accept')
      expect(acceptButton?.attributes('aria-pressed')).toBe('false')
    })

    it('renders a field corroborated up from image confidence 0.62 to 0.72 (RD-2 +0.10) as accepted by default, crossing the threshold', () => {
      // RD-2: min(1.0, max(image, provider) + 0.10) applied once, no stacking.
      // 0.62 + 0.10 = 0.72, which the synthesis side would have already computed
      // before this reaches the proposal; the editor only needs to render
      // whatever final confidence it receives correctly against the threshold.
      const corroboratedConfidence = Math.min(1.0, 0.62 + 0.1)
      const wrapper = mount(DeepProposalEditor, {
        props: {
          proposal: fieldProposal({
            confidence: corroboratedConfidence,
            evidence: [{ field: 'field', value: 'Some value', citation: 'https://en.numista.com/catalogue/pieces1.html' }],
          }),
        },
      })
      const acceptButton = wrapper.findAll('button[aria-pressed]').find((b) => b.text() === 'Accept')
      expect(acceptButton?.attributes('aria-pressed')).toBe('true')
      expect(corroboratedConfidence).toBeGreaterThanOrEqual(0.7)
    })

    it('lets an explicit owner rejection override a confidence-based default acceptance', async () => {
      const wrapper = mount(DeepProposalEditor, {
        props: { proposal: fieldProposal({ confidence: 0.9, evidence: [] }) },
      })
      const rejectButton = wrapper.findAll('button[aria-pressed]').find((b) => b.text() === 'Reject')
      await rejectButton?.trigger('click')
      expect(wrapper.emitted('update-field')?.[0]).toEqual(['field', { accepted: false }])
    })
  })
})
