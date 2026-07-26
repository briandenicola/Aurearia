import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetCreationWizard from '@/components/sets/SetCreationWizard.vue'

describe('SetCreationWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('emits backend contract field creationMode for dynamic tracker sets', async () => {
    const wrapper = mount(SetCreationWizard, { attachTo: document.body })
    await flushPromises()

    await wrapper.find('#setType').setValue('tracker')
    await wrapper.find('#trackerCreationMode').setValue('dynamic')
    await wrapper.find('#trackerPrompt').setValue('  all American wheat pennies  ')
    await wrapper.find('#setName').setValue('Wheat Cents')
    await wrapper.find('form').trigger('submit.prevent')

    const submitEvents = wrapper.emitted('submit') as unknown[][]
    expect(submitEvents).toBeTruthy()
    const payload = submitEvents[0][0] as Record<string, unknown>

    expect(payload.creationMode).toBe('dynamic')
    expect(payload.trackerPrompt).toBe('all American wheat pennies')
    expect(payload).not.toHaveProperty('trackerCreationMode')
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
