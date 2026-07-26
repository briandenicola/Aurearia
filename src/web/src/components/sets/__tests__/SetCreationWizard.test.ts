import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetCreationWizard from '@/components/sets/SetCreationWizard.vue'

describe('SetCreationWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('blocks direct agentic set creation while proposal workflow is pending', async () => {
    const wrapper = mount(SetCreationWizard, { attachTo: document.body })
    await flushPromises()

    await wrapper.find('#setType').setValue('agentic')
    await wrapper.find('#agenticPrompt').setValue('  All US Silver Quarters from 1940s to 1960s  ')
    await wrapper.find('#setName').setValue('US Silver Quarters')
    await wrapper.find('form').trigger('submit.prevent')

    expect(wrapper.text()).toContain('Agentic builder unavailable')
    expect(wrapper.text()).toContain('No Agentic set will be created until a proposal is generated and approved.')
    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeDefined()
    expect(wrapper.emitted('submit')).toBeUndefined()
    expect(wrapper.find('#trackerCreationMode').exists()).toBe(false)
  })

  it('shows goal completion guidance and hides legacy goal target fields', async () => {
    const wrapper = mount(SetCreationWizard, { attachTo: document.body })
    await flushPromises()

    await wrapper.find('#setType').setValue('goal')

    expect(wrapper.find('#templateId').exists()).toBe(false)
    expect(wrapper.find('#csvTargets').exists()).toBe(false)
    expect(wrapper.find('#targetCompletionDate').exists()).toBe(false)
    expect(wrapper.text()).toContain('Goal sets track both collection and wishlist members.')
    expect(wrapper.text()).toContain('2 / (2 + 5) = 28.6%')
  })

  it('does not emit obsolete template/csv/date values for goal sets', async () => {
    const wrapper = mount(SetCreationWizard, {
      attachTo: document.body,
      props: {
        initialValue: {
          setType: 'goal',
          templateId: 'roman-template',
          targetCompletionDate: '2027-01-01',
        },
      },
    })
    await flushPromises()

    await wrapper.find('#setName').setValue('Roman Goal Set')
    await wrapper.find('form').trigger('submit.prevent')

    const submitEvents = wrapper.emitted('submit') as unknown[][]
    expect(submitEvents).toBeTruthy()
    const payload = submitEvents[0][0] as Record<string, unknown>
    const csvArg = submitEvents[0][1]

    expect(payload.setType).toBe('goal')
    expect(payload.templateId).toBeUndefined()
    expect(payload.targetCompletionDate).toBeUndefined()
    expect(csvArg).toBeUndefined()
  })
})
