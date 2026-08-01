import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EmperorStatsPage from '@/pages/EmperorStatsPage.vue'
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
    owned: 31,
    total: 87,
    percentage: 35.6,
    dynasties: [
      { dynasty: 'Julio-Claudian', owned: 8, total: 12, figures: [] },
      { dynasty: 'Flavian', owned: 3, total: 3, figures: [] },
    ],
  },
  suggestions: [
    { id: 1, name: 'Tiberius', aliases: [], role: 'emperor', region: 'west', dynasty: 'Julio-Claudian', reignStart: 14, reignEnd: 37, sortOrder: 1, rarityTier: 'common' },
  ],
  usurpers: {
    roles: ['usurper'],
    owned: 2,
    total: 10,
    percentage: 20,
    dynasties: [],
  },
}

describe('EmperorStatsPage', () => {
  beforeEach(() => {
    mockGetProgress.mockReset()
  })

  it('renders core completion metrics and category rows', async () => {
    mockGetProgress.mockResolvedValue({ data: fullResult })
    const wrapper = mount(EmperorStatsPage, {
      global: {
        stubs: { RouterLink: { props: ['to'], template: '<a :to="to"><slot /></a>' } },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('31 of 87')
    expect(wrapper.text()).toContain('Remaining Augustuses')
    expect(wrapper.text()).toContain('56')
    expect(wrapper.text()).toContain('Completed Dynasties')
    expect(wrapper.text()).toContain('1 of 2')
    expect(wrapper.text()).toContain('Pursuit Suggestions')
    expect(wrapper.text()).toContain('Category Coverage')
    expect(wrapper.text()).toContain('Usurpers')
  })

  it('shows API errors', async () => {
    mockGetProgress.mockRejectedValue({ message: 'network down' })
    const wrapper = mount(EmperorStatsPage, {
      global: {
        stubs: { RouterLink: { props: ['to'], template: '<a :to="to"><slot /></a>' } },
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('network down')
  })
})
