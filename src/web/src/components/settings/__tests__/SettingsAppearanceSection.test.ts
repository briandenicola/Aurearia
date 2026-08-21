/**
 * Tests for S1 (Appearance section renders Swipe Navigation row and emits on toggle),
 * S3 (toggle calls PUT /user/profile with exactly { pwaSwipeNavEnabled }),
 * S4 (request rejection reverts toggle and surfaces error -- no silent failure).
 *
 * Design authority: .squad/decisions/inbox/maximus-pwa-swipe-reliability-review.md §13
 */

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SettingsAppearanceSection from '@/components/settings/SettingsAppearanceSection.vue'
import type { Theme } from '@/types'
import type { FeltColor } from '@/composables/useTrayPreference'

const mockUpdateProfile = vi.fn()

const authUser = {
  pwaSwipeNavEnabled: false,
}

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: authUser }),
}))

vi.mock('@/api/client', () => ({
  updateProfile: (...args: unknown[]) => mockUpdateProfile(...args),
}))

const defaultProps = {
  theme: 'dark' as Theme,
  timezone: 'UTC',
  timezones: ['UTC', 'America/New_York'],
  defaultView: 'grid' as 'grid' | 'swipe',
  defaultSort: 'updated_at_desc',
  trayFeltColor: 'red' as FeltColor,
}

function mountSection() {
  return mount(SettingsAppearanceSection, { props: defaultProps })
}

function checkboxForRowContaining(wrapper: ReturnType<typeof mount>, text: string) {
  const checkbox = wrapper.findAll('input[type="checkbox"]').find((input) => {
    const row = input.element.closest('div')
    return row?.textContent?.includes(text) ?? false
  })
  expect(checkbox, `expected to find a checkbox in a row containing "${text}"`).toBeTruthy()
  return checkbox!
}

describe('S1: SettingsAppearanceSection renders Swipe Navigation row', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authUser.pwaSwipeNavEnabled = false
  })

  it('renders the Swipe Navigation row with the correct label', () => {
    const wrapper = mountSection()
    expect(wrapper.text()).toContain('Swipe Navigation')
  })

  it('renders the correct description for the toggle', () => {
    const wrapper = mountSection()
    expect(wrapper.text()).toContain('Applies to the installed app only')
  })

  it('renders the toggle unchecked when auth user has pwaSwipeNavEnabled=false', () => {
    authUser.pwaSwipeNavEnabled = false
    const wrapper = mountSection()
    const cb = checkboxForRowContaining(wrapper, 'Swipe Navigation')
    expect((cb.element as HTMLInputElement).checked).toBe(false)
  })

  it('persists the server-confirmed value to localStorage on success', async () => {
    // savePwaSwipeNav (the single save path) writes localStorage; this test proves
    // the component exercises that path and the value reaches persistent storage.
    mockUpdateProfile.mockResolvedValue({
      data: {
        pwaSwipeNavEnabled: true,
        id: 1, username: 'user', role: 'user', email: '', avatarPath: '',
        isPublic: false, bio: '', zipCode: '',
        numisBidsUsername: '', numisBidsConfigured: false,
        cngUsername: '', cngConfigured: false,
        parcelAppConfigured: false, pushoverEnabled: false,
        coinOfDayEnabled: true, coinOfDayIncludeWishlist: true,
        emperorTrackerEnabled: false, emperorTrackerShowUsurpers: false,
        emperorTrackerShowEmpresses: false, emperorTrackerShowOtherFigures: false,
      },
    })
    const wrapper = mountSection()
    await checkboxForRowContaining(wrapper, 'Swipe Navigation').setValue(true)
    await flushPromises()
    const stored = JSON.parse(localStorage.getItem('user') ?? '{}')
    expect(stored.pwaSwipeNavEnabled).toBe(true)
    wrapper.unmount()
  })
})

describe('S3: toggle issues exactly { pwaSwipeNavEnabled } to PUT /user/profile', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authUser.pwaSwipeNavEnabled = false
  })

  it('calls updateProfile with exactly { pwaSwipeNavEnabled: true } on enable', async () => {
    mockUpdateProfile.mockResolvedValue({
      data: {
        pwaSwipeNavEnabled: true,
        id: 1, username: 'user', role: 'user', email: '', avatarPath: '',
        isPublic: false, bio: '', zipCode: '',
        numisBidsUsername: '', numisBidsConfigured: false,
        cngUsername: '', cngConfigured: false,
        parcelAppConfigured: false, pushoverEnabled: false,
        coinOfDayEnabled: true, coinOfDayIncludeWishlist: true,
        emperorTrackerEnabled: false, emperorTrackerShowUsurpers: false,
        emperorTrackerShowEmpresses: false, emperorTrackerShowOtherFigures: false,
      },
    })
    const wrapper = mountSection()
    await checkboxForRowContaining(wrapper, 'Swipe Navigation').setValue(true)
    await flushPromises()
    expect(mockUpdateProfile).toHaveBeenCalledWith({ pwaSwipeNavEnabled: true })
    // The payload must contain ONLY pwaSwipeNavEnabled -- no other profile fields
    const callArg = mockUpdateProfile.mock.calls[0]?.[0] as Record<string, unknown>
    expect(Object.keys(callArg)).toEqual(['pwaSwipeNavEnabled'])
  })

  it('calls updateProfile with exactly { pwaSwipeNavEnabled: false } on disable', async () => {
    authUser.pwaSwipeNavEnabled = true
    mockUpdateProfile.mockResolvedValue({
      data: {
        pwaSwipeNavEnabled: false,
        id: 1, username: 'user', role: 'user', email: '', avatarPath: '',
        isPublic: false, bio: '', zipCode: '',
        numisBidsUsername: '', numisBidsConfigured: false,
        cngUsername: '', cngConfigured: false,
        parcelAppConfigured: false, pushoverEnabled: false,
        coinOfDayEnabled: true, coinOfDayIncludeWishlist: true,
        emperorTrackerEnabled: false, emperorTrackerShowUsurpers: false,
        emperorTrackerShowEmpresses: false, emperorTrackerShowOtherFigures: false,
      },
    })
    const wrapper = mountSection()
    await checkboxForRowContaining(wrapper, 'Swipe Navigation').setValue(false)
    await flushPromises()
    expect(mockUpdateProfile).toHaveBeenCalledWith({ pwaSwipeNavEnabled: false })
    const callArg = mockUpdateProfile.mock.calls[0]?.[0] as Record<string, unknown>
    expect(Object.keys(callArg)).toEqual(['pwaSwipeNavEnabled'])
  })

  it('updates auth.user.pwaSwipeNavEnabled on successful save', async () => {
    mockUpdateProfile.mockResolvedValue({
      data: {
        pwaSwipeNavEnabled: true,
        id: 1, username: 'user', role: 'user', email: '', avatarPath: '',
        isPublic: false, bio: '', zipCode: '',
        numisBidsUsername: '', numisBidsConfigured: false,
        cngUsername: '', cngConfigured: false,
        parcelAppConfigured: false, pushoverEnabled: false,
        coinOfDayEnabled: true, coinOfDayIncludeWishlist: true,
        emperorTrackerEnabled: false, emperorTrackerShowUsurpers: false,
        emperorTrackerShowEmpresses: false, emperorTrackerShowOtherFigures: false,
      },
    })
    const wrapper = mountSection()
    await checkboxForRowContaining(wrapper, 'Swipe Navigation').setValue(true)
    await flushPromises()
    expect(authUser.pwaSwipeNavEnabled).toBe(true)
    wrapper.unmount()
  })
})

describe('S4: rejected request reverts toggle and surfaces error', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authUser.pwaSwipeNavEnabled = false
  })

  it('reverts the toggle to previous state on API failure', async () => {
    mockUpdateProfile.mockRejectedValueOnce(new Error('network error'))
    const wrapper = mountSection()
    const cb = checkboxForRowContaining(wrapper, 'Swipe Navigation')

    await cb.setValue(true)
    await flushPromises()

    // The checkbox should revert to unchecked
    expect((cb.element as HTMLInputElement).checked).toBe(false)
    wrapper.unmount()
  })

  it('surfaces an error message on API failure -- no silent failure', async () => {
    mockUpdateProfile.mockRejectedValueOnce(new Error('network error'))
    const wrapper = mountSection()
    await checkboxForRowContaining(wrapper, 'Swipe Navigation').setValue(true)
    await flushPromises()
    expect(wrapper.text()).toContain('Failed to save swipe navigation setting')
    wrapper.unmount()
  })

  it('does not update auth.user on failure', async () => {
    mockUpdateProfile.mockRejectedValueOnce(new Error('network error'))
    const wrapper = mountSection()
    await checkboxForRowContaining(wrapper, 'Swipe Navigation').setValue(true)
    await flushPromises()
    expect(authUser.pwaSwipeNavEnabled).toBe(false)
    wrapper.unmount()
  })
})