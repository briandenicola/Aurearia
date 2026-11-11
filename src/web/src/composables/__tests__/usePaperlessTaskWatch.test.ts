import { flushPromises } from '@vue/test-utils'
import { effectScope, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { PAPERLESS_WATCH_INTERVAL_MS, usePaperlessTaskWatch } from '@/composables/usePaperlessTaskWatch'

const mockResolve = vi.fn()
vi.mock('@/api/client', () => ({
  resolvePaperlessInvoiceTask: (...args: unknown[]) => mockResolve(...args),
  getApiErrorMessage: (err: { response?: { data?: { error?: string } } }) => err.response?.data?.error ?? '',
}))

function run(taskId: string, enabled = true) {
  const scope = effectScope()
  const onSettled = vi.fn()
  const watcher = scope.run(() => usePaperlessTaskWatch(ref(7), ref(taskId), ref(enabled), onSettled))!
  return { scope, onSettled, error: watcher.error }
}

describe('usePaperlessTaskWatch', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockResolve.mockReset()
  })
  afterEach(() => vi.useRealTimers())

  it('keeps checking a pending task and reloads the coin once it links', async () => {
    mockResolve
      .mockResolvedValueOnce({ data: { status: 'pending', taskId: 't-1' } })
      .mockResolvedValueOnce({ data: { status: 'linked', taskId: 't-1', documentId: 4950 } })
    const { scope, onSettled } = run('t-1')

    await vi.advanceTimersByTimeAsync(PAPERLESS_WATCH_INTERVAL_MS)
    expect(mockResolve).toHaveBeenCalledWith(7, 't-1')
    expect(onSettled).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(PAPERLESS_WATCH_INTERVAL_MS)
    await flushPromises()
    expect(onSettled).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(PAPERLESS_WATCH_INTERVAL_MS * 3)
    expect(mockResolve).toHaveBeenCalledTimes(2)
    scope.stop()
  })

  it('reports a failed task and reloads the coin', async () => {
    mockResolve.mockRejectedValue({ response: { data: { error: 'paperless-ngx could not consume the document: duplicate' } } })
    const { scope, onSettled, error } = run('t-2')
    await vi.advanceTimersByTimeAsync(PAPERLESS_WATCH_INTERVAL_MS)
    await flushPromises()
    expect(error.value).toContain('duplicate')
    expect(onSettled).toHaveBeenCalledTimes(1)
    scope.stop()
  })

  it('does nothing without a task, when disabled, and after the page is left', async () => {
    run('').scope.stop()
    run('t-3', false).scope.stop()
    const { scope } = run('t-4')
    scope.stop()
    await vi.advanceTimersByTimeAsync(PAPERLESS_WATCH_INTERVAL_MS * 3)
    expect(mockResolve).not.toHaveBeenCalled()
  })
})
