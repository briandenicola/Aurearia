import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  applyAccountPreferences, getPreferenceScope, savePreference, setPreferenceScope,
} from '@/composables/usePreferenceSync'

const mockUpdateProfile = vi.fn()

vi.mock('@/api/client', () => ({
  updateProfile: (...args: unknown[]) => mockUpdateProfile(...args),
}))

const emptyPrefs = { theme: '', timezone: '', defaultView: '', defaultSort: '', trayFeltColor: '' }

describe('usePreferenceSync', () => {
  beforeEach(() => {
    localStorage.clear()
    mockUpdateProfile.mockReset().mockResolvedValue({ data: {} })
    document.documentElement.removeAttribute('data-theme')
  })

  it('defaults to account scope', () => {
    expect(getPreferenceScope()).toBe('account')
  })

  it('applies account preferences to this device', () => {
    applyAccountPreferences({ ...emptyPrefs, theme: 'capitoline', trayFeltColor: 'navy', timezone: 'America/Chicago' })

    expect(localStorage.getItem('theme')).toBe('capitoline')
    expect(localStorage.getItem('tray:feltColor')).toBe('navy')
    expect(localStorage.getItem('timezone')).toBe('America/Chicago')
    expect(document.documentElement.getAttribute('data-theme')).toBe('capitoline')
    expect(mockUpdateProfile).not.toHaveBeenCalled()
  })

  it('seeds the account from this device for preferences the account has not set', () => {
    localStorage.setItem('defaultSort', 'created_at_asc')
    localStorage.setItem('theme', 'louvre')

    applyAccountPreferences({ ...emptyPrefs, theme: 'byzantine' })

    expect(localStorage.getItem('theme')).toBe('byzantine')
    expect(mockUpdateProfile).toHaveBeenCalledWith({ preferences: { defaultSort: 'created_at_asc' } })
  })

  it('ignores account preferences and skips saving when the device opts out', async () => {
    localStorage.setItem('theme', 'light')
    await setPreferenceScope('device')

    applyAccountPreferences({ ...emptyPrefs, theme: 'byzantine' })
    savePreference('theme', 'dark')

    expect(localStorage.getItem('theme')).toBe('light')
    expect(mockUpdateProfile).not.toHaveBeenCalled()
  })

  it('saves a changed preference to the account when syncing', () => {
    savePreference('defaultView', 'grid')
    expect(mockUpdateProfile).toHaveBeenCalledWith({ preferences: { defaultView: 'grid' } })
  })

  it('uploads this device\'s settings when sync is turned back on', async () => {
    localStorage.setItem('preferences:scope', 'device')
    localStorage.setItem('theme', 'modern-greek')
    localStorage.setItem('tray:feltColor', 'green')

    await setPreferenceScope('account')

    expect(getPreferenceScope()).toBe('account')
    expect(mockUpdateProfile).toHaveBeenCalledWith({ preferences: { theme: 'modern-greek', trayFeltColor: 'green' } })
  })
})
