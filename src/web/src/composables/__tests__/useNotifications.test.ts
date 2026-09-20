import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { useNotifications } from '../useNotifications'

const getCount = vi.hoisted(() => vi.fn())
vi.mock('@/api/client', () => ({ getUnreadNotificationCount: getCount }))

beforeEach(() => {
  useNotifications().stopPolling()
  vi.useFakeTimers()
  getCount.mockReset().mockResolvedValue({ data: { count: 2 } })
})
afterEach(() => { useNotifications().stopPolling(); vi.useRealTimers() })

it('clears counts and invalidates old account responses when polling restarts', async () => {
  let resolve!: (value: { data: { count: number } }) => void
  getCount.mockReturnValueOnce(new Promise(done => { resolve = done }))
  const state = useNotifications()
  state.startPolling()
  state.unreadCount.value = 9
  state.stopPolling()
  expect(state.unreadCount.value).toBe(0)
  state.startPolling()
  await Promise.resolve()
  resolve({ data: { count: 99 } })
  await Promise.resolve()
  expect(state.unreadCount.value).toBe(2)
  expect(vi.getTimerCount()).toBe(1)
})

it('does not allow an older overlapping refresh to replace the latest count', async () => {
  let resolve!: (value: { data: { count: number } }) => void
  getCount.mockReturnValueOnce(new Promise(done => { resolve = done }))
  const state = useNotifications()
  state.startPolling()
  await state.refresh()
  resolve({ data: { count: 99 } })
  await Promise.resolve()
  expect(state.unreadCount.value).toBe(2)
})

it('keeps one timer across repeated starts and stops all polling on logout', async () => {
  const state = useNotifications()
  state.startPolling()
  state.startPolling()
  expect(vi.getTimerCount()).toBe(1)
  await vi.advanceTimersByTimeAsync(60_000)
  expect(getCount).toHaveBeenCalledTimes(2)
  state.stopPolling()
  await vi.advanceTimersByTimeAsync(120_000)
  expect(getCount).toHaveBeenCalledTimes(2)
  expect(vi.getTimerCount()).toBe(0)
})

it('exposes refresh errors, preserves the last count, and clears the error on recovery', async () => {
  const state = useNotifications()
  state.startPolling()
  await Promise.resolve()
  getCount.mockRejectedValueOnce(new Error('offline'))
  await state.refresh()
  expect(state.unreadCount.value).toBe(2)
  expect(state.error.value).toContain('Unable to refresh')
  await state.refresh()
  expect(state.error.value).toBe('')
  state.stopPolling()
  const calls = getCount.mock.calls.length
  await state.refresh()
  expect(getCount).toHaveBeenCalledTimes(calls)
})

it('ignores prior-account failures after the new account succeeds', async () => {
  let reject!: (reason: Error) => void
  getCount.mockReturnValueOnce(new Promise((_, no) => { reject = no }))
  const state = useNotifications()
  state.startPolling()
  state.stopPolling()
  state.startPolling()
  await Promise.resolve()
  reject(new Error('old account failed'))
  await Promise.resolve()
  expect(state.unreadCount.value).toBe(2)
  expect(state.error.value).toBe('')
})
