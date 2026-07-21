import { ref } from 'vue'
import { registerSW } from 'virtual:pwa-register'

const updateAvailable = ref(false)
const UPDATE_CHECK_INTERVAL_MS = 60 * 60 * 1000
const UPDATE_CHECK_THROTTLE_MS = 5 * 60 * 1000
let lastUpdateCheck = -UPDATE_CHECK_THROTTLE_MS

function requestUpdateCheck(registration: ServiceWorkerRegistration) {
  const now = Date.now()
  if (now >= lastUpdateCheck && now - lastUpdateCheck < UPDATE_CHECK_THROTTLE_MS) return
  lastUpdateCheck = now
  void registration.update().catch((err) => {
    console.warn('[pwa] Service worker update check failed', err)
  })
}

const applyUpdate = registerSW({
  immediate: true,
  onNeedRefresh() {
    updateAvailable.value = true
  },
  onOfflineReady() {
    // App ready to work offline
  },
  onRegisteredSW(_swScriptUrl, registration) {
    if (!registration) return

    requestUpdateCheck(registration)
    setInterval(() => requestUpdateCheck(registration), UPDATE_CHECK_INTERVAL_MS)

    window.addEventListener('focus', () => requestUpdateCheck(registration))
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') {
        requestUpdateCheck(registration)
      }
    })
  },
})

export function usePwaUpdate() {
  return {
    updateAvailable,
    refresh: () => applyUpdate(true),
  }
}
