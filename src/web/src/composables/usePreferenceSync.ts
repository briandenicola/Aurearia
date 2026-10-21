import { updateProfile } from '@/api/client'
import type { UserPreferences } from '@/types'

/**
 * Display preferences live in localStorage so they apply instantly (including
 * before login). When this device's scope is 'account' (the default), they are
 * also saved to the user's account and applied on every device they sign in on.
 * 'device' keeps this browser's choices to itself. The scope is per-device.
 */
export type PreferenceScope = 'account' | 'device'
export type PreferenceKey = keyof UserPreferences

const SCOPE_KEY = 'preferences:scope'

const LOCAL_KEYS: Record<PreferenceKey, string> = {
  theme: 'theme',
  timezone: 'timezone',
  defaultView: 'defaultView',
  defaultSort: 'defaultSort',
  trayFeltColor: 'tray:feltColor',
}

const PREFERENCE_KEYS = Object.keys(LOCAL_KEYS) as PreferenceKey[]

export function getPreferenceScope(): PreferenceScope {
  return localStorage.getItem(SCOPE_KEY) === 'device' ? 'device' : 'account'
}

function readLocalPreferences(): Partial<UserPreferences> {
  const prefs: Partial<UserPreferences> = {}
  for (const key of PREFERENCE_KEYS) {
    const value = localStorage.getItem(LOCAL_KEYS[key])
    if (value) prefs[key] = value
  }
  return prefs
}

async function pushPreferences(prefs: Partial<UserPreferences>) {
  if (Object.keys(prefs).length === 0) return
  await updateProfile({ preferences: prefs })
}

/**
 * Applies the account's saved preferences to this device. Any preference the
 * account has never stored is seeded from this device's current value, so the
 * first device a user signs in on after upgrading becomes the source of truth.
 */
export function applyAccountPreferences(prefs: Partial<UserPreferences> | undefined) {
  if (!prefs || getPreferenceScope() !== 'account') return

  const seed: Partial<UserPreferences> = {}
  for (const key of PREFERENCE_KEYS) {
    const remote = prefs[key]
    if (remote) {
      localStorage.setItem(LOCAL_KEYS[key], remote)
    } else {
      const local = localStorage.getItem(LOCAL_KEYS[key])
      if (local) seed[key] = local
    }
  }
  if (prefs.theme) document.documentElement.setAttribute('data-theme', prefs.theme)
  pushPreferences(seed).catch(() => { /* ignore — retried on next sign-in */ })
}

/** Saves one preference to the account when this device syncs. Callers write localStorage themselves. */
export function savePreference(key: PreferenceKey, value: string) {
  if (getPreferenceScope() !== 'account') return
  pushPreferences({ [key]: value }).catch(() => { /* ignore — local value still applies */ })
}

/**
 * Switches this device between account-synced and device-only preferences.
 * Turning sync on saves this device's current choices to the account.
 */
export async function setPreferenceScope(scope: PreferenceScope) {
  localStorage.setItem(SCOPE_KEY, scope)
  if (scope === 'account') {
    await pushPreferences(readLocalPreferences())
  }
}
