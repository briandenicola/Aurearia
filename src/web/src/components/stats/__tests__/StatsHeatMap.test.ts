import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import StatsHeatMap from '@/components/stats/StatsHeatMap.vue'

const mockGetDistribution = vi.fn()
const mockPush = vi.fn()

vi.mock('@/api/client', () => ({
  getDistribution: () => mockGetDistribution(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
}))

function exposedFetchDistribution(wrapper: ReturnType<typeof mount>): Promise<void> {
  return (wrapper.vm as unknown as { fetchDistribution: () => Promise<void> }).fetchDistribution()
}

describe('StatsHeatMap', () => {
  beforeEach(() => {
    mockGetDistribution.mockReset()
    mockPush.mockReset()
    mockGetDistribution.mockResolvedValue({
      data: {
        cells: [
          { era: 'ancient', category: 'Roman', count: 3 },
          { era: 'ancient', category: 'Greek', count: 0 },
          { era: 'medieval', category: 'Byzantine', count: 1 },
        ],
      },
    })
  })

  it('renders distribution cells from the stats API', async () => {
    const wrapper = mount(StatsHeatMap)

    await exposedFetchDistribution(wrapper)
    await flushPromises()

    expect(mockGetDistribution).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Collection Distribution')
    expect(wrapper.text()).toContain('Roman')
    expect(wrapper.text()).toContain('Greek')
    expect(wrapper.text()).toContain('ancient')
    expect(wrapper.find('[title="ancient / Roman: 3 coins"]').text()).toBe('3')
    expect(wrapper.find('[title="medieval / Byzantine: 1 coins"]').text()).toBe('1')
  })

  it('navigates to the category filter when a populated heatmap cell is clicked', async () => {
    const wrapper = mount(StatsHeatMap)

    await exposedFetchDistribution(wrapper)
    await flushPromises()
    await wrapper.find('[title="ancient / Roman: 3 coins"]').trigger('click')

    expect(mockPush).toHaveBeenCalledWith({ path: '/', query: { category: 'Roman' } })
  })

  it('does not navigate when an empty heatmap cell is clicked', async () => {
    const wrapper = mount(StatsHeatMap)

    await exposedFetchDistribution(wrapper)
    await flushPromises()
    await wrapper.find('[title="ancient / Greek: 0 coins"]').trigger('click')

    expect(mockPush).not.toHaveBeenCalled()
  })
})
