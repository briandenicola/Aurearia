import { describe, it, expect, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { useTrayPreference } from '../useTrayPreference'

describe('useTrayPreference', () => {
  const STORAGE_KEY = 'tray:feltColor'

  beforeEach(() => {
    // Clear localStorage before each test
    localStorage.clear()
  })

  it('returns default color when localStorage is empty', () => {
    const { feltColor } = useTrayPreference()
    expect(feltColor.value).toBe('red')
  })

  it('reads felt color from localStorage', () => {
    localStorage.setItem(STORAGE_KEY, 'green')
    const { feltColor } = useTrayPreference()
    expect(feltColor.value).toBe('green')
  })

  it('updates feltColor reactively', () => {
    const { feltColor } = useTrayPreference()
    expect(feltColor.value).toBe('red')
    
    feltColor.value = 'navy'
    expect(feltColor.value).toBe('navy')
  })

  it('persists feltColor to localStorage on change', async () => {
    const { feltColor } = useTrayPreference()
    feltColor.value = 'green'
    
    await nextTick()
    expect(localStorage.getItem(STORAGE_KEY)).toBe('green')
  })

  it('handles invalid color value by using default', () => {
    localStorage.setItem(STORAGE_KEY, 'invalid-color')
    const { feltColor } = useTrayPreference()
    expect(feltColor.value).toBe('red')
  })

  it('persists navy theme', async () => {
    const { feltColor } = useTrayPreference()
    feltColor.value = 'navy'
    
    await nextTick()
    expect(localStorage.getItem(STORAGE_KEY)).toBe('navy')
    
    // Simulate page reload
    const { feltColor: feltColor2 } = useTrayPreference()
    expect(feltColor2.value).toBe('navy')
  })
})
