import { ref } from 'vue'
import {
  createDeepIdentificationJob,
  getDeepIdentificationJob,
  cancelDeepIdentificationJob,
  retryDeepIdentificationJob,
  getApiErrorMessage,
} from '@/api/client'
import type { CreateDeepIdentificationJobInput, DeepJob, DeepJobEnvelope, DeepProviderId } from '@/types'

/**
 * Job-lifecycle composable for Deep Agentic Coin Identification (Feature 344).
 * Wraps the additive `/api/deep-identification/jobs` REST contract. Never
 * touches the existing quick-lookup (`lookupCoin`) or saved-coin CRUD paths.
 */
export function useDeepIdentification() {
  const job = ref<DeepJob | null>(null)
  const starting = ref(false)
  const loading = ref(false)
  const cancelling = ref(false)
  const retrying = ref(false)
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

  /**
   * Requests cancellation of a running/queued job (T100). Per the SSE
   * contract, the terminal state itself always arrives via the event
   * stream (or a subsequent GET), not from this response alone - callers
   * should keep listening rather than treat the response job snapshot as
   * final in a cancel-vs-complete race.
   */
  async function cancel(jobId: number): Promise<DeepJob | null> {
    cancelling.value = true
    error.value = ''
    try {
      const { data } = await cancelDeepIdentificationJob(jobId)
      job.value = data.job
      return data.job
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to cancel Deep Analysis.'
      return null
    } finally {
      cancelling.value = false
    }
  }

  /**
   * Starts a new retry job linked to a terminal job (T100). The caller is
   * responsible for navigating to the new job's route (the retry is a new
   * job row, not a resumption of the old one).
   */
  async function retry(jobId: number, input?: { notes?: string; providers?: DeepProviderId[] }): Promise<DeepJob | null> {
    retrying.value = true
    error.value = ''
    try {
      const { data } = await retryDeepIdentificationJob(jobId, input)
      job.value = data.job
      return data.job
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to retry Deep Analysis.'
      return null
    } finally {
      retrying.value = false
    }
  }

  return {
    job,
    starting,
    loading,
    cancelling,
    retrying,
    error,
    start,
    refresh,
    cancel,
    retry,
  }
}

