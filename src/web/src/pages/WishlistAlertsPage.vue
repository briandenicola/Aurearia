<template>
  <div class="container">
    <div class="page-header">
      <div>
        <h1>Wishlist Search Alerts</h1>
        <p class="subtitle">Discovery alerts find acquisition ideas. Availability checking for saved wishlist URLs remains separate.</p>
      </div>
      <div v-if="isPwa" class="pwa-actions">
        <button class="pwa-icon-btn" type="button" title="New Search Alert" @click="startCreate">
          <Search :size="22" />
        </button>
      </div>
      <button v-else class="btn btn-primary" type="button" @click="startCreate"><Search :size="16" /> New Search Alert</button>
    </div>

    <p v-if="error" class="page-error">{{ error }}</p>
    <div v-if="loading" class="loading-overlay"><div class="spinner"></div></div>
    <div v-else-if="!alerts.length" class="empty-state">
      <h3>No search alerts yet</h3>
      <p>Create a discovery alert with criteria such as ruler, type, price range, and source domains.</p>
      <button class="btn btn-primary" type="button" @click="startCreate"><Search :size="16" /> Create Search Alert</button>
    </div>

    <div v-else class="alerts-layout" :class="{ 'has-selection': selectedAlert }">
      <aside class="alerts-list" aria-label="Search alerts">
        <article v-for="alert in alerts" :key="alert.id" class="alert-card" :class="{ selected: selectedAlert?.id === alert.id }">
          <button class="select-alert" type="button" @click="selectAlert(alert)">
            <h2>{{ alert.name }}</h2>
            <AlertCriteriaSummary :alert="alert" />
            <p class="manual-copy">Cadence metadata only. Run Now starts manual in-app review.</p>
          </button>
          <div class="card-actions">
            <button class="btn btn-secondary btn-sm" type="button" @click="edit(alert)"><Pencil :size="14" /> Edit</button>
            <button class="btn btn-secondary btn-sm" type="button" @click="toggle(alert)">{{ alert.isActive ? 'Disable' : 'Enable' }}</button>
            <button class="btn btn-danger btn-sm" type="button" @click="remove(alert)"><Trash2 :size="14" /> Delete</button>
          </div>
        </article>
      </aside>

      <main v-if="selectedAlert" class="review-panel">
        <section class="selected-summary">
          <div>
            <p class="section-label">Selected alert</p>
            <h2>{{ selectedAlert.name }}</h2>
            <p class="subtitle">Candidates stay in this review queue until you dismiss, restore, or explicitly save them as wishlist items.</p>
          </div>
          <button class="btn btn-primary" type="button" :disabled="running || !selectedAlert.isActive" @click="runNow">
            <Play :size="16" /> {{ running ? 'Running...' : 'Run Now' }}
          </button>
        </section>
        <p v-if="!selectedAlert.isActive" class="message">Enable this search alert before running discovery.</p>
        <p v-if="runMessage" class="message">{{ runMessage }}</p>

        <AlertRunHistory
          :runs="runs"
          :selected-run="selectedRun"
          :selected-run-id="selectedRun?.id ?? null"
          :loading="runsLoading"
          :error="runsError"
          @select="loadRunDetail"
          @refresh="loadRuns"
        />

        <section class="candidate-section">
          <div class="section-header">
            <div>
              <h3>Candidate review</h3>
              <p class="subtitle">Source-backed acquisition candidates, not saved wishlist URL availability results.</p>
            </div>
            <div class="filters">
              <select v-model="candidateState" @change="loadCandidates">
                <option value="">Active and needs review</option>
                <option value="active">Active</option>
                <option value="needs_review">Needs review</option>
                <option value="dismissed">Dismissed</option>
                <option value="converted">Converted</option>
                <option value="suppressed">Suppressed</option>
              </select>
              <select v-model="provenanceStatus" @change="loadCandidates">
                <option value="">Any provenance</option>
                <option value="verified">Verified</option>
                <option value="partial">Partial</option>
                <option value="unverified">Unverified</option>
              </select>
            </div>
          </div>

          <p v-if="candidatesError" class="page-error">{{ candidatesError }}</p>
          <div v-if="candidatesLoading" class="message">Loading candidates...</div>
          <div v-else-if="!candidates.length" class="empty-inline">No candidates match this review filter.</div>
          <div v-else class="candidate-list">
            <CandidateReviewCard
              v-for="candidate in candidates"
              :key="candidate.id"
              :candidate="candidate"
              :busy="candidateBusyId === candidate.id"
              :duplicate-warnings="duplicateWarnings[candidate.id] ?? []"
              @dismiss="dismissCandidate"
              @restore="restoreCandidate"
              @convert="convertCandidate"
            />
          </div>

          <details class="criteria-adjustment">
            <summary>Adjust criteria from this review context</summary>
            <p class="subtitle">Applies to future search alert runs only. Converted wishlist items are not changed.</p>
            <AlertForm :alert="selectedAlert" :saving="adjusting" @save="adjustCriteria" @cancel="noop" />
          </details>
        </section>
      </main>
    </div>

    <section v-if="creating || editing" class="editor-panel">
      <h2>{{ editing ? 'Edit Search Alert' : 'Create Search Alert' }}</h2>
      <AlertForm :alert="editing" :saving="saving" @save="save" @cancel="closeEditor" />
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Search, Pencil, Trash2, Play } from 'lucide-vue-next'
import AlertCriteriaSummary from '@/components/wishlist-alerts/AlertCriteriaSummary.vue'
import AlertForm from '@/components/wishlist-alerts/AlertForm.vue'
import AlertRunHistory from '@/components/wishlist-alerts/AlertRunHistory.vue'
import CandidateReviewCard from '@/components/wishlist-alerts/CandidateReviewCard.vue'
import { usePwa } from '@/composables/usePwa'
import {
  adjustWishlistSearchAlertCriteria,
  convertWishlistSearchAlertCandidate,
  createWishlistSearchAlert,
  deleteWishlistSearchAlert,
  dismissWishlistSearchAlertCandidate,
  getApiErrorMessage,
  getWishlistSearchAlertRun,
  listWishlistSearchAlertCandidates,
  listWishlistSearchAlertRuns,
  listWishlistSearchAlerts,
  restoreWishlistSearchAlertCandidate,
  runWishlistSearchAlert,
  updateWishlistSearchAlert,
} from '@/api/client'
import type { AlertCandidate, AlertCandidateState, AlertRun, CandidateDismissalReason, CandidateProvenanceStatus, CoinMutationPayload, WishlistSearchAlert, WishlistSearchAlertInput } from '@/types'

const alerts = ref<WishlistSearchAlert[]>([])
const selectedAlertId = ref<number | null>(null)
const loading = ref(false)
const saving = ref(false)
const running = ref(false)
const adjusting = ref(false)
const error = ref('')
const runMessage = ref('')
const editing = ref<WishlistSearchAlert | null>(null)
const creating = ref(false)
const runs = ref<AlertRun[]>([])
const selectedRun = ref<AlertRun | null>(null)
const runsLoading = ref(false)
const runsError = ref('')
const candidates = ref<AlertCandidate[]>([])
const candidatesLoading = ref(false)
const candidatesError = ref('')
const candidateState = ref<AlertCandidateState | ''>('')
const provenanceStatus = ref<CandidateProvenanceStatus | ''>('')
const candidateBusyId = ref<number | null>(null)
const duplicateWarnings = ref<Record<number, string[]>>({})
const { isPwa } = usePwa()
const RUN_POLL_INTERVAL_MS = 3000
let runPollTimer: ReturnType<typeof setTimeout> | null = null

const selectedAlert = computed(() => alerts.value.find((alert) => alert.id === selectedAlertId.value) ?? null)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await listWishlistSearchAlerts({ page: 1, limit: 100 })
    alerts.value = res.data.alerts
    if (selectedAlertId.value && !alerts.value.some((alert) => alert.id === selectedAlertId.value)) clearSelection()
    if (selectedAlertId.value) await loadSelectedData()
    else clearReviewState()
  } catch (err) {
    error.value = getApiErrorMessage(err) || 'Failed to load search alerts.'
  } finally {
    loading.value = false
  }
}

async function loadSelectedData() {
  await Promise.all([loadRuns(), loadCandidates()])
}

function startCreate() { creating.value = true; editing.value = null }
function edit(alert: WishlistSearchAlert) { editing.value = alert; creating.value = false }
function closeEditor() { creating.value = false; editing.value = null }
function noop() {}

function clearReviewState() {
  clearRunPollTimer()
  running.value = false
  selectedRun.value = null
  runs.value = []
  runsError.value = ''
  candidates.value = []
  candidatesError.value = ''
  duplicateWarnings.value = {}
  runMessage.value = ''
}

function clearSelection() {
  selectedAlertId.value = null
  clearReviewState()
}

async function selectAlert(alert: WishlistSearchAlert) {
  clearRunPollTimer()
  selectedAlertId.value = alert.id
  clearReviewState()
  await loadSelectedData()
}

async function save(input: WishlistSearchAlertInput) {
  saving.value = true
  error.value = ''
  try {
    if (editing.value) await updateWishlistSearchAlert(editing.value.id, input)
    else await createWishlistSearchAlert(input)
    closeEditor()
    await load()
  } catch (err) {
    error.value = getApiErrorMessage(err) || 'Failed to save search alert.'
  } finally {
    saving.value = false
  }
}

async function toggle(alert: WishlistSearchAlert) {
  await updateWishlistSearchAlert(alert.id, {
    name: alert.name,
    criteria: toInputCriteria(alert),
    cadence: alert.cadence,
    isActive: !alert.isActive,
  })
  await load()
}

async function remove(alert: WishlistSearchAlert) {
  if (!confirm(`Delete search alert "${alert.name}"?`)) return
  await deleteWishlistSearchAlert(alert.id)
  if (selectedAlertId.value === alert.id) clearSelection()
  await load()
}

async function runNow() {
  if (!selectedAlert.value) return
  running.value = true
  runMessage.value = ''
  error.value = ''
  let queued = false
  try {
    const alertId = selectedAlert.value.id
    const res = await runWishlistSearchAlert(selectedAlert.value.id, 20)
    queued = true
    runMessage.value = 'Search alert run queued. You can leave this page; results will appear in run history.'
    await Promise.all([load(), loadRuns()])
    if (res.data.runId) await pollAlertRun(alertId, res.data.runId)
  } catch (err) {
    error.value = getApiErrorMessage(err) || 'Search alert discovery failed. Try again later.'
    running.value = false
  } finally {
    if (!queued) running.value = false
  }
}

async function loadRuns() {
  if (!selectedAlertId.value) return
  runsLoading.value = true
  runsError.value = ''
  try {
    const res = await listWishlistSearchAlertRuns(selectedAlertId.value, { page: 1, limit: 20 })
    runs.value = res.data.runs
    if (!selectedRun.value && runs.value[0]) {
      const run = await loadRunDetail(runs.value[0].id)
      if (run && !isTerminalRunStatus(run.status)) {
        running.value = true
        scheduleRunPoll(run.alertId, run.id)
      }
    }
  } catch (err) {
    runsError.value = getApiErrorMessage(err) || 'Failed to load run history.'
  } finally {
    runsLoading.value = false
  }
}

async function loadRunDetail(runId: number) {
  if (!selectedAlertId.value) return null
  const res = await getWishlistSearchAlertRun(selectedAlertId.value, runId)
  selectedRun.value = res.data
  return res.data
}

async function loadCandidates() {
  if (!selectedAlertId.value) return
  candidatesLoading.value = true
  candidatesError.value = ''
  try {
    const res = await listWishlistSearchAlertCandidates(selectedAlertId.value, { state: candidateState.value, provenanceStatus: provenanceStatus.value, page: 1, limit: 50 })
    candidates.value = res.data.candidates
  } catch (err) {
    candidatesError.value = getApiErrorMessage(err) || 'Failed to load candidates.'
  } finally {
    candidatesLoading.value = false
  }
}

async function dismissCandidate(candidate: AlertCandidate, reason: CandidateDismissalReason, notes: string) {
  if (!selectedAlertId.value) return
  candidateBusyId.value = candidate.id
  try {
    await dismissWishlistSearchAlertCandidate(selectedAlertId.value, candidate.id, { reason, notes })
    await loadCandidates()
  } catch (err) {
    candidatesError.value = getApiErrorMessage(err) || 'Failed to dismiss candidate.'
  } finally {
    candidateBusyId.value = null
  }
}

async function restoreCandidate(candidate: AlertCandidate) {
  if (!selectedAlertId.value) return
  candidateBusyId.value = candidate.id
  try {
    await restoreWishlistSearchAlertCandidate(selectedAlertId.value, candidate.id)
    await loadCandidates()
  } catch (err) {
    candidatesError.value = getApiErrorMessage(err) || 'Failed to restore candidate.'
  } finally {
    candidateBusyId.value = null
  }
}

async function convertCandidate(candidate: AlertCandidate, coin: CoinMutationPayload, acknowledgeDuplicateWarning: boolean) {
  if (!selectedAlertId.value) return
  candidateBusyId.value = candidate.id
  candidatesError.value = ''
  try {
    const res = await convertWishlistSearchAlertCandidate(selectedAlertId.value, candidate.id, { coin, acknowledgeDuplicateWarning })
    duplicateWarnings.value[candidate.id] = res.data.warnings ?? []
    await loadCandidates()
  } catch (err) {
    const response = (err as { response?: { status?: number; data?: { warnings?: string[] } } }).response
    if (response?.status === 409 && response.data?.warnings?.length) duplicateWarnings.value[candidate.id] = response.data.warnings
    candidatesError.value = getApiErrorMessage(err) || 'Review duplicate warnings or required fields before conversion.'
  } finally {
    candidateBusyId.value = null
  }
}

async function adjustCriteria(input: WishlistSearchAlertInput) {
  if (!selectedAlertId.value) return
  adjusting.value = true
  try {
    const candidateIds = candidates.value.slice(0, 20).map((candidate) => candidate.id)
    await adjustWishlistSearchAlertCriteria(selectedAlertId.value, { candidateIds, criteria: input.criteria })
    await load()
  } catch (err) {
    error.value = getApiErrorMessage(err) || 'Failed to adjust criteria.'
  } finally {
    adjusting.value = false
  }
}

function toInputCriteria(alert: WishlistSearchAlert): WishlistSearchAlertInput['criteria'] {
  return {
    rulerOrIssuer: alert.rulerOrIssuer,
    coinType: alert.coinType,
    dateFrom: alert.dateFrom,
    dateTo: alert.dateTo,
    mint: alert.mint,
    material: alert.material,
    gradeOrCondition: alert.gradeOrCondition,
    priceMin: alert.priceMin,
    priceMax: alert.priceMax,
    currency: alert.currency,
    dealerPreference: alert.dealerPreference,
    sourceFilters: [...alert.sourceFilters],
    keywords: alert.keywords,
    notes: alert.notes,
  }
}

function runResultMessage(status: string, count: number, duplicates: number) {
  if (status === 'queued' || status === 'running') return 'Run is still processing. Results will appear when discovery finishes.'
  if (status === 'failed') return 'Run failed with a stored, sanitized error. Review run history for details.'
  if (status === 'rate_limited') return 'Run was rate limited. Review run history before retrying.'
  if (count === 0) return 'Run completed with no candidates. Consider broadening criteria.'
  return `Run ${status.replace(/_/g, ' ')} with ${count} candidates and ${duplicates} duplicates.`
}

async function pollAlertRun(alertId: number, runId: number) {
  if (selectedAlertId.value !== alertId) return
  try {
    const res = await getWishlistSearchAlertRun(alertId, runId)
    selectedRun.value = res.data
    if (isTerminalRunStatus(res.data.status)) {
      await finishAlertRun(res.data)
      return
    }
  } catch {
    // Keep polling; transient network errors should not lose the backend-owned run.
  }
  scheduleRunPoll(alertId, runId)
}

function scheduleRunPoll(alertId: number, runId: number) {
  clearRunPollTimer()
  runPollTimer = setTimeout(() => {
    void pollAlertRun(alertId, runId)
  }, RUN_POLL_INTERVAL_MS)
}

async function finishAlertRun(run: AlertRun) {
  clearRunPollTimer()
  running.value = false
  runMessage.value = runResultMessage(run.status, run.resultCount, run.duplicateCount)
  await Promise.all([load(), loadRuns(), loadCandidates()])
  await loadRunDetail(run.id)
}

function isTerminalRunStatus(status: AlertRun['status']) {
  return ['completed', 'failed', 'partial', 'rate_limited', 'cancelled'].includes(status)
}

function clearRunPollTimer() {
  if (runPollTimer) {
    clearTimeout(runPollTimer)
    runPollTimer = null
  }
}

onMounted(load)
onBeforeUnmount(clearRunPollTimer)
</script>

<style scoped>
.subtitle, .manual-copy, .message { color: var(--text-muted); margin: 0.25rem 0 0; }
.alerts-layout { display: grid; gap: 1rem; align-items: start; }
.alerts-list, .review-panel, .candidate-section { display: grid; gap: 1rem; }
.review-panel { border-top: 1px solid var(--border-accent); padding-top: 1rem; }
.alert-card, .editor-panel, .selected-summary, .candidate-section, .criteria-adjustment { border: 1px solid var(--border-subtle); border-radius: var(--radius-md); background: var(--bg-card); padding: 1rem; }
.alert-card.selected { border-color: var(--accent-gold); box-shadow: var(--shadow-glow); }
.select-alert { width: 100%; border: 0; background: transparent; color: inherit; padding: 0; text-align: left; cursor: pointer; }
.alert-card h2, .selected-summary h2 { margin: 0 0 0.5rem; font-size: 1.2rem; }
.card-actions, .section-header, .filters, .selected-summary { display: flex; gap: 0.75rem; flex-wrap: wrap; justify-content: space-between; align-items: flex-start; }
.filters select { border: 1px solid var(--border-subtle); border-radius: var(--radius-sm); padding: 0.5rem; background: var(--bg-input); color: var(--text-primary); }
.candidate-list { display: grid; gap: 1rem; }
.empty-inline { border: 1px dashed var(--border-subtle); border-radius: var(--radius-sm); padding: 1rem; color: var(--text-muted); text-align: center; }
.criteria-adjustment summary { color: var(--accent-gold); cursor: pointer; }
/*
 * :deep() audit — child-component class override
 * .alert-form is rendered inside WishlistAlertForm. The criteria-adjustment
 * detail/summary controls when the form is shown inline (vs. in the drawer);
 * the margin-top is positional context from this page, not the child's concern.
 */
.criteria-adjustment :deep(.alert-form) { margin-top: 0.75rem; }
.editor-panel { margin-top: 1rem; }
.page-error { color: var(--accent-bronze); }
.section-label { font-size: 0.7rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: var(--text-muted); margin: 0; }
@media (min-width: 901px) {
  .alerts-layout.has-selection { grid-template-columns: minmax(260px, 0.9fr) minmax(0, 1.6fr); }
  .review-panel { border-top: 0; border-left: 1px solid var(--border-accent); padding-top: 0; padding-left: 1rem; }
}
</style>
