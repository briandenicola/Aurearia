<template>
  <div class="flex min-w-0 flex-col gap-4">
    <section class="admin-section card min-w-0">
      <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p class="section-label">Security</p>
          <h2 class="m-0 text-xl font-medium">Security Overview</h2>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="loading" @click="loadAll">
          {{ loading ? 'Loading...' : 'Refresh' }}
        </button>
      </div>

      <div
        v-if="error"
        class="mb-4 flex items-start gap-2 rounded-sm border border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.15)] p-3 text-body text-[var(--color-negative)]"
        role="alert"
      >
        <AlertTriangle :size="18" />
        <span>{{ error }}</span>
      </div>

      <div class="grid gap-3 [grid-template-columns:repeat(auto-fit,minmax(150px,1fr))]">
        <div v-for="card in summaryCards" :key="card.label" class="flex flex-col gap-[0.35rem] rounded-sm border border-border-subtle bg-input p-3">
          <span class="section-label">{{ card.label }}</span>
          <span class="font-['Cinzel'] text-xl font-semibold text-gold">{{ card.value }}</span>
        </div>
      </div>
    </section>

    <section class="admin-section card min-w-0">
      <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p class="section-label">Exposure Check</p>
          <h2 class="m-0 text-xl font-medium">Public-Facing Readiness</h2>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="exposureLoading" @click="loadExposure">
          {{ exposureLoading ? 'Checking...' : 'Run Check' }}
        </button>
      </div>

      <div v-if="exposure" class="flex flex-col gap-3">
        <p v-if="exposure.publicIp" class="m-0 text-base text-text-secondary">
          API sees your IP as <strong class="text-gold">{{ exposure.publicIp }}</strong>
        </p>
        <div
          v-for="check in exposureChecks"
          :key="check.label"
          class="grid items-start gap-3 rounded-sm border border-border-subtle bg-input p-3 [grid-template-columns:max-content_minmax(0,1fr)]"
          :class="!check.ok ? 'border-border-accent' : ''"
        >
          <span class="chip-sm inline-flex min-w-16 justify-center">{{ check.ok ? 'OK' : 'Review' }}</span>
          <div class="min-w-0">
            <strong class="block leading-[1.3]">{{ check.label }}</strong>
            <p class="mt-1 break-words text-base text-text-secondary">{{ check.message }}</p>
          </div>
        </div>
        <div
          v-if="exposure.warnings?.length"
          class="flex items-start gap-2 rounded-sm border border-border-accent bg-[var(--accent-gold-glow)] p-3 text-body text-warning"
        >
          <AlertTriangle :size="18" />
          <ul class="m-0 list-disc pl-4">
            <li v-for="warning in exposure.warnings" :key="warning">{{ warning }}</li>
          </ul>
        </div>
      </div>
      <p v-else class="text-body text-text-muted">Run the exposure check to validate beta deployment settings.</p>
    </section>

    <section class="admin-section card min-w-0">
      <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p class="section-label">Events</p>
          <h2 class="m-0 text-xl font-medium">Security Events</h2>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="eventsLoading" @click="loadEvents">
          {{ eventsLoading ? 'Loading...' : 'Apply Filters' }}
        </button>
      </div>

      <form class="mb-4 grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-4" @submit.prevent="loadEvents">
        <input v-model="filters.type" class="form-input min-w-0 w-full" placeholder="Type" />
        <select v-model="filters.outcome" class="form-select min-w-0 w-full">
          <option value="">All outcomes</option>
          <option value="success">Success</option>
          <option value="failure">Failure</option>
          <option value="blocked">Blocked</option>
        </select>
        <select v-model="filters.severity" class="form-select min-w-0 w-full">
          <option value="">All severities</option>
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="critical">Critical</option>
        </select>
        <input v-model="filters.username" class="form-input min-w-0 w-full" placeholder="User" autocomplete="off" />
        <input v-model="filters.ip" class="form-input min-w-0 w-full" placeholder="IP" autocomplete="off" />
        <input v-model="filters.since" class="form-input min-w-0 w-full appearance-none overflow-hidden" type="date" aria-label="Since date" />
        <select v-model.number="filters.limit" class="form-select min-w-0 w-full">
          <option :value="25">25</option>
          <option :value="50">50</option>
          <option :value="100">100</option>
          <option :value="250">250</option>
        </select>
        <button class="btn btn-primary btn-sm w-full justify-center" type="submit" :disabled="eventsLoading">Filter</button>
      </form>

      <div class="w-full max-w-full min-w-0 overflow-x-auto">
        <table class="w-full min-w-[44rem] table-fixed border-collapse [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-3 [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.08em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-3 [&_td]:align-top [&_td]:text-left">
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
              <td colspan="7" class="p-6 text-center text-body text-text-muted">No security events match the current filters.</td>
            </tr>
            <tr v-for="event in events" :key="event.id">
              <td class="w-32 max-w-32 break-words text-body text-text-muted">{{ formatDateTime(event.timestamp) }}</td>
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

    <section class="admin-section card min-w-0">
      <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p class="section-label">IP Rules</p>
          <h2 class="m-0 text-xl font-medium">Manual Bans</h2>
        </div>
      </div>

      <form class="mb-4 grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-4" @submit.prevent="submitBan">
        <input v-model="newRule.cidr" class="form-input min-w-0 w-full" required placeholder="CIDR or IP, e.g. 203.0.113.0/24" />
        <input v-model="newRule.duration" class="form-input min-w-0 w-full" placeholder="Duration, e.g. 24h or 7d" />
        <input v-model="newRule.reason" class="form-input min-w-0 w-full" required placeholder="Reason" />
        <button class="btn btn-primary btn-sm w-full justify-center" type="submit" :disabled="banSaving">
          {{ banSaving ? 'Adding...' : 'Add Ban' }}
        </button>
      </form>

      <div class="w-full max-w-full min-w-0 overflow-x-auto">
        <table class="w-full min-w-[44rem] table-fixed border-collapse [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-3 [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.08em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-3 [&_td]:align-top [&_td]:text-left">
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
              <td colspan="5" class="p-6 text-center text-body text-text-muted">No active manual IP bans.</td>
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

    <section class="admin-section card min-w-0">
      <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <p class="section-label">Lockouts</p>
          <h2 class="m-0 text-xl font-medium">Locked Users</h2>
        </div>
      </div>
      <div v-if="lockedUsers.length === 0" class="text-body text-text-muted">
        No locked accounts reported by the current user list.
      </div>
      <div v-else class="flex flex-col gap-3">
        <div
          v-for="user in lockedUsers"
          :key="user.id"
          class="flex flex-col gap-3 rounded-sm border border-border-subtle bg-input p-3 sm:flex-row sm:items-start sm:justify-between"
        >
          <div>
            <strong>{{ user.username }}</strong>
            <p class="mt-1 break-words text-base text-text-secondary">Locked until {{ formatDateTime(user.lockedUntil ?? '') }}</p>
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

