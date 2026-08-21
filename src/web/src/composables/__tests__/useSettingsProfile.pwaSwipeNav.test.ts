import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSettingsProfile } from '../useSettingsProfile'
import { updateProfile } from '@/api/client'
import type { User } from '@/types'

// QA contract tests for the pwaSwipeNavEnabled preference in useSettingsProfile.
// Design authority: .squad/decisions/inbox/maximus-pwa-swipe-setting-review.md §3 §4
//
// Verified contracts:
// - DEFAULT-OFF: a user with no stored preference defaults to false (fail closed).
// - Save payload always includes pwaSwipeNavEnabled (pointer-omission safe).
// - Server-confirmed value syncs into auth store AND localStorage on success.
// - On API failure, auth store AND localStorage are untouched (confirmed, not optimistic).

vi.mock('@/api/client', () => ({
  changePassword: vi.fn(),
  uploadAvatar: vi.fn(),
  deleteAvatar: vi.fn(),
  updateProfile: vi.fn(),
  validateNumisBidsCredentials: vi.fn(),
  testPushover: vi.fn(),
  onTokenRefreshed: vi.fn(),
  login: vi.fn(),
  register: vi.fn(),
}))

function seedStoredUser(overrides: Partial<User> = {}) {
  const user: Partial<User> = {
    id: 1,
    username: 'brian',
    email: 'brian@example.com',
    role: 'user',
    ...overrides,
  }
  localStorage.setItem('user', JSON.stringify(user))
  localStorage.setItem('token', 'test-token')
}

function makeProfileResponse(pwaSwipeNavEnabled: boolean) {
  return {
    data: {
      id: 1, username: 'brian', role: 'user', email: 'brian@example.com',
      avatarPath: '', isPublic: false, bio: '', zipCode: '',
      numisBidsUsername: '', numisBidsConfigured: false,
      cngUsername: '', cngConfigured: false,
      parcelAppConfigured: false, pushoverEnabled: false,
      coinOfDayEnabled: true, coinOfDayIncludeWishlist: true,
      emperorTrackerEnabled: false, emperorTrackerShowUsurpers: false,
      emperorTrackerShowEmpresses: false, emperorTrackerShowOtherFigures: false,
      pwaSwipeNavEnabled,
    },
  }
}

describe('useSettingsProfile -- pwaSwipeNavEnabled (PWA swipe navigation preference)', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('defaults to false when stored user has no pwaSwipeNavEnabled field (legacy/new account)', () => {
    seedStoredUser() // no pwaSwipeNavEnabled key at all
    setActivePinia(createPinia())

    const { pwaSwipeNavEnabled } = useSettingsProfile()

    expect(pwaSwipeNavEnabled.value).toBe(false)
  })

  it('honors an explicit stored false value (not coerced to anything else)', () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())

    const { pwaSwipeNavEnabled } = useSettingsProfile()

    expect(pwaSwipeNavEnabled.value).toBe(false)
  })

  it('initializes to true when stored user has pwaSwipeNavEnabled: true', () => {
    seedStoredUser({ pwaSwipeNavEnabled: true })
    setActivePinia(createPinia())

    const { pwaSwipeNavEnabled } = useSettingsProfile()

    expect(pwaSwipeNavEnabled.value).toBe(true)
  })

  it('Account save (handleSaveProfile) does NOT include pwaSwipeNavEnabled -- Appearance tab owns it via savePwaSwipeNav', async () => {
    // D8: pwaSwipeNavEnabled was removed from the Account save payload so Account saves
    // cannot clobber the Appearance-managed value.
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue(
      makeProfileResponse(false) as Awaited<ReturnType<typeof updateProfile>>,
    )

    const settings = useSettingsProfile()
    await settings.handleSaveProfile()

    expect(updateProfile).toHaveBeenCalledTimes(1)
    const payload = vi.mocked(updateProfile).mock.calls[0]?.[0] as Record<string, unknown>
    expect(payload).not.toHaveProperty('pwaSwipeNavEnabled')
  })

  it('savePwaSwipeNav sends ONLY pwaSwipeNavEnabled -- narrow payload, not the full profile', async () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue(
      makeProfileResponse(true) as Awaited<ReturnType<typeof updateProfile>>,
    )

    const settings = useSettingsProfile()
    await settings.savePwaSwipeNav(true)

    expect(updateProfile).toHaveBeenCalledTimes(1)
    const payload = vi.mocked(updateProfile).mock.calls[0]?.[0] as Record<string, unknown>
    expect(Object.keys(payload)).toEqual(['pwaSwipeNavEnabled'])
    expect(payload.pwaSwipeNavEnabled).toBe(true)
  })

  it('savePwaSwipeNav syncs the server-confirmed value into the auth store user on success', async () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue(
      makeProfileResponse(true) as Awaited<ReturnType<typeof updateProfile>>,
    )

    const settings = useSettingsProfile()
    await settings.savePwaSwipeNav(true)

    const { useAuthStore } = await import('@/stores/auth')
    const auth = useAuthStore()
    expect(auth.user?.pwaSwipeNavEnabled).toBe(true)
  })

  it('savePwaSwipeNav persists the server-confirmed value to localStorage on success', async () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue(
      makeProfileResponse(true) as Awaited<ReturnType<typeof updateProfile>>,
    )

    const settings = useSettingsProfile()
    await settings.savePwaSwipeNav(true)

    const stored = JSON.parse(localStorage.getItem('user') ?? '{}')
    expect(stored.pwaSwipeNavEnabled).toBe(true)
  })

  it('does NOT update auth store or localStorage when save fails (confirmed, not optimistic)', async () => {
    // Exercises the actual UI-invoked path: savePwaSwipeNav throws on failure,
    // rolls back the local ref, and leaves auth store + localStorage untouched.
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockRejectedValue(new Error('network error'))

    const settings = useSettingsProfile()
    await expect(settings.savePwaSwipeNav(true)).rejects.toThrow()

    const { useAuthStore } = await import('@/stores/auth')
    const auth = useAuthStore()
    // Auth store must stay at the pre-save value.
    expect(auth.user?.pwaSwipeNavEnabled).not.toBe(true)

    // localStorage must not have been updated.
    const stored = JSON.parse(localStorage.getItem('user') ?? '{}')
    expect(stored.pwaSwipeNavEnabled).not.toBe(true)

    // Local ref must be rolled back (optimistic update reverted).
    expect(settings.pwaSwipeNavEnabled.value).toBe(false)
  })
})
