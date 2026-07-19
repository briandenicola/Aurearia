import { ref } from 'vue'
import { registerSW } from 'virtual:pwa-register'

const updateAvailable = ref(false)

const applyUpdate = registerSW({
  immediate: true,
  onNeedRefresh() {
    updateAvailable.value = true
  },
  onOfflineReady() {
    // App ready to work offline
  },
  onRegisteredSW(_swScriptUrl, registration) {
    // Check for updates every hour
    if (registration) {
      setInterval(() => {
        registration.update()
      }, 60 * 60 * 1000)
    }
  },
})

export function usePwaUpdate() {
  return {
    updateAvailable,
    refresh: () => applyUpdate(true),
  }
}
