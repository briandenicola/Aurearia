import { onMounted, ref } from 'vue'
import { getDeepIdentificationCapability } from '@/api/client'

/**
 * useDeepIdentificationCapability probes the backend-authoritative Deep
 * Analysis feature flag (FR-008) so the entry point can be hidden while the
 * feature is disabled. It reads only the capability boolean, never admin
 * settings, and fails closed: if the probe is unavailable the entry point
 * stays hidden and the backend remains the source of truth for job creation.
 */
export function useDeepIdentificationCapability() {
  const enabled = ref(false)
  const loaded = ref(false)

  async function load(): Promise<void> {
    try {
      const { data } = await getDeepIdentificationCapability()
      enabled.value = data?.enabled === true
    } catch {
      enabled.value = false
    } finally {
      loaded.value = true
    }
  }

  onMounted(load)

  return { enabled, loaded, load }
}
