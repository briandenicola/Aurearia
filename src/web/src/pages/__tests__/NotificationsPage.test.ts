import { flushPromises, mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NotificationsPage from '../NotificationsPage.vue'
import type { Notification } from '@/types'

const mocks = vi.hoisted(() => ({
  getNotifications: vi.fn(),
  markNotificationRead: vi.fn(),
  markAllNotificationsRead: vi.fn(),
  deleteNotification: vi.fn(),
  getUnreadNotificationCount: vi.fn(),
}))

vi.mock('@/api/client', () => mocks)

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: 1,
    userId: 3,
    type: 'wishlist_availability_run',
    title: 'Availability run',
    message: 'A run completed',
    referenceId: 17,
    referenceUrl: undefined,
    isRead: false,
    createdAt: '2026-08-01T10:00:00Z',
    ...overrides,
  }
}

async function buildRouter() {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/notifications', name: 'notifications', component: NotificationsPage },
      { path: '/admin/availability-cycles/:id', name: 'admin-availability-cycle', component: { template: '<div />' } },
      { path: '/wishlist/availability-runs/:id', name: 'wishlist-availability-run', component: { template: '<div />' } },
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

describe('NotificationsPage wishlist_availability_run routing', () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset())
  })

  it('routes admin failure notifications to the admin cycle detail page via referenceUrl', async () => {
    const { wrapper, pushSpy } = await mountWith(
      notification({ referenceId: 42, referenceUrl: '/admin/availability-cycles/42' }),
    )

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith('/admin/availability-cycles/42')
  })

  it('routes owner terminal notifications to the owner run detail page via referenceUrl', async () => {
    const { wrapper, pushSpy } = await mountWith(
      notification({ referenceId: 17, referenceUrl: '/wishlist/availability-runs/17' }),
    )

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith('/wishlist/availability-runs/17')
  })

  it('falls back to the owner run route using referenceId when referenceUrl is absent', async () => {
    const { wrapper, pushSpy } = await mountWith(
      notification({ referenceId: 17, referenceUrl: undefined }),
    )

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith('/wishlist/availability-runs/17')
  })

  it('ignores an unsafe absolute referenceUrl and falls back to the local route', async () => {
    const { wrapper, pushSpy } = await mountWith(
      notification({ referenceId: 17, referenceUrl: 'https://evil.example.com/phish' }),
    )

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith('/wishlist/availability-runs/17')
  })

  it('preserves existing routing for unrelated notification types', async () => {
    const { wrapper, pushSpy } = await mountWith(
      notification({ type: 'wishlist_unavailable', referenceId: 99, referenceUrl: undefined }),
    )

    await wrapper.find('.card').trigger('click')
    await flushPromises()

    expect(pushSpy).toHaveBeenCalledWith('/coin/99')
  })
})
