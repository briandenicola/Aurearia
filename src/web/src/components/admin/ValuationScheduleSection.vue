<template>
  <div>
    <h3 class="subsection-title">Collection Valuation</h3>
    <div class="avail-settings">
      <div class="form-group avail-toggle-row">
        <label class="form-label">Enable Scheduled Valuation</label>
        <label class="toggle-switch">
          <input
            type="checkbox"
            :checked="settings.ValuationCheckEnabled === 'true'"
            @change="settings.ValuationCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="toggle-slider"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.ValuationCheckStartTime"
          class="form-input avail-interval-input"
          type="time"
        />
        <span class="form-hint">The valuation cycle starts at this time on scheduled days.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (days)</label>
        <input
          v-model="settings.ValuationCheckIntervalDays"
          class="form-input avail-interval-input"
          type="number"
          min="1"
          step="1"
        />
        <span class="form-hint">How often to run (e.g. 7 = weekly). AI valuations are costly so daily runs are not recommended.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Max Coins Per Run</label>
        <input
          v-model="settings.ValuationMaxCoins"
          class="form-input avail-interval-input"
          type="number"
          min="1"
          step="10"
        />
        <span class="form-hint">Limit how many coins are valuated per run to control AI costs.</span>
      </div>
      <div class="avail-save-row">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save-settings')">
          {{ settingsSaving ? 'Saving...' : 'Save Valuation Settings' }}
        </button>
        <button class="btn btn-secondary btn-sm" :disabled="valTriggerLoading" @click="triggerManualValuation()">
          {{ valTriggerLoading ? 'Starting...' : 'Run Now' }}
        </button>
        <span v-if="valSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': valSettingsError }">{{ valSettingsMsg }}</span>
      </div>
    </div>

    <hr class="section-divider" />
    <h3 class="subsection-title">Valuation Run History</h3>

    <div v-if="valLoading" class="loading-overlay"><div class="spinner"></div></div>
    <div v-else-if="valRuns.length === 0" class="logs-empty">No valuation runs recorded yet.</div>
    <template v-else>
      <table class="users-table avail-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Trigger</th>
            <th>Status</th>
            <th>Checked</th>
            <th>Updated</th>
            <th>Skipped</th>
            <th>Errors</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in valRuns" :key="run.id">
            <tr class="avail-row" :class="{ 'avail-row-expanded': valExpandedRunId === run.id }" @click="toggleValRunDetail(run.id)">
              <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
              <td>{{ run.triggerType }}</td>
              <td>
                <span class="val-status-badge" :class="'val-status-' + run.status">{{ run.status }}</span>
                <span v-if="run.status === 'running' && run.totalCoins > 0" class="val-progress">
                  {{ run.coinsChecked + run.coinsSkipped + run.errors }} / {{ run.totalCoins }}
                </span>
                <button v-if="run.status === 'running'" class="btn-cancel-run" @click.stop="cancelRun(run.id)">Cancel</button>
              </td>
              <td>{{ run.coinsChecked }}</td>
              <td class="avail-count-available">{{ run.coinsUpdated }}</td>
              <td class="avail-count-unknown">{{ run.coinsSkipped }}</td>
              <td class="avail-count-unavailable">{{ run.errors }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="valExpandedRunId === run.id && valExpandedResults" class="avail-detail-row">
              <td colspan="8">
                <div v-if="valExpandedLoading" class="loading-overlay"><div class="spinner"></div></div>
                <table v-else-if="valExpandedResults.length" class="avail-detail-table val-detail-table">
                  <thead>
                    <tr>
                      <th>Coin</th>
                      <th>Previous</th>
                      <th>Estimated</th>
                      <th>Confidence</th>
                      <th>Status</th>
                      <th>Reasoning</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="r in valExpandedResults" :key="r.id">
                      <td>{{ r.coinName }}</td>
                      <td>{{ r.previousValue != null ? `$${r.previousValue.toFixed(2)}` : '--' }}</td>
                      <td class="val-value">{{ r.estimatedValue > 0 ? `$${r.estimatedValue.toFixed(2)}` : '--' }}</td>
                      <td>
                        <span v-if="r.confidence" class="val-confidence" :class="'val-conf-' + r.confidence">{{ r.confidence }}</span>
                        <span v-else>--</span>
                      </td>
                      <td>
                        <span class="listing-status-badge" :class="'val-result-' + r.status">{{ r.status }}</span>
                      </td>
                      <td class="avail-reason">{{ r.reasoning || r.errorMessage || '--' }}</td>
                    </tr>
                  </tbody>
                </table>
                <p v-else class="logs-empty">No results for this run.</p>
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <div class="avail-pagination">
        <button class="btn btn-secondary btn-sm" :disabled="valPage <= 1" @click="valPage--; loadValRuns()">Prev</button>
        <span class="avail-page-info">Page {{ valPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="valRuns.length < 5" @click="valPage++; loadValRuns()">Next</button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getValuationRuns, getValuationRunDetail, triggerValuation, cancelValuationRun } from '@/api/client'
import type { AppSettings, ValuationRun } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
  valSettingsMsg: string
  valSettingsError: boolean
}>()

const emit = defineEmits<{
  'save-settings': []
  'update:valSettingsMsg': [value: string]
  'update:valSettingsError': [value: boolean]
}>()

const valRuns = ref<ValuationRun[]>([])
const valTotal = ref(0)
const valPage = ref(1)
const valLoading = ref(false)
const valTriggerLoading = ref(false)
const valExpandedRunId = ref<number | null>(null)
const valExpandedResults = ref<ValuationRun['results']>(undefined)
const valExpandedLoading = ref(false)
let valPollTimer: ReturnType<typeof setInterval> | null = null

async function loadValRuns() {
  valLoading.value = true
  try {
    const res = await getValuationRuns(valPage.value, 5)
    valRuns.value = res.data.runs ?? []
    valTotal.value = res.data.total ?? 0

    // Auto-poll while any run is still "running"
    const hasRunning = valRuns.value.some(r => r.status === 'running')
    if (hasRunning && !valPollTimer) {
      valPollTimer = setInterval(() => { loadValRuns() }, 5000)
    } else if (!hasRunning && valPollTimer) {
      clearInterval(valPollTimer)
      valPollTimer = null
    }
  } catch { /* ignore */ } finally {
    valLoading.value = false
  }
}

async function toggleValRunDetail(runId: number) {
  if (valExpandedRunId.value === runId) {
    valExpandedRunId.value = null
    valExpandedResults.value = undefined
    return
  }
  valExpandedRunId.value = runId
  valExpandedResults.value = []
  valExpandedLoading.value = true
  try {
    const res = await getValuationRunDetail(runId)
    valExpandedResults.value = res.data.results ?? []
  } catch {
    valExpandedResults.value = []
  } finally {
    valExpandedLoading.value = false
  }
}

async function triggerManualValuation() {
  valTriggerLoading.value = true
  emit('update:valSettingsMsg', '')
  emit('update:valSettingsError', false)
  try {
    await triggerValuation()
    emit('update:valSettingsMsg', 'Valuation started — progress updates below')
    setTimeout(() => { emit('update:valSettingsMsg', '') }, 10000)
    setTimeout(() => { loadValRuns() }, 2000)
  } catch {
    emit('update:valSettingsMsg', 'Failed to trigger valuation')
    emit('update:valSettingsError', true)
  } finally {
    valTriggerLoading.value = false
  }
}

async function cancelRun(runId: number) {
  try {
    await cancelValuationRun(runId)
    emit('update:valSettingsMsg', 'Cancellation requested')
    setTimeout(() => { emit('update:valSettingsMsg', '') }, 5000)
    setTimeout(() => { loadValRuns() }, 1000)
  } catch {
    emit('update:valSettingsMsg', 'Failed to cancel run')
    emit('update:valSettingsError', true)
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}

function formatDuration(ms: number) {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

onMounted(() => {
  loadValRuns()
})

onUnmounted(() => {
  if (valPollTimer) {
    clearInterval(valPollTimer)
    valPollTimer = null
  }
})
</script>

<style scoped>
.subsection-title {
  font-size: 1rem;
  font-weight: 600;
  margin-bottom: 1rem;
  color: var(--text-primary, #e0e0e0);
}

.section-divider {
  border: none;
  border-top: 1px solid var(--border-subtle, #333);
  margin: 1.5rem 0;
}

.avail-settings {
  margin-bottom: 1rem;
}

.avail-toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.avail-save-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 1rem;
}

.avail-save-msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
}

.avail-save-error {
  color: #e74c3c;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 42px;
  height: 22px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
  border-radius: 22px;
  transition: background 0.2s;
}

.toggle-slider::before {
  content: '';
  position: absolute;
  width: 16px;
  height: 16px;
  left: 2px;
  bottom: 2px;
  background: var(--text-secondary);
  border-radius: 50%;
  transition: transform 0.2s;
}

.toggle-switch input:checked + .toggle-slider {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
}

.toggle-switch input:checked + .toggle-slider::before {
  transform: translateX(20px);
  background: var(--accent-gold);
}

.avail-interval-input {
  max-width: 120px;
}

.avail-table {
  font-size: 0.82rem;
  table-layout: fixed;
  width: 100%;
}

.avail-row {
  cursor: pointer;
  transition: background var(--transition-fast);
}

.avail-row:hover {
  background: var(--bg-primary);
}

.avail-row-expanded {
  background: var(--bg-primary);
}

.avail-count-available { color: #2ecc71; font-weight: 600; }
.avail-count-unavailable { color: #e74c3c; font-weight: 600; }
.avail-count-unknown { color: #f1c40f; font-weight: 600; }

.avail-detail-row td {
  padding: 0.5rem;
  background: var(--bg-body);
  overflow: hidden;
}

.avail-detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.78rem;
  table-layout: fixed;
}

.avail-detail-table th,
.avail-detail-table td {
  padding: 0.4rem 0.5rem;
  text-align: left;
  border-bottom: 1px solid var(--border-subtle);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.avail-detail-table th {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
}

.avail-reason {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.listing-status-badge {
  display: inline-block;
  padding: 0.15rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}

.avail-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-top: 1rem;
}

.avail-page-info {
  font-size: 0.82rem;
  color: var(--text-secondary);
}

.date-cell {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.logs-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-family: 'Inter', sans-serif;
}

/* Valuation-specific */
.val-status-badge {
  display: inline-block;
  padding: 0.15rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}

.val-status-running {
  background: rgba(52, 152, 219, 0.15);
  color: #3498db;
}

.val-progress {
  margin-left: 0.35rem;
  font-size: 0.7rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.val-status-completed {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.val-status-failed {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.val-status-cancelled {
  background: rgba(243, 156, 18, 0.15);
  color: #f39c12;
}

.btn-cancel-run {
  margin-left: 0.4rem;
  padding: 0.1rem 0.4rem;
  font-size: 0.65rem;
  border: 1px solid rgba(231, 76, 60, 0.4);
  border-radius: var(--radius-full);
  background: transparent;
  color: #e74c3c;
  cursor: pointer;
  vertical-align: middle;
}
.btn-cancel-run:hover {
  background: rgba(231, 76, 60, 0.15);
}

.val-value {
  font-weight: 600;
  color: var(--accent-gold);
}

.val-confidence {
  display: inline-block;
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 600;
}

.val-conf-high {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.val-conf-medium {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
}

.val-conf-low {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.val-result-success {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.val-result-skipped {
  background: rgba(149, 165, 166, 0.15);
  color: #95a5a6;
}

.val-result-error {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.val-detail-table th:nth-child(1),
.val-detail-table td:nth-child(1) { width: 22%; }
.val-detail-table th:nth-child(2),
.val-detail-table td:nth-child(2) { width: 12%; }
.val-detail-table th:nth-child(3),
.val-detail-table td:nth-child(3) { width: 12%; }
.val-detail-table th:nth-child(4),
.val-detail-table td:nth-child(4) { width: 10%; }
.val-detail-table th:nth-child(5),
.val-detail-table td:nth-child(5) { width: 10%; }
.val-detail-table th:nth-child(6),
.val-detail-table td:nth-child(6) { width: 34%; }
</style>
