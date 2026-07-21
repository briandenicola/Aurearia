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

  it('checks for updates on registration and every hour', () => {
    const registration = { update: vi.fn().mockResolvedValue(undefined) } as unknown as ServiceWorkerRegistration
    capturedOptions.onRegisteredSW?.('sw.js', registration)

    expect(registration.update).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(60 * 60 * 1000)
    expect(registration.update).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(60 * 60 * 1000)
    expect(registration.update).toHaveBeenCalledTimes(3)
  })

  it('checks for updates when the app returns to the foreground', () => {
    const registration = { update: vi.fn().mockResolvedValue(undefined) } as unknown as ServiceWorkerRegistration
    const focusSpy = vi.spyOn(window, 'addEventListener')
    const visibilityListenerSpy = vi.spyOn(document, 'addEventListener')
    const visibilitySpy = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    capturedOptions.onRegisteredSW?.('sw.js', registration)

    expect(registration.update).toHaveBeenCalledTimes(1)
    const focusHandler = focusSpy.mock.calls.find(([event]) => event === 'focus')?.[1] as (() => void) | undefined
    const visibilityHandler = visibilityListenerSpy.mock.calls.find(([event]) => event === 'visibilitychange')?.[1] as (() => void) | undefined

    vi.advanceTimersByTime(5 * 60 * 1000 + 1)
    focusHandler?.()
    expect(registration.update).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(5 * 60 * 1000 + 1)
    visibilityHandler?.()
    expect(registration.update).toHaveBeenCalledTimes(3)

    focusSpy.mockRestore()
    visibilityListenerSpy.mockRestore()
    visibilitySpy.mockRestore()
  })
})
