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

  async function refreshActiveJobCount(): Promise<void> {
    if (!deepAnalysisEnabled.value) {
      activeJobCount.value = 0
      return
    }
    try {
      const { data } = await listDeepIdentificationJobs({ activeOnly: true })
      activeJobCount.value = data.jobs.length
    } catch {
      // Fail open on the count probe alone - the backend remains
      // authoritative and still enforces the real limit on submit.
      activeJobCount.value = 0
    }
  }

  function openDeepAnalysisModal(): void {
    showDeepAnalysisModal.value = true
  }

  function closeDeepAnalysisModal(): void {
    showDeepAnalysisModal.value = false
  }

  async function submitDeepAnalysis(input: CreateDeepIdentificationJobInput): Promise<DeepJob | null> {
    const job = await deepIdentification.start(input)
    if (job) {
      showDeepAnalysisModal.value = false
      await refreshActiveJobCount()
      await router.push(`/deep-analysis/${job.id}`)
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
    refreshActiveJobCount,
    openDeepAnalysisModal,
    closeDeepAnalysisModal,
    submitDeepAnalysis,
  }
}

export type DeepAnalysisLauncher = ReturnType<typeof useDeepAnalysisLauncher>
