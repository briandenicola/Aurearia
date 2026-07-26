import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetCreationWizard from '@/components/sets/SetCreationWizard.vue'

describe('SetCreationWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('emits an agentic submit payload with the prompt instead of creating a set locally', async () => {
    const wrapper = mount(SetCreationWizard, { attachTo: document.body })
    await flushPromises()

    await wrapper.find('#setType').setValue('agentic')
    await wrapper.find('#agenticPrompt').setValue('  All US Silver Quarters from 1940s to 1960s  ')
    await wrapper.find('#setName').setValue('US Silver Quarters')

    expect(wrapper.text()).toContain('How agentic sets work')
    expect(wrapper.find('button[type="submit"]').attributes('disabled')).toBeUndefined()
    expect(wrapper.find('button[type="submit"]').text()).toBe('Create')

    await wrapper.find('form').trigger('submit.prevent')

    const submitEvents = wrapper.emitted('submit') as unknown[][]
    expect(submitEvents).toBeTruthy()
    const payload = submitEvents[0][0] as Record<string, unknown>
    const csvArg = submitEvents[0][1]

    expect(payload.setType).toBe('agentic')
    expect(payload.agenticPrompt).toBe('All US Silver Quarters from 1940s to 1960s')
    expect(csvArg).toBeUndefined()
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
