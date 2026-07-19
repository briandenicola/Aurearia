import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'

type RegisterSWOptions = {
  onNeedRefresh?: () => void
  onOfflineReady?: () => void
  onRegisteredSW?: (swScriptUrl: string, registration: ServiceWorkerRegistration | undefined) => void
}

const mocks = vi.hoisted(() => ({
  mockApplyUpdate: vi.fn().mockResolvedValue(undefined),
  capturedOptions: {} as RegisterSWOptions,
}))

vi.mock('virtual:pwa-register', () => ({
  registerSW: (options: RegisterSWOptions) => {
    mocks.capturedOptions = options
    return mocks.mockApplyUpdate
  },
}))

// Imported after the mock so the module's top-level registerSW() call is captured above.
import { usePwaUpdate } from '@/composables/usePwaUpdate'

const { mockApplyUpdate, capturedOptions } = mocks

describe('usePwaUpdate', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('flips updateAvailable when the service worker reports a new version', () => {
    const { updateAvailable } = usePwaUpdate()
    expect(updateAvailable.value).toBe(false)

    capturedOptions.onNeedRefresh?.()

    expect(updateAvailable.value).toBe(true)
  })

  it('refresh() applies the update and reloads the page', () => {
    const { refresh } = usePwaUpdate()

    refresh()

    expect(mockApplyUpdate).toHaveBeenCalledWith(true)
  })

  it('polls the registration for updates every hour', () => {
    const registration = { update: vi.fn() } as unknown as ServiceWorkerRegistration
    capturedOptions.onRegisteredSW?.('sw.js', registration)

    expect(registration.update).not.toHaveBeenCalled()

    vi.advanceTimersByTime(60 * 60 * 1000)
    expect(registration.update).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(60 * 60 * 1000)
    expect(registration.update).toHaveBeenCalledTimes(2)
  })
})
