<template>
  <div class="admin-security-section">
    <section class="admin-section card">
      <div class="section-heading">
        <div>
          <p class="section-label">Security</p>
          <h2>Security Overview</h2>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadAll">
          {{ loading ? 'Loading...' : 'Refresh' }}
        </button>
      </div>

      <div v-if="error" class="security-alert error" role="alert">
        <AlertTriangle :size="18" />
        <span>{{ error }}</span>
      </div>

      <div class="summary-grid">
        <div v-for="card in summaryCards" :key="card.label" class="summary-card">
          <span class="summary-label">{{ card.label }}</span>
          <span class="summary-value">{{ card.value }}</span>
        </div>
      </div>
    </section>

    <section class="admin-section card">
      <div class="section-heading">
        <div>
          <p class="section-label">Exposure Check</p>
          <h2>Public-Facing Readiness</h2>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="exposureLoading" @click="loadExposure">
          {{ exposureLoading ? 'Checking...' : 'Run Check' }}
        </button>
      </div>

      <div v-if="exposure" class="exposure-list">
        <p v-if="exposure.publicIp" class="ip-row">
          API sees your IP as <strong>{{ exposure.publicIp }}</strong>
        </p>
        <div
          v-for="check in exposureChecks"
          :key="check.label"
          class="exposure-row"
          :class="{ warning: !check.ok }"
        >
          <span class="chip-sm">{{ check.ok ? 'OK' : 'Review' }}</span>
          <div>
            <strong>{{ check.label }}</strong>
            <p>{{ check.message }}</p>
          </div>
        </div>
        <div v-if="exposure.warnings?.length" class="security-alert warning">
          <AlertTriangle :size="18" />
          <ul>
            <li v-for="warning in exposure.warnings" :key="warning">{{ warning }}</li>
          </ul>
        </div>
      </div>
      <p v-else class="empty-copy">Run the exposure check to validate beta deployment settings.</p>
    </section>

    <section class="admin-section card">
      <div class="section-heading">
        <div>
          <p class="section-label">Events</p>
          <h2>Security Events</h2>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="eventsLoading" @click="loadEvents">
          {{ eventsLoading ? 'Loading...' : 'Apply Filters' }}
        </button>
      </div>

      <form class="filters-grid" @submit.prevent="loadEvents">
        <input v-model="filters.type" class="form-input" placeholder="Type" />
        <select v-model="filters.outcome" class="form-select">
          <option value="">All outcomes</option>
          <option value="success">Success</option>
          <option value="failure">Failure</option>
          <option value="blocked">Blocked</option>
        </select>
        <select v-model="filters.severity" class="form-select">
          <option value="">All severities</option>
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="critical">Critical</option>
        </select>
        <input v-model="filters.username" class="form-input" placeholder="User" autocomplete="off" />
        <input v-model="filters.ip" class="form-input" placeholder="IP" autocomplete="off" />
        <input v-model="filters.since" class="form-input" type="date" aria-label="Since date" />
        <select v-model.number="filters.limit" class="form-select">
          <option :value="25">25</option>
          <option :value="50">50</option>
          <option :value="100">100</option>
          <option :value="250">250</option>
        </select>
        <button class="btn btn-primary btn-sm" type="submit" :disabled="eventsLoading">Filter</button>
      </form>

      <div class="table-wrap">
        <table class="security-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Type</th>
              <th>Severity</th>
              <th>Outcome</th>
              <th>User</th>
              <th>IP</th>
              <th>Message</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="events.length === 0">
              <td colspan="7" class="empty-cell">No security events match the current filters.</td>
            </tr>
            <tr v-for="event in events" :key="event.id">
              <td class="date-cell">{{ formatDateTime(event.timestamp) }}</td>
              <td>{{ event.type }}</td>
              <td><span class="chip-sm">{{ event.severity }}</span></td>
              <td>{{ event.outcome ?? '—' }}</td>
              <td>{{ event.username ?? '—' }}</td>
              <td>{{ event.ip ?? '—' }}</td>
              <td>{{ event.message ?? '—' }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="admin-section card">
      <div class="section-heading">
        <div>
          <p class="section-label">IP Rules</p>
          <h2>Manual Bans</h2>
        </div>
      </div>

      <form class="ban-form" @submit.prevent="submitBan">
        <input v-model="newRule.cidr" class="form-input" required placeholder="CIDR or IP, e.g. 203.0.113.0/24" />
        <input v-model="newRule.duration" class="form-input" placeholder="Duration, e.g. 24h or 7d" />
        <input v-model="newRule.reason" class="form-input" required placeholder="Reason" />
        <button class="btn btn-primary btn-sm" type="submit" :disabled="banSaving">
          {{ banSaving ? 'Adding...' : 'Add Ban' }}
        </button>
      </form>

      <div class="table-wrap">
        <table class="security-table">
          <thead>
            <tr>
              <th>CIDR</th>
              <th>Reason</th>
              <th>Expires</th>
              <th>Created By</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="ipRules.length === 0">
              <td colspan="5" class="empty-cell">No active manual IP bans.</td>
            </tr>
            <tr v-for="rule in ipRules" :key="rule.id">
              <td>{{ rule.cidr }}</td>
              <td>{{ rule.reason }}</td>
              <td>{{ rule.expiresAt ? formatDateTime(rule.expiresAt) : 'Never' }}</td>
              <td>{{ rule.createdBy ?? '—' }}</td>
              <td>
                <button class="btn btn-danger btn-xs" :disabled="deletingRuleId === rule.id" @click="removeRule(rule.id)">
                  {{ deletingRuleId === rule.id ? 'Deleting...' : 'Delete' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <section class="admin-section card">
      <div class="section-heading">
        <div>
          <p class="section-label">Lockouts</p>
          <h2>Locked Users</h2>
        </div>
      </div>
      <div v-if="lockedUsers.length === 0" class="empty-copy">
        No locked accounts reported by the current user list.
      </div>
      <div v-else class="locked-list">
        <div v-for="user in lockedUsers" :key="user.id" class="locked-row">
          <div>
            <strong>{{ user.username }}</strong>
            <p>Locked until {{ formatDateTime(user.lockedUntil ?? '') }}</p>
          </div>
          <button class="btn btn-secondary btn-sm" :disabled="unlockingUserId === user.id" @click="unlock(user.id)">
            {{ unlockingUserId === user.id ? 'Unlocking...' : 'Unlock' }}
          </button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { AlertTriangle } from 'lucide-vue-next'
import {
  createSecurityIpRule,
  deleteSecurityIpRule,
  getSecurityEvents,
  getSecurityExposureCheck,
  getSecurityIpRules,
  getSecuritySummary,
  unlockUser,
} from '@/api/client'
import type { SecurityEvent, SecurityEventFilters, SecurityExposureCheck, SecurityIpRule, SecuritySummary, UserInfo } from '@/types'

const props = defineProps<{
  users: UserInfo[]
  registrationMode?: string
}>()

const emit = defineEmits<{
  unlocked: [userId: number]
}>()

const loading = ref(false)
const eventsLoading = ref(false)
const exposureLoading = ref(false)
const error = ref('')
const summary = ref<SecuritySummary>({
  failedLogins: 0,
  lockedAccounts: 0,
  activeBans: 0,
  recentEvents: 0,
})
const events = ref<SecurityEvent[]>([])
const ipRules = ref<SecurityIpRule[]>([])
const exposure = ref<SecurityExposureCheck | null>(null)
const deletingRuleId = ref<number | null>(null)
const unlockingUserId = ref<number | null>(null)
const banSaving = ref(false)

const filters = reactive<SecurityEventFilters>({
  type: '',
  severity: '',
  username: '',
  ip: '',
  outcome: '',
  since: '',
  limit: 50,
})

const newRule = reactive({
  cidr: '',
  duration: '',
  reason: '',
})

const summaryCards = computed(() => [
  { label: 'Failed Logins', value: summary.value.failedLogins },
  { label: 'Locked Accounts', value: summary.value.lockedAccounts },
  { label: 'Active Bans', value: summary.value.activeBans },
  { label: 'Recent Security Events', value: summary.value.recentEvents },
])

const lockedUsers = computed(() =>
  props.users.filter((user) => {
    if (!user.lockedUntil) return false
    return new Date(user.lockedUntil).getTime() > Date.now()
  }),
)

const exposureChecks = computed(() => {
  const current = exposure.value
  if (!current) return []
  const config = current.config
  const registrationMode = config?.registrationMode ?? props.registrationMode
  return [
    buildExposureRow('Proxy Headers', current.proxy ?? config?.trustedProxiesConfigured ?? !hasWarning(current, 'Trusted proxies'), current.proxyWarning, 'Proxy headers look constrained.'),
    buildExposureRow('CORS', current.cors ?? !hasWarning(current, 'CORS'), current.corsWarning, 'CORS is not reporting broad origins.'),
    buildExposureRow('WebAuthn', current.webAuthn ?? !hasWarning(current, 'WebAuthn'), current.webAuthnWarning, 'WebAuthn relying-party settings are configured.'),
    buildExposureRow('Public App URL', current.publicAppUrl ?? current.publicAppURL ?? Boolean(config?.publicAppURL && !hasWarning(current, 'PublicAppURL')), current.publicAppUrlWarning, 'Public app URL is configured.'),
    buildExposureRow('Registration', current.registration ?? (registrationMode ? registrationMode === 'closed' : !hasWarning(current, 'Registration')), current.registrationWarning, `Registration mode: ${registrationMode || 'backend default'}.`),
    buildExposureRow('Agent Token', current.agentToken ?? config?.agentInternalTokenSet, current.agentTokenWarning, 'Agent token exposure check passed.'),
  ]
})

function buildExposureRow(label: string, ok: boolean | undefined, warning: string | undefined, fallback: string) {
  return {
    label,
    ok: ok !== false,
    message: warning || fallback,
  }
}

function hasWarning(current: SecurityExposureCheck, token: string) {
  return (current.warnings ?? []).some((warning) => warning.toLowerCase().includes(token.toLowerCase()))
}

onMounted(() => {
  void loadAll()
})

async function loadAll() {
  loading.value = true
  error.value = ''
  try {
    await Promise.all([loadSummary(), loadEvents(), loadIpRules(), loadExposure()])
  } catch {
    error.value = 'Failed to load security data'
  } finally {
    loading.value = false
  }
}

async function loadSummary() {
  const res = await getSecuritySummary()
  summary.value = normalizeSummary(res.data)
}

async function loadEvents() {
  eventsLoading.value = true
  try {
    const params = compactFilters(filters)
    const res = await getSecurityEvents(params)
    const rawEvents = Array.isArray(res.data) ? res.data : (res.data.events ?? [])
    events.value = rawEvents.map(normalizeEvent)
  } finally {
    eventsLoading.value = false
  }
}

async function loadIpRules() {
  const res = await getSecurityIpRules()
  ipRules.value = Array.isArray(res.data) ? res.data : (res.data.rules ?? res.data.ipRules ?? [])
}

async function loadExposure() {
  exposureLoading.value = true
  try {
    const res = await getSecurityExposureCheck()
    exposure.value = res.data
  } finally {
    exposureLoading.value = false
  }
}

async function submitBan() {
  banSaving.value = true
  error.value = ''
  try {
    await createSecurityIpRule({
      cidr: newRule.cidr.trim(),
      durationMinutes: parseDurationMinutes(newRule.duration),
      reason: newRule.reason.trim(),
    })
    newRule.cidr = ''
    newRule.duration = ''
    newRule.reason = ''
    await Promise.all([loadIpRules(), loadSummary()])
  } catch {
    error.value = 'Failed to add IP ban'
  } finally {
    banSaving.value = false
  }
}

async function removeRule(id: number) {
  deletingRuleId.value = id
  error.value = ''
  try {
    await deleteSecurityIpRule(id)
    await Promise.all([loadIpRules(), loadSummary()])
  } catch {
    error.value = 'Failed to delete IP ban'
  } finally {
    deletingRuleId.value = null
  }
}

async function unlock(userId: number) {
  unlockingUserId.value = userId
  error.value = ''
  try {
    await unlockUser(userId)
    emit('unlocked', userId)
    await loadSummary()
  } catch {
    error.value = 'Failed to unlock user'
  } finally {
    unlockingUserId.value = null
  }
}

function compactFilters(source: SecurityEventFilters): SecurityEventFilters {
  const params: SecurityEventFilters = {}
  if (source.type) params.type = source.type
  if (source.severity) params.severity = source.severity
  if (source.username) params.username = source.username
  if (source.ip) params.clientIp = source.ip
  if (source.outcome) params.outcome = source.outcome
  if (source.since) params.since = new Date(`${source.since}T00:00:00`).toISOString()
  if (source.limit) params.limit = source.limit
  return params
}

function normalizeSummary(data: SecuritySummary | Record<string, unknown>): SecuritySummary {
  const root = data as Record<string, unknown>
  const raw = (typeof root.summary === 'object' && root.summary !== null ? root.summary : root) as Record<string, unknown>
  return {
    failedLogins: asNumber(raw.failedLogins ?? raw.loginFailures ?? raw.failed_login_count ?? raw.failedLoginCount),
    lockedAccounts: asNumber(raw.lockedAccounts ?? raw.locked_account_count ?? raw.lockedAccountCount),
    activeBans: asNumber(raw.activeBans ?? raw.activeIpRuleCount ?? raw.active_bans ?? raw.activeBanCount),
    recentEvents: asNumber(raw.recentEvents ?? raw.recent_security_events ?? raw.recentEventCount),
  }
}

function normalizeEvent(event: SecurityEvent): SecurityEvent {
  return {
    ...event,
    timestamp: event.timestamp ?? event.createdAt ?? '',
    ip: event.ip ?? event.clientIp ?? null,
    severity: event.severity ?? '—',
  }
}

function parseDurationMinutes(duration: string) {
  const value = duration.trim()
  if (!value) return undefined
  const match = value.match(/^(\d+)\s*([mhdw])?$/i)
  if (!match) return undefined
  const amount = Number(match[1] ?? 0)
  const unit = (match[2] ?? 'm').toLowerCase()
  const multipliers: Record<string, number> = { m: 1, h: 60, d: 1440, w: 10080 }
  return amount * (multipliers[unit] ?? 1)
}

function asNumber(value: unknown) {
  return typeof value === 'number' ? value : Number(value ?? 0)
}

function formatDateTime(value: string) {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
</script>

<style scoped>
.admin-security-section {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.section-heading h2 {
  margin: 0;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 0.75rem;
}

.summary-card {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.summary-label {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.summary-value {
  color: var(--accent-gold);
  font-family: 'Cinzel', serif;
  font-size: 1.5rem;
  font-weight: 600;
}

.security-alert {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.75rem;
  margin-bottom: 1rem;
  background: var(--accent-gold-glow);
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-sm);
  color: var(--text-warning);
  font-size: 0.85rem;
}

.security-alert.error {
  color: var(--color-negative);
}

.security-alert ul {
  margin: 0;
  padding-left: 1rem;
}

.exposure-list,
.locked-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.ip-row {
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.ip-row strong {
  color: var(--accent-gold);
}

.exposure-row,
.locked-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.exposure-row.warning {
  border-color: var(--border-accent);
}

.exposure-row p,
.locked-row p {
  margin: 0.25rem 0 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.filters-grid,
.ban-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(130px, 1fr));
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.table-wrap {
  overflow-x: auto;
}

.security-table {
  width: 100%;
  border-collapse: collapse;
}

.security-table th,
.security-table td {
  text-align: left;
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
  vertical-align: top;
}

.security-table th {
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}

.date-cell,
.empty-copy,
.empty-cell {
  color: var(--text-muted);
  font-size: 0.85rem;
}

.empty-cell {
  text-align: center;
  padding: 1.5rem;
}

@media (max-width: 640px) {
  .section-heading,
  .locked-row {
    flex-direction: column;
  }
}
</style>
