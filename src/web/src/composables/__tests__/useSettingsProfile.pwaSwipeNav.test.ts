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

  it('sends pwaSwipeNavEnabled in the updateProfile payload', async () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue(
      makeProfileResponse(true) as Awaited<ReturnType<typeof updateProfile>>,
    )

    const settings = useSettingsProfile()
    settings.pwaSwipeNavEnabled.value = true
    await settings.handleSaveProfile()

    expect(updateProfile).toHaveBeenCalledTimes(1)
    const payload = vi.mocked(updateProfile).mock.calls[0]?.[0] as Record<string, unknown>
    expect(payload).toMatchObject({ pwaSwipeNavEnabled: true })
  })

  it('syncs the server-confirmed value into the auth store user on success', async () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue(
      makeProfileResponse(true) as Awaited<ReturnType<typeof updateProfile>>,
    )

    const settings = useSettingsProfile()
    settings.pwaSwipeNavEnabled.value = true
    await settings.handleSaveProfile()

    const { useAuthStore } = await import('@/stores/auth')
    const auth = useAuthStore()
    expect(auth.user?.pwaSwipeNavEnabled).toBe(true)
  })

  it('persists the server-confirmed value to localStorage on success', async () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue(
      makeProfileResponse(true) as Awaited<ReturnType<typeof updateProfile>>,
    )

    const settings = useSettingsProfile()
    settings.pwaSwipeNavEnabled.value = true
    await settings.handleSaveProfile()

    const stored = JSON.parse(localStorage.getItem('user') ?? '{}')
    expect(stored.pwaSwipeNavEnabled).toBe(true)
  })

  it('does NOT update auth store or localStorage when save fails (confirmed, not optimistic)', async () => {
    seedStoredUser({ pwaSwipeNavEnabled: false })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockRejectedValue(new Error('network error'))

    const settings = useSettingsProfile()
    settings.pwaSwipeNavEnabled.value = true
    await settings.handleSaveProfile()

    const { useAuthStore } = await import('@/stores/auth')
    const auth = useAuthStore()
    // Auth store must stay at the pre-save value.
    expect(auth.user?.pwaSwipeNavEnabled).not.toBe(true)

    // localStorage must not have been updated.
    const stored = JSON.parse(localStorage.getItem('user') ?? '{}')
    expect(stored.pwaSwipeNavEnabled).not.toBe(true)

    expect(settings.profileError.value).toBe(true)
  })
})
