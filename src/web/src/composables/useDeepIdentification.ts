import { ref } from 'vue'
import { createDeepIdentificationJob, getDeepIdentificationJob, getApiErrorMessage } from '@/api/client'
import type { CreateDeepIdentificationJobInput, DeepJob, DeepJobEnvelope } from '@/types'

/**
 * Job-lifecycle composable for Deep Agentic Coin Identification (Feature 344).
 * Wraps the additive `/api/deep-identification/jobs` REST contract. Never
 * touches the existing quick-lookup (`lookupCoin`) or saved-coin CRUD paths.
 */
export function useDeepIdentification() {
  const job = ref<DeepJob | null>(null)
  const starting = ref(false)
  const loading = ref(false)
  const error = ref('')

  async function start(input: CreateDeepIdentificationJobInput): Promise<DeepJob | null> {
    starting.value = true
    error.value = ''
    try {
      const { data } = await createDeepIdentificationJob(input)
      job.value = data.job
      return data.job
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to start Deep Analysis.'
      return null
    } finally {
      starting.value = false
    }
  }

  async function refresh(jobId: number): Promise<DeepJobEnvelope | null> {
    loading.value = true
    error.value = ''
    try {
      const { data } = await getDeepIdentificationJob(jobId)
      job.value = data.job
      return data
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to load Deep Analysis job.'
      return null
    } finally {
      loading.value = false
    }
  }

  return {
    job,
    starting,
    loading,
    error,
    start,
    refresh,
  }
}
