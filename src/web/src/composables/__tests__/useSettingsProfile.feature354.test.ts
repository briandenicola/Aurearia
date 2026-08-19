import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSettingsProfile } from '../useSettingsProfile'
import { updateProfile } from '@/api/client'
import type { User } from '@/types'

// Independent QA regression coverage for spec 354 (D-Settings toggle,
// FR-021..024), owned by Brutus (Tester/QA). Dedicated new file — no
// existing test previously covered `coinOfDayIncludeWishlist` at the
// composable level, so this is purely additive (not a duplicate of any
// Aurelia-authored file).
//
// Frozen contract under test:
//  - Brian approved DEFAULT-ON: a user with no stored preference (new
//    account, or one that predates this feature) must default to
//    `coinOfDayIncludeWishlist = true` (`?? true` fallback), never to
//    opt-out-by-default.
//  - Brian approved "no Settings frequency hint" — this toggle is a plain
//    on/off switch with no extra frequency copy/control; this suite does
//    not assert on any additional frequency UI existing (regression
//    tripwire if one were ever added without a spec update).
//  - The toggle round-trips through the same `updateProfile` PATCH payload
//    contract as every other settings field (no new endpoint).

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

describe('useSettingsProfile — Coin of the Day wishlist-inclusion toggle (spec 354)', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('defaults coinOfDayIncludeWishlist to true when the stored user has no explicit value (new/legacy account)', () => {
    seedStoredUser() // no coinOfDayIncludeWishlist field at all
    setActivePinia(createPinia())

    const { coinOfDayIncludeWishlist } = useSettingsProfile()

    expect(coinOfDayIncludeWishlist.value).toBe(true)
  })

  it('honors an explicit stored false value rather than forcing the default back on', () => {
    seedStoredUser({ coinOfDayIncludeWishlist: false })
    setActivePinia(createPinia())

    const { coinOfDayIncludeWishlist } = useSettingsProfile()

    expect(coinOfDayIncludeWishlist.value).toBe(false)
  })

  it('sends coinOfDayIncludeWishlist in the save payload alongside coinOfDayEnabled', async () => {
    seedStoredUser({ coinOfDayEnabled: true, coinOfDayIncludeWishlist: true })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue({
      data: {
        email: 'brian@example.com', bio: '', zipCode: '', isPublic: false,
        numisBidsUsername: '', numisBidsConfigured: false, cngUsername: '', cngConfigured: false,
        parcelAppConfigured: false, pushoverEnabled: false,
        coinOfDayEnabled: true, coinOfDayIncludeWishlist: false,
        emperorTrackerEnabled: false, emperorTrackerShowUsurpers: false,
        emperorTrackerShowEmpresses: false, emperorTrackerShowOtherFigures: false,
      },
    } as Awaited<ReturnType<typeof updateProfile>>)

    const settings = useSettingsProfile()
    settings.coinOfDayIncludeWishlist.value = false
    await settings.handleSaveProfile()

    expect(updateProfile).toHaveBeenCalledTimes(1)
    const payload = vi.mocked(updateProfile).mock.calls[0]?.[0] as Record<string, unknown>
    expect(payload).toMatchObject({ coinOfDayEnabled: true, coinOfDayIncludeWishlist: false })
  })

  it('persists the server-confirmed value back onto the auth store user after save (survives reload)', async () => {
    seedStoredUser({ coinOfDayIncludeWishlist: true })
    setActivePinia(createPinia())
    vi.mocked(updateProfile).mockResolvedValue({
      data: {
        email: 'brian@example.com', bio: '', zipCode: '', isPublic: false,
        numisBidsUsername: '', numisBidsConfigured: false, cngUsername: '', cngConfigured: false,
        parcelAppConfigured: false, pushoverEnabled: false,
        coinOfDayEnabled: true, coinOfDayIncludeWishlist: false,
        emperorTrackerEnabled: false, emperorTrackerShowUsurpers: false,
        emperorTrackerShowEmpresses: false, emperorTrackerShowOtherFigures: false,
      },
    } as Awaited<ReturnType<typeof updateProfile>>)

    const settings = useSettingsProfile()
    settings.coinOfDayIncludeWishlist.value = false
    await settings.handleSaveProfile()

    const stored = JSON.parse(localStorage.getItem('user') || '{}')
    expect(stored.coinOfDayIncludeWishlist).toBe(false)
  })
})
