import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SetInsightsPage from '@/pages/SetInsightsPage.vue'

const mockCreateSetSnapshot = vi.fn()
const mockGetSet = vi.fn()
const mockGetSetAnalytics = vi.fn()
const mockGetSetTrends = vi.fn()
const mockPush = vi.fn()

vi.mock('@/api/client', () => ({
  createSetSnapshot: (...args: unknown[]) => mockCreateSetSnapshot(...args),
  getSet: (...args: unknown[]) => mockGetSet(...args),
  getSetAnalytics: (...args: unknown[]) => mockGetSetAnalytics(...args),
  getSetTrends: (...args: unknown[]) => mockGetSetTrends(...args),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { id: '7' } }),
  useRouter: () => ({ push: mockPush }),
}))

const setFixture = {
  id: 7,
  name: 'Imperial Portrait Types',
  color: '#c9a84c',
}

describe('SetInsightsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetSet.mockResolvedValue({ data: setFixture })
    mockGetSetAnalytics.mockResolvedValue({ data: { roiPercent: 8.4, acquisitionRatePerMonth: 0.9 } })
    mockGetSetTrends.mockResolvedValue({
      data: {
        snapshots: [{ snapshotDate: '2026-08-01T00:00:00Z', totalValue: 1234.56 }],
      },
    })
    mockCreateSetSnapshot.mockResolvedValue({})
  })

  it('loads analytics and value trend data', async () => {
    const wrapper = mount(SetInsightsPage)
    await flushPromises()

    expect(mockGetSet).toHaveBeenCalledWith(7)
    expect(mockGetSetAnalytics).toHaveBeenCalledWith(7)
    expect(mockGetSetTrends).toHaveBeenCalledWith(7, '1y')
    expect(wrapper.text()).toContain('Analytics')
    expect(wrapper.text()).toContain('8.4%')
    expect(wrapper.text()).toContain('0.9/mo')
  })

  it('navigates back to set detail', async () => {
    const wrapper = mount(SetInsightsPage)
    await flushPromises()

    await wrapper.find('button.btn.btn-ghost').trigger('click')
    expect(mockPush).toHaveBeenCalledWith({ name: 'set-detail', params: { id: 7 } })
  })

  it('captures a snapshot from the trend card action', async () => {
    const wrapper = mount(SetInsightsPage)
    await flushPromises()

    await wrapper.find('button.btn.btn-secondary').trigger('click')
    expect(mockCreateSetSnapshot).toHaveBeenCalledWith(7)
  })
})
