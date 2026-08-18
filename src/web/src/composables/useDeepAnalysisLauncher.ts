import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useDeepIdentification } from '@/composables/useDeepIdentification'
import { useDeepIdentificationCapability } from '@/composables/useDeepIdentificationCapability'
import { listDeepIdentificationJobs } from '@/api/client'
import type { CreateDeepIdentificationJobInput, DeepJob } from '@/types'

/**
 * Shared Deep Analysis launch flow (T087/F6). `CoinActionsPanel.vue` and
 * `CoinLookupPage.vue` each bolted on their own copy of the same
 * "capability gate -> open modal -> submit -> navigate to /deep-analysis/:id"
 * sequence when Feature 344 shipped the entry point; this composable is the
 * single place that logic now lives, so both call sites stay in lockstep.
 */
export function useDeepAnalysisLauncher() {
  const router = useRouter()
  const deepIdentification = useDeepIdentification()
  const {
    enabled: deepAnalysisEnabled,
    providers: deepAnalysisProviders,
  } = useDeepIdentificationCapability()
  const showDeepAnalysisModal = ref(false)

  // The number of the current user's active Deep Analysis jobs (T088),
  // used to disable the entry point instead of letting the user hit the
  // `MaxActivePerUser` limit and get an error toast. The exact configured
  // limit is admin-only (`DeepIdentificationMaxActivePerUser`, default 1)
  // and is not exposed to normal users by any endpoint - see T088 summary.
  // Any active job at all is treated as "at the limit" so this is accurate
  // for the default and never worse than the previous unwired `disabled`.
  const activeJobCount = ref(0)
  // The id of the (first) currently active job, if any - used so a
  // `job_at_capacity` conflict can point the user straight at the job
  // that is already running instead of just naming the problem.
  const activeJobId = ref<number | null>(null)
  // Set to the running job's id only right after a submit is rejected with
  // `job_at_capacity` (HTTP 409); cleared on every new submit attempt. The
  // start panel uses this to render a "View running analysis" link next to
  // the error text so the user has somewhere to go - that job's own page
  // already has the Cancel control (DeepAnalysisProgressTimeline).
  const capacityConflictJobId = ref<number | null>(null)

  async function refreshActiveJobCount(): Promise<void> {
    if (!deepAnalysisEnabled.value) {
      activeJobCount.value = 0
      activeJobId.value = null
      return
    }
    try {
      const { data } = await listDeepIdentificationJobs({ activeOnly: true })
      activeJobCount.value = data.jobs.length
      activeJobId.value = data.jobs[0]?.id ?? null
    } catch {
      // Fail open on the count probe alone - the backend remains
      // authoritative and still enforces the real limit on submit.
      activeJobCount.value = 0
      activeJobId.value = null
    }
  }

  function openDeepAnalysisModal(): void {
    showDeepAnalysisModal.value = true
  }

  function closeDeepAnalysisModal(): void {
    showDeepAnalysisModal.value = false
  }

  async function submitDeepAnalysis(input: CreateDeepIdentificationJobInput): Promise<DeepJob | null> {
    capacityConflictJobId.value = null
    const job = await deepIdentification.start(input)
    if (job) {
      showDeepAnalysisModal.value = false
      await refreshActiveJobCount()
      await router.push(`/deep-analysis/${job.id}`)
      return job
    }
    // `job_at_capacity` (HTTP 409) means the new submission never started -
    // the modal stays open with the error visible (deepIdentification.error)
    // and `starting` is already back to false, so nothing spins forever.
    // Refresh the active-job probe so the conflict link points at whichever
    // job is actually running right now, not a stale id.
    if (deepIdentification.errorCode.value === 'job_at_capacity') {
      await refreshActiveJobCount()
      capacityConflictJobId.value = activeJobId.value
    }
    return job
  }

  // Probe the count as soon as the capability flag confirms Deep Analysis
  // is enabled (the capability probe itself resolves asynchronously on
  // mount in `useDeepIdentificationCapability`).
  watch(deepAnalysisEnabled, (enabled) => {
    if (enabled) void refreshActiveJobCount()
  }, { immediate: true })

  return {
    deepIdentification,
    deepAnalysisEnabled,
    deepAnalysisProviders,
    showDeepAnalysisModal,
    activeJobCount,
    activeJobId,
    capacityConflictJobId,
    refreshActiveJobCount,
    openDeepAnalysisModal,
    closeDeepAnalysisModal,
    submitDeepAnalysis,
  }
}

export type DeepAnalysisLauncher = ReturnType<typeof useDeepAnalysisLauncher>
