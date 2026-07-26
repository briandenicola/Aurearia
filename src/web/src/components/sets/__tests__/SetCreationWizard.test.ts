import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetCreationWizard from '@/components/sets/SetCreationWizard.vue'

const mockGetSetTemplates = vi.fn()

vi.mock('@/api/client', () => ({
  getSetTemplates: (...args: unknown[]) => mockGetSetTemplates(...args),
}))

describe('SetCreationWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetSetTemplates.mockResolvedValue({ data: { templates: [] } })
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
})
