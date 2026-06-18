import { ref, watch } from 'vue'

export type FeltColor = 'red' | 'green' | 'navy'

const STORAGE_KEY = 'tray:feltColor'
const DEFAULT_COLOR: FeltColor = 'red'

const validColors: FeltColor[] = ['red', 'green', 'navy']

/**
 * Composable for managing tray felt color theme preference
 * Persists to localStorage
 */
export function useTrayPreference() {
  // Read from localStorage with validation
  const stored = localStorage.getItem(STORAGE_KEY)
  const isValid = stored && validColors.includes(stored as FeltColor)
  const initialColor: FeltColor = isValid ? (stored as FeltColor) : DEFAULT_COLOR

  const feltColor = ref<FeltColor>(initialColor)

  // Persist to localStorage on change
  watch(feltColor, (newColor) => {
    localStorage.setItem(STORAGE_KEY, newColor)
  })

  return {
    feltColor,
  }
}
