import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminWishlistSearchAlertSchedule from '../AdminWishlistSearchAlertSchedule.vue'

const mocks = vi.hoisted(() => ({
  getAdminWishlistSearchAlertRuns: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

const settings = {
  WishlistSearchAlertsCheckEnabled: 'true',
  WishlistSearchAlertsCheckStartTime: '08:00',
}

describe('AdminWishlistSearchAlertSchedule', () => {
  beforeEach(() => {
    mocks.getAdminWishlistSearchAlertRuns.mockReset()
  })

  it('renders scheduled alert run history with owner and result counts', async () => {
    mocks.getAdminWishlistSearchAlertRuns.mockResolvedValue({
      data: {
        runs: [{
          id: 4,
          alertId: 2,
          alertName: 'Roman silver',
          userId: 7,
          userName: 'collector',
          triggerType: 'scheduled',
          status: 'completed',
          startedAt: '2026-09-15T08:00:00Z',
          completedAt: '2026-09-15T08:00:02Z',
          durationMs: 2000,
          resultCount: 5,
          newCount: 2,
          duplicateCount: 3,
          dismissedCount: 0,
          partialWarnings: [],
          errorMessage: '',
          rateLimitStatus: 'ok',
        }],
        total: 1,
        page: 1,
        limit: 5,
      },
    })

    const wrapper = mount(AdminWishlistSearchAlertSchedule, {
      props: { settings, settingsSaving: false },
    })
    await flushPromises()

    expect(mocks.getAdminWishlistSearchAlertRuns).toHaveBeenCalledWith(1, 5)
    expect(wrapper.text()).toContain('Wishlist Search Alert Run History')
    expect(wrapper.text()).toContain('Roman silver')
    expect(wrapper.text()).toContain('collector')
    expect(wrapper.text()).toContain('scheduled')
    expect(wrapper.find('.overflow-x-auto table').exists()).toBe(true)
  })

  it('shows an explicit error instead of an empty-history result when loading fails', async () => {
    mocks.getAdminWishlistSearchAlertRuns.mockRejectedValue(new Error('network error'))
    const wrapper = mount(AdminWishlistSearchAlertSchedule, {
      props: { settings, settingsSaving: false },
    })
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('Unable to load')
    expect(wrapper.text()).not.toContain('No wishlist search alert runs recorded yet.')
  })
})
