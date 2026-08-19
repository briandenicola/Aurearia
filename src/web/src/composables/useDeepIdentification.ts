import { ref } from 'vue'
import {
  createDeepIdentificationJob,
  getDeepIdentificationJob,
  cancelDeepIdentificationJob,
  retryDeepIdentificationJob,
  patchDeepIdentificationProposal,
  applyDeepIdentificationProposal,
  deleteDeepIdentificationJob,
  getApiErrorMessage,
  getApiErrorCode,
} from '@/api/client'
import type {
  ApplyDeepIdentificationProposalInput,
  CreateDeepIdentificationJobInput,
  DeepApplyResult,
  DeepJob,
  DeepJobEnvelope,
  DeepProposal,
  DeepProposalFieldEdit,
  DeepProviderId,
  DeepReport,
} from '@/types'

/**
 * Job-lifecycle composable for Deep Agentic Coin Identification (Feature 344).
 * Wraps the additive `/api/deep-identification/jobs` REST contract. Never
 * touches the existing quick-lookup (`lookupCoin`) or saved-coin CRUD paths.
 */
export function useDeepIdentification() {
  const job = ref<DeepJob | null>(null)
  const report = ref<DeepReport | null>(null)
  const proposal = ref<DeepProposal | null>(null)
  const starting = ref(false)
  const loading = ref(false)
  const cancelling = ref(false)
  const retrying = ref(false)
  const applying = ref(false)
  const deleting = ref(false)
  const error = ref('')
  // Machine-readable code for the most recent `start()` failure, e.g.
  // `job_at_capacity` (HTTP 409: a genuinely different in-flight job is
  // already running for this user - distinct from an idempotent duplicate
  // submission, which still returns 200 with `reused: true` and never
  // reaches this branch). Callers use this to offer a targeted recovery
  // action instead of only showing the generic error text.
  const errorCode = ref('')

  async function start(input: CreateDeepIdentificationJobInput): Promise<DeepJob | null> {
    starting.value = true
    error.value = ''
    errorCode.value = ''
    try {
      const { data } = await createDeepIdentificationJob(input)
      job.value = data.job
      return data.job
    } catch (err) {
      errorCode.value = getApiErrorCode(err)
      const fallback = errorCode.value === 'job_at_capacity'
        ? 'An analysis is already running. Wait for it to finish or cancel it.'
        : 'Unable to start Deep Analysis.'
      error.value = getApiErrorMessage(err) || fallback
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
      report.value = data.report ?? null
      proposal.value = data.proposal ?? null
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

  /**
   * Saves one field's owner edit/accept-reject decision (T121/T122). Never
   * writes coin/draft data - only the job's own proposal (FR-031/FR-032).
   * Applies an optimistic local update to `proposal` immediately, then
   * reconciles with the server-confirmed document.
   */
  async function updateProposalField(jobId: number, name: string, edit: DeepProposalFieldEdit): Promise<DeepProposal | null> {
    error.value = ''
    const current = proposal.value
    if (current) {
      const existing = current.fields[name]
      if (existing) {
        const next: DeepProposal = {
          ...current,
          fields: {
            ...current.fields,
            [name]: {
              ...existing,
              ...(edit.ownerValue !== undefined ? { ownerValue: edit.ownerValue, ownerEdited: true } : {}),
              ...(edit.accepted !== undefined ? { accepted: edit.accepted ?? null } : {}),
            },
          },
        }
        proposal.value = next
      }
    }
    try {
      const { data } = await patchDeepIdentificationProposal(jobId, { fields: { [name]: edit } })
      proposal.value = data
      return data
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to save your proposal edit.'
      return null
    }
  }

  /**
   * Saves multiple field decisions in a single request. The server applies
   * every edit atomically (`UpdateProposal` validates all names up front,
   * then writes the whole document in one save) — prefer this over calling
   * `updateProposalField` in a loop whenever more than one field needs to be
   * persisted at once (e.g. RD-3's pre-Apply confidence-default finalize),
   * both for round-trip cost and to avoid a half-written proposal if one of
   * several individual PATCHes were to fail partway through.
   */
  async function updateProposalFields(jobId: number, edits: Record<string, DeepProposalFieldEdit>): Promise<DeepProposal | null> {
    error.value = ''
    if (Object.keys(edits).length === 0) return proposal.value
    const current = proposal.value
    if (current) {
      const nextFields = { ...current.fields }
      for (const [name, edit] of Object.entries(edits)) {
        const existing = nextFields[name]
        if (!existing) continue
        nextFields[name] = {
          ...existing,
          ...(edit.ownerValue !== undefined ? { ownerValue: edit.ownerValue, ownerEdited: true } : {}),
          ...(edit.accepted !== undefined ? { accepted: edit.accepted ?? null } : {}),
        }
      }
      proposal.value = { ...current, fields: nextFields }
    }
    try {
      const { data } = await patchDeepIdentificationProposal(jobId, { fields: edits })
      proposal.value = data
      return data
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to save your proposal edits.'
      return null
    }
  }

  /**
   * Confirms the proposal through the existing Go-owned write path
   * (T121/T124): never called without an explicit user confirm action.
   */
  async function applyProposal(jobId: number, input: ApplyDeepIdentificationProposalInput): Promise<DeepApplyResult | null> {
    applying.value = true
    error.value = ''
    try {
      const { data } = await applyDeepIdentificationProposal(jobId, input)
      if (job.value) {
        job.value = {
          ...job.value,
          appliedCoinId: data.coinId ?? job.value.appliedCoinId,
          appliedDraftId: data.draftId ?? job.value.appliedDraftId,
          appliedAt: data.appliedAt,
        }
      }
      return data
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to apply the Deep Analysis proposal.'
      return null
    } finally {
      applying.value = false
    }
  }

  /**
   * Hard-deletes a terminal, owned job (spec 354 T017/T042/T044). Only
   * ever called after an explicit owner confirm — the job/report/proposal
   * are cleared locally on success so the caller can navigate away.
   */
  async function deleteJob(jobId: number): Promise<boolean> {
    deleting.value = true
    error.value = ''
    try {
      await deleteDeepIdentificationJob(jobId)
      if (job.value?.id === jobId) {
        job.value = null
        report.value = null
        proposal.value = null
      }
      return true
    } catch (err) {
      error.value = getApiErrorMessage(err) || 'Unable to delete this Deep Analysis run.'
      return false
    } finally {
      deleting.value = false
    }
  }

  return {
    job,
    report,
    proposal,
    starting,
    loading,
    cancelling,
    retrying,
    applying,
    deleting,
    error,
    errorCode,
    start,
    refresh,
    cancel,
    retry,
    updateProposalField,
    updateProposalFields,
    applyProposal,
    deleteJob,
  }
}

