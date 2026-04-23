<template>
  <div>
    <h3 class="subsection-title">Wishlist Availability Check</h3>
    <div class="avail-settings">
      <div class="form-group avail-toggle-row">
        <label class="form-label">Enable Automatic Checks</label>
        <label class="toggle-switch">
          <input
            type="checkbox"
            :checked="settings.WishlistCheckEnabled === 'true'"
            @change="settings.WishlistCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
          />
          <span class="toggle-slider"></span>
        </label>
      </div>
      <div class="form-group">
        <label class="form-label">Start Time (daily anchor)</label>
        <input
          v-model="settings.WishlistCheckStartTime"
          class="form-input avail-interval-input"
          type="time"
        />
        <span class="form-hint">The first check runs at this time each day. Subsequent checks repeat at the interval below.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Repeat Interval (minutes)</label>
        <input
          v-model="settings.WishlistCheckInterval"
          class="form-input avail-interval-input"
          type="number"
          min="5"
          step="5"
        />
        <span class="form-hint">How often to repeat after the start time (e.g. 120 = every 2 hours).</span>
      </div>
      <div class="avail-save-row">
        <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="$emit('save-settings')">
          {{ settingsSaving ? 'Saving...' : 'Save Schedule Settings' }}
        </button>
        <span v-if="availSettingsMsg" class="avail-save-msg" :class="{ 'avail-save-error': availSettingsError }">{{ availSettingsMsg }}</span>
      </div>
    </div>

    <hr class="section-divider" />
    <h3 class="subsection-title">Availability Run History</h3>

    <div v-if="availLoading" class="loading-overlay"><div class="spinner"></div></div>
    <div v-else-if="availRuns.length === 0" class="logs-empty">No availability runs recorded yet.</div>
    <template v-else>
      <table class="users-table avail-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Trigger</th>
            <th>Checked</th>
            <th>Avail</th>
            <th>Unavail</th>
            <th>Unknown</th>
            <th>Errors</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in availRuns" :key="run.id">
            <tr class="avail-row" :class="{ 'avail-row-expanded': expandedRunId === run.id }" @click="toggleRunDetail(run.id)">
              <td class="date-cell">{{ formatDate(run.startedAt) }}</td>
              <td>{{ run.triggerType }}</td>
              <td>{{ run.coinsChecked }}</td>
              <td class="avail-count-available">{{ run.available }}</td>
              <td class="avail-count-unavailable">{{ run.unavailable }}</td>
              <td class="avail-count-unknown">{{ run.unknown }}</td>
              <td>{{ run.errors }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="expandedRunId === run.id && expandedResults" class="avail-detail-row">
              <td colspan="8">
                <div v-if="expandedLoading" class="loading-overlay"><div class="spinner"></div></div>
                <table v-else-if="expandedResults.length" class="avail-detail-table">
                  <thead>
                    <tr>
                      <th>Coin</th>
                      <th>URL</th>
                      <th>Status</th>
                      <th>Reason</th>
                      <th>HTTP</th>
                      <th>Agent</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="r in expandedResults" :key="r.id">
                      <td>{{ r.coinName }}</td>
                      <td><a v-if="r.url" :href="r.url" target="_blank" rel="noopener" class="avail-link" @click.stop>{{ truncateUrl(r.url) }}</a><span v-else class="text-muted">--</span></td>
                      <td>
                        <span class="listing-status-badge" :class="'listing-' + r.status">{{ r.status }}</span>
                      </td>
                      <td class="avail-reason">{{ r.reason || '--' }}</td>
                      <td>{{ r.httpStatus ?? '--' }}</td>
                      <td>{{ r.agentUsed ? 'Yes' : 'No' }}</td>
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
        <button class="btn btn-secondary btn-sm" :disabled="availPage <= 1" @click="availPage--; loadAvailRuns()">Prev</button>
        <span class="avail-page-info">Page {{ availPage }}</span>
        <button class="btn btn-secondary btn-sm" :disabled="availRuns.length < 5" @click="availPage++; loadAvailRuns()">Next</button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAvailabilityRuns, getAvailabilityRunDetail } from '@/api/client'
import type { AppSettings, AvailabilityRun } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
  availSettingsMsg: string
  availSettingsError: boolean
}>()

defineEmits<{
  'save-settings': []
}>()

const availRuns = ref<AvailabilityRun[]>([])
const availTotal = ref(0)
const availPage = ref(1)
const availLoading = ref(false)
const expandedRunId = ref<number | null>(null)
const expandedResults = ref<AvailabilityRun['results']>(undefined)
const expandedLoading = ref(false)

async function loadAvailRuns() {
  availLoading.value = true
  try {
    const res = await getAvailabilityRuns(availPage.value, 5)
    availRuns.value = res.data.runs ?? []
    availTotal.value = res.data.total ?? 0
  } catch { /* ignore */ } finally {
    availLoading.value = false
  }
}

async function toggleRunDetail(runId: number) {
  if (expandedRunId.value === runId) {
    expandedRunId.value = null
    expandedResults.value = undefined
    return
  }
  expandedRunId.value = runId
  expandedResults.value = []
  expandedLoading.value = true
  try {
    const res = await getAvailabilityRunDetail(runId)
    expandedResults.value = res.data.results ?? []
  } catch {
    expandedResults.value = []
  } finally {
    expandedLoading.value = false
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}

function formatDuration(ms: number) {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function truncateUrl(url: string) {
  try {
    const u = new URL(url)
    const path = u.pathname.length > 20 ? u.pathname.substring(0, 17) + '...' : u.pathname
    return u.hostname + path
  } catch {
    if (url.length <= 35) return url
    return url.substring(0, 32) + '...'
  }
}

onMounted(() => {
  loadAvailRuns()
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

.avail-detail-table th:nth-child(1),
.avail-detail-table td:nth-child(1) { width: 22%; }
.avail-detail-table th:nth-child(2),
.avail-detail-table td:nth-child(2) { width: 22%; }
.avail-detail-table th:nth-child(3),
.avail-detail-table td:nth-child(3) { width: 10%; }
.avail-detail-table th:nth-child(4),
.avail-detail-table td:nth-child(4) { width: 28%; }
.avail-detail-table th:nth-child(5),
.avail-detail-table td:nth-child(5) { width: 8%; }
.avail-detail-table th:nth-child(6),
.avail-detail-table td:nth-child(6) { width: 10%; }

.avail-detail-table th {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  font-weight: 600;
}

.avail-link {
  color: var(--accent-gold);
  text-decoration: none;
  font-size: 0.75rem;
}

.avail-link:hover {
  text-decoration: underline;
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

.listing-available {
  background: rgba(46, 204, 113, 0.15);
  color: #2ecc71;
}

.listing-unavailable {
  background: rgba(231, 76, 60, 0.15);
  color: #e74c3c;
}

.listing-unknown {
  background: rgba(241, 196, 15, 0.15);
  color: #f1c40f;
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

.text-muted {
  color: var(--text-muted);
}

.logs-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-family: 'Inter', sans-serif;
}
</style>
