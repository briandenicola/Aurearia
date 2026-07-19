import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PwaUpdateBanner from '../PwaUpdateBanner.vue'

const mocks = vi.hoisted(() => ({
  updateAvailable: { value: false },
  refresh: vi.fn(),
}))

vi.mock('@/composables/usePwaUpdate', () => ({
  usePwaUpdate: () => mocks,
}))

describe('PwaUpdateBanner', () => {
  beforeEach(() => {
    mocks.refresh.mockClear()
  })

  it('is hidden until an update is available', () => {
    mocks.updateAvailable.value = false
    const wrapper = mount(PwaUpdateBanner)

    expect(wrapper.text()).not.toContain('Update available')
  })

  it('shows the banner and refreshes when clicked', async () => {
    mocks.updateAvailable.value = true
    const wrapper = mount(PwaUpdateBanner)

    expect(wrapper.text()).toContain('Update available')
    await wrapper.findAll('button').find(button => button.text() === 'Refresh')!.trigger('click')

    expect(mocks.refresh).toHaveBeenCalled()
  })

  it('dismisses without calling refresh', async () => {
    mocks.updateAvailable.value = true
    const wrapper = mount(PwaUpdateBanner)

    await wrapper.find('button[aria-label="Dismiss"]').trigger('click')

    expect(wrapper.text()).not.toContain('Update available')
    expect(mocks.refresh).not.toHaveBeenCalled()
  })
})
