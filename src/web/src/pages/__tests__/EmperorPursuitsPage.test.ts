import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EmperorPursuitsPage from '@/pages/EmperorPursuitsPage.vue'
import type { EmperorTrackerResult } from '@/types'

const mockGetProgress = vi.fn()

vi.mock('@/api/client', () => ({
  getEmperorTrackerProgress: () => mockGetProgress(),
  getApiErrorMessage: (error: unknown) => {
    const maybeError = error as { message?: string }
    return maybeError.message ?? ''
  },
}))

const fullResult: EmperorTrackerResult = {
  emperor: {
    roles: ['emperor'],
    owned: 1,
    total: 2,
    percentage: 50,
    dynasties: [],
  },
  suggestions: [
    { id: 2, name: 'Tiberius', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: 14, reignEnd: 37, sortOrder: 2, rarityTier: 'common' },
  ],
}

describe('EmperorPursuitsPage', () => {
  beforeEach(() => {
    mockGetProgress.mockReset()
  })

  it('renders suggestion rows and back link', async () => {
    mockGetProgress.mockResolvedValue({ data: fullResult })
    const wrapper = mount(EmperorPursuitsPage, {
      global: {
        stubs: { RouterLink: { props: ['to'], template: '<a :to="to"><slot /></a>' } },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('What to Pursue Next')
    expect(wrapper.text()).toContain('Tiberius')
    expect(wrapper.find('a[aria-label="Back to Emperors"]').attributes('to')).toBe('/sets/emperors')
  })

  it('opens the agent with a search prompt for a suggestion', async () => {
    const dispatchSpy = vi.spyOn(window, 'dispatchEvent')
    mockGetProgress.mockResolvedValue({ data: fullResult })
    const wrapper = mount(EmperorPursuitsPage, {
      global: {
        stubs: { RouterLink: { props: ['to'], template: '<a :to="to"><slot /></a>' } },
      },
    })
    await flushPromises()

    await wrapper.find('button[aria-label="Ask the agent to search for Tiberius coins"]').trigger('click')

    expect(dispatchSpy).toHaveBeenCalledWith(expect.objectContaining({
      type: 'open-agent-chat',
      detail: expect.objectContaining({
        prompt: expect.stringContaining('Look for available Tiberius coins'),
      }),
    }))
    dispatchSpy.mockRestore()
  })
})
