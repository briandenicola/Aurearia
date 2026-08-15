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
})
