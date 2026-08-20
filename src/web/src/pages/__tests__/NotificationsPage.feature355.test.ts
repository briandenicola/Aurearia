import { flushPromises, mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NotificationsPage from '../NotificationsPage.vue'
import type { Notification } from '@/types'

// Independent QA regression coverage for Feature 355 (Wishlist Purchase
// Reminders) -- notification deep-link contract.
// Owned by Brutus (Tester/QA).
//
// Frozen contract (spec.md FR-017, D10, US5 AC4):
//   - A notification with type=purchase_reminder and referenceUrl=/coin/{coinId}
//     must navigate to /coin/{coinId} when clicked.
//   - The existing generic referenceUrl handler covers this with no new code.
//   - markNotificationRead must be called on click.
//
// These tests use the EXISTING NotificationsPage.vue and should PASS today
// because D10 confirms the generic referenceUrl path already handles the
// routing. If they fail, the existing generic handler is broken (regression).

const mocks = vi.hoisted(() => ({
  getNotifications: vi.fn(),
  markNotificationRead: vi.fn(),
  markAllNotificationsRead: vi.fn(),
  deleteNotification: vi.fn(),
  getUnreadNotificationCount: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

function purchaseReminderNotification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: 100,
    userId: 7,
    type: 'purchase_reminder',
    title: 'Purchase Reminder',
    message: 'Remember to buy: Trajan Denarius',
    referenceId: 42,
    referenceUrl: '/coin/99',
    isRead: false,
    createdAt: '2026-09-15T08:00:00Z',
    ...overrides,
  }
}

async function buildRouter() {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/notifications', name: 'notifications', component: NotificationsPage },
      { path: '/coin/:id', name: 'coin', component: { template: '<div />' } },
    ],
  })
  router.push('/notifications')
  await router.isReady()
  return router
}

async function mountWith(n: Notification) {
  mocks.getNotifications.mockResolvedValue({ data: { notifications: [n], total: 1 } })
  mocks.markNotificationRead.mockResolvedValue({})
  mocks.getUnreadNotificationCount.mockResolvedValue({ data: { count: 0 } })
  const router = await buildRouter()
  const pushSpy = vi.spyOn(router, 'push')
  const wrapper = mount(NotificationsPage, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router, pushSpy }
}

describe('NotificationsPage — purchase_reminder deep-link contract (Feature 355)', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset())
  })

  it('routes purchase_reminder notification to /coin/{coinId} via referenceUrl on click', async () => {
    // FR-017 / D10: generic referenceUrl path handles routing -- no dedicated case needed.
    const { wrapper, pushSpy } = await mountWith(
      purchaseReminderNotification({ referenceUrl: '/coin/99' }),
    )

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith('/coin/99')
  })

  it('marks the purchase_reminder notification as read on click', async () => {
    const n = purchaseReminderNotification({ id: 100, referenceUrl: '/coin/99' })
    const { wrapper } = await mountWith(n)

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    expect(mocks.markNotificationRead).toHaveBeenCalledWith(100)
  })

  it('shows purchase_reminder notification with its title and message', async () => {
    const { wrapper } = await mountWith(
      purchaseReminderNotification({
        title: 'Purchase Reminder',
        message: 'Remember to buy: Trajan Denarius',
      }),
    )

    expect(wrapper.text()).toContain('Purchase Reminder')
    expect(wrapper.text()).toContain('Remember to buy: Trajan Denarius')
  })

  it('routes to different coin IDs correctly for multiple reminders', async () => {
    // FR-017: each purchase_reminder links to its own /coin/{id}.
    mocks.getNotifications.mockResolvedValue({
      data: {
        notifications: [
          purchaseReminderNotification({ id: 1, referenceUrl: '/coin/10' }),
          purchaseReminderNotification({ id: 2, referenceUrl: '/coin/20' }),
        ],
        total: 2,
      },
    })
    mocks.markNotificationRead.mockResolvedValue({})
    mocks.getUnreadNotificationCount.mockResolvedValue({ data: { count: 0 } })
    const router = await buildRouter()
    const pushSpy = vi.spyOn(router, 'push')
    const wrapper = mount(NotificationsPage, { global: { plugins: [router] } })
    await flushPromises()

    const cards = wrapper.findAll('.card')
    expect(cards).toHaveLength(2)

    await cards[0].trigger('click')
    await flushPromises()
    expect(pushSpy).toHaveBeenLastCalledWith('/coin/10')

    await cards[1].trigger('click')
    await flushPromises()
    expect(pushSpy).toHaveBeenLastCalledWith('/coin/20')
  })

  it('treats a purchase_reminder without referenceUrl as non-navigable (graceful no-op)', async () => {
    // Defensive: if a notification lacks referenceUrl (should not happen per FR-007),
    // the page must not throw -- it falls through to the wishlist/null branch.
    const { wrapper, pushSpy } = await mountWith(
      purchaseReminderNotification({ referenceUrl: undefined }),
    )

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    // No navigation to /coin/... should occur.
    const coinPushes = pushSpy.mock.calls.filter(([arg]) =>
      typeof arg === 'string' && arg.startsWith('/coin/'),
    )
    expect(coinPushes).toHaveLength(0)
  })
})
