<template>
  <!-- Wishlist Availability Check -->
  <h3 class="mb-4 text-base font-semibold text-text-primary">Wishlist Availability Check</h3>
  <p class="mb-4 text-base text-text-secondary">Monitors dealer sites for coins on your wishlist and sends alerts when availability changes.</p>
  <div class="mb-4">
    <div class="form-group flex items-center justify-between gap-3">
      <label class="form-label">Enable Automatic Checks</label>
      <label class="relative inline-block h-[22px] w-[42px]">
        <input
          class="peer sr-only" type="checkbox"
          :checked="settings.WishlistCheckEnabled === 'true'"
          @change="settings.WishlistCheckEnabled = ($event.target as HTMLInputElement).checked ? 'true' : 'false'"
        />
        <span class="absolute inset-0 rounded-full border border-border-subtle bg-surface transition-colors after:absolute after:bottom-[2px] after:left-[2px] after:h-4 after:w-4 after:rounded-full after:bg-[var(--text-secondary)] after:transition-transform peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-checked:after:translate-x-5 peer-checked:after:bg-gold peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2"></span>
      </label>
    </div>
    <div class="form-group">
      <label class="form-label">Start Time (daily anchor)</label>
      <input
        v-model="settings.WishlistCheckStartTime"
        class="form-input w-full max-w-[120px]"
        type="time"
      />
      <span class="form-hint">The first check runs at this time each day. Subsequent checks repeat at the interval below.</span>
    </div>
    <div class="form-group">
      <label class="form-label">Repeat Interval (minutes)</label>
      <input
        v-model="settings.WishlistCheckInterval"
        class="form-input w-full max-w-[120px]"
        type="number"
        min="5"
        step="5"
      />
      <span class="form-hint">How often to repeat after the start time (e.g. 120 = every 2 hours).</span>
    </div>
    <div class="mt-4 flex w-full flex-col gap-3 md:flex-row md:items-center">
      <button class="btn btn-primary btn-sm" :disabled="settingsSaving" @click="emit('save')">
        {{ settingsSaving ? 'Saving...' : 'Save Schedule Settings' }}
      </button>
      <span v-if="settingsMsg" class="text-body text-gold md:mr-auto" :class="settingsError ? 'text-[var(--color-negative)]' : ''">{{ settingsMsg }}</span>
      <button class="btn btn-secondary btn-sm md:ml-auto" :disabled="triggerLoading" @click="triggerManualAvailabilityCheck()">
        {{ triggerLoading ? 'Queuing...' : 'Run Now' }}
      </button>
    </div>
  </div>

  <slot name="additional-settings" />

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h3 class="mb-4 text-base font-semibold text-text-primary">Availability Run History</h3>

  <div v-if="cyclesLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="cycles.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No availability cycles recorded yet.</div>
  <template v-else>
    <div class="overflow-x-auto">
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th>Status</th>
            <th>Total</th>
            <th class="hidden md:table-cell">Queued</th>
            <th class="hidden md:table-cell">Running</th>
            <th>Done</th>
            <th>Failed</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="cycle in cycles" :key="cycle.id">
            <tr class="cursor-pointer transition-colors hover:bg-surface" :class="{ 'bg-surface': expandedCycleId === cycle.id }" @click="toggleCycleDetail(cycle.id)">
              <td class="text-body text-text-secondary">{{ formatDate(cycle.startedAt) }}</td>
              <td class="hidden md:table-cell">{{ cycle.triggerType }}</td>
              <td>
                <span class="chip-sm" :class="statusClass(cycle.status)">{{ cycle.status === 'completed' ? 'done' : cycle.status }}</span>
              </td>
              <td>{{ cycle.totalChildren }}</td>
              <td class="hidden md:table-cell">{{ cycle.queuedChildren }}</td>
              <td class="hidden md:table-cell">{{ cycle.runningChildren }}</td>
              <td class="font-semibold text-[var(--color-positive)]">{{ cycle.completedChildren }}</td>
              <td class="font-semibold text-[var(--color-negative)]">{{ cycle.failedChildren }}</td>
            </tr>
            <tr v-if="expandedCycleId === cycle.id" class="bg-surface-secondary">
              <td :colspan="cyclesColspan">
                <div v-if="expandedChildrenLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
                <div v-else-if="expandedChildren.length" class="overflow-x-auto">
                  <table class="w-full border-collapse text-[0.78rem] md:table-fixed [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-[0.4rem] [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-[0.4rem] [&_td]:overflow-hidden [&_td]:text-ellipsis [&_td]:whitespace-nowrap">
                    <thead>
                      <tr>
                        <th>User</th>
                        <th>Status</th>
                        <th>Checked</th>
                        <th>Avail</th>
                        <th>Unavail</th>
                        <th>Unknown</th>
                        <th>Errors</th>
                        <th>Failure</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="child in expandedChildren" :key="child.id">
                        <td>{{ child.userName ?? `User #${child.userId}` }}</td>
                        <td><span class="chip-sm" :class="statusClass(child.status)">{{ child.status === 'completed' ? 'done' : child.status }}</span></td>
                        <td>{{ child.coinsChecked }}</td>
                        <td>{{ child.available }}</td>
                        <td>{{ child.unavailable }}</td>
                        <td>{{ child.unknown }}</td>
                        <td>{{ child.errors }}</td>
                        <td class="max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">{{ child.failMessage || '--' }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <p v-else class="px-8 py-8 text-center font-sans text-text-muted">No child runs for this cycle.</p>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="cyclesPage <= 1" @click="prevCyclesPage()">Prev</button>
      <span class="text-[0.82rem] text-text-secondary">Page {{ cyclesPage }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="cycles.length < 5" @click="nextCyclesPage()">Next</button>
    </div>
  </template>

  <hr class="my-6 border-0 border-t border-border-subtle" />
  <h4 class="section-label mb-4 flex items-center gap-2">
    Legacy Runs
    <span class="chip-sm">Legacy</span>
  </h4>

  <div v-if="legacyLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
  <div v-else-if="legacyRuns.length === 0" class="px-8 py-8 text-center font-sans text-text-muted">No legacy availability runs recorded.</div>
  <template v-else>
    <div class="overflow-x-auto">
      <table class="w-full border-collapse text-[0.8rem] md:table-fixed md:text-[0.82rem] [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-[0.35rem] [&_th]:py-2 [&_th]:text-left [&_th]:text-sm [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted md:[&_th]:px-2 md:[&_th]:py-3 [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-[0.35rem] [&_td]:py-2 [&_td]:text-left md:[&_td]:px-2 md:[&_td]:py-3">
        <thead>
          <tr>
            <th>Date</th>
            <th class="hidden md:table-cell">Trigger</th>
            <th class="hidden md:table-cell">Status</th>
            <th>Checked</th>
            <th class="hidden md:table-cell">Avail</th>
            <th>Unavail</th>
            <th class="hidden md:table-cell">Unknown</th>
            <th class="hidden md:table-cell">Errors</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="run in legacyRuns" :key="run.id">
            <tr class="cursor-pointer transition-colors hover:bg-surface" :class="{ 'bg-surface': expandedRunId === run.id }" @click="toggleRunDetail(run.id)">
              <td class="text-body text-text-secondary">
                {{ formatDate(run.startedAt) }}
                <span class="chip-sm ml-2">Legacy</span>
              </td>
              <td class="hidden md:table-cell">{{ run.triggerType }}</td>
              <td class="hidden md:table-cell">
                <span class="chip-sm" :class="statusClass(run.status)">{{ run.status === 'completed' ? 'done' : run.status }}</span>
              </td>
              <td>{{ run.coinsChecked }}</td>
              <td class="hidden font-semibold text-[var(--color-positive)] md:table-cell">{{ run.available }}</td>
              <td class="font-semibold text-[var(--color-negative)]">{{ run.unavailable }}</td>
              <td class="hidden font-semibold text-warning md:table-cell">{{ run.unknown }}</td>
              <td class="hidden md:table-cell">{{ run.errors }}</td>
              <td>{{ formatDuration(run.durationMs) }}</td>
            </tr>
            <tr v-if="expandedRunId === run.id && expandedResults" class="bg-surface-secondary">
              <td :colspan="legacyColspan">
                <div v-if="expandedLoading" class="flex justify-center py-8"><div class="spinner"></div></div>
                <div v-else-if="expandedResults.length" class="overflow-x-auto">
                  <table class="w-full border-collapse text-[0.78rem] md:table-fixed [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-[0.4rem] [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-[0.4rem] [&_td]:overflow-hidden [&_td]:text-ellipsis [&_td]:whitespace-nowrap">
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
                      <tr v-for="result in expandedResults" :key="result.id">
                        <td>{{ result.coinName }}</td>
                        <td>
                          <SafeExternalLink
                            v-if="safeRunUrl(result.url)"
                            :href="result.url"
                            target="_blank"
                            rel="noopener"
                            class="text-gold no-underline hover:underline focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                            @click.stop
                          >
                            {{ truncateUrl(result.url) }}
                          </SafeExternalLink>
                          <span v-else class="text-text-muted">--</span>
                        </td>
                        <td>
                          <span class="inline-block rounded-full px-2 py-[0.15rem] text-label font-semibold" :class="result.status === 'available' ? 'bg-[rgba(46,204,113,0.15)] text-[var(--color-positive)]' : result.status === 'unavailable' ? 'bg-[rgba(231,76,60,0.15)] text-[var(--color-negative)]' : 'bg-[rgba(241,196,15,0.15)] text-warning'">{{ result.status }}</span>
                        </td>
                        <td class="max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">{{ result.reason || '--' }}</td>
                        <td>{{ result.httpStatus ?? '--' }}</td>
                        <td>{{ result.agentUsed ? 'Yes' : 'No' }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
                <p v-else class="px-8 py-8 text-center font-sans text-text-muted">No results for this run.</p>
              </td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <div class="mt-4 flex items-center justify-center gap-3">
      <button class="btn btn-secondary btn-sm" :disabled="legacyPage <= 1" @click="prevLegacyPage()">Prev</button>
      <span class="text-[0.82rem] text-text-secondary">Page {{ legacyPage }}</span>
      <button class="btn btn-secondary btn-sm" :disabled="legacyRuns.length < 5" @click="nextLegacyPage()">Next</button>
    </div>
  </template>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  getAvailabilityCycleDetail,
  getAvailabilityCycles,
  getAvailabilityRunDetail,
  getAvailabilityRuns,
  triggerAvailabilityCheck,
} from '@/api/client'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'
import type { AppSettings, AvailabilityCycle, AvailabilityRun } from '@/types'

defineProps<{
  settings: AppSettings
  settingsSaving: boolean
  settingsMsg: string
  settingsError: boolean
}>()

const emit = defineEmits<{
  save: []
  'update:settingsMsg': [val: string]
  'update:settingsError': [val: boolean]
}>()

const isMobile = ref(window.innerWidth <= 600)
const cyclesColspan = computed(() => isMobile.value ? 5 : 8)
const legacyColspan = computed(() => isMobile.value ? 4 : 9)
const triggerLoading = ref(false)
const timers: ReturnType<typeof setTimeout>[] = []
let pollTimer: ReturnType<typeof setInterval> | null = null
const CYCLES_LIMIT = 5
const LEGACY_LIMIT = 5

// Parent availability cycles (new Feature 353 roll-up rows)
const cycles = ref<AvailabilityCycle[]>([])
const cyclesPage = ref(1)
const cyclesLoading = ref(false)
const expandedCycleId = ref<number | null>(null)
const expandedChildren = ref<AvailabilityRun[]>([])
const expandedChildrenLoading = ref(false)

// Legacy UserID=0 aggregate rows, retained read-only from /admin/availability-runs
const legacyRuns = ref<AvailabilityRun[]>([])
const legacyPage = ref(1)
const legacyLoading = ref(false)
const expandedRunId = ref<number | null>(null)
const expandedResults = ref<AvailabilityRun['results']>(undefined)
const expandedLoading = ref(false)

function safeRunUrl(url: string | null | undefined): string | null {
  return sanitizeExternalUrl(url)
}

function onResize() {
  isMobile.value = window.innerWidth <= 600
}

function statusClass(status: string): string {
  if (status === 'queued') return 'text-gold'
  if (status === 'running') return 'text-[var(--accent-bronze)]'
  if (status === 'failed' || status === 'partial_failure') return 'text-[var(--color-negative)]'
  return 'text-[var(--color-positive)]'
}

async function loadCycles() {
  cyclesLoading.value = true
  try {
    const res = await getAvailabilityCycles(cyclesPage.value, CYCLES_LIMIT)
    cycles.value = res.data?.cycles ?? []
  } catch {
    cycles.value = []
  } finally {
    cyclesLoading.value = false
  }
}

async function prevCyclesPage() {
  cyclesPage.value = Math.max(1, cyclesPage.value - 1)
  await loadCycles()
}

async function nextCyclesPage() {
  cyclesPage.value++
  await loadCycles()
}

async function toggleCycleDetail(cycleId: number) {
  if (expandedCycleId.value === cycleId) {
    expandedCycleId.value = null
    expandedChildren.value = []
    return
  }
  expandedCycleId.value = cycleId
  expandedChildren.value = []
  expandedChildrenLoading.value = true
  try {
    const res = await getAvailabilityCycleDetail(cycleId)
    expandedChildren.value = res.data?.children ?? []
  } catch {
    expandedChildren.value = []
  } finally {
    expandedChildrenLoading.value = false
  }
}

async function loadLegacyRuns() {
  legacyLoading.value = true
  try {
    const res = await getAvailabilityRuns(legacyPage.value, LEGACY_LIMIT)
    legacyRuns.value = (res.data?.runs ?? []).filter((run) => run.userId === 0)
  } catch {
    legacyRuns.value = []
  } finally {
    legacyLoading.value = false
  }
}

async function prevLegacyPage() {
  legacyPage.value = Math.max(1, legacyPage.value - 1)
  await loadLegacyRuns()
}

async function nextLegacyPage() {
  legacyPage.value++
  await loadLegacyRuns()
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

async function loadHistoryWithPoll() {
  try {
    await Promise.all([loadCycles(), loadLegacyRuns()])
    const hasActive = cycles.value.some(cycle => cycle.status === 'queued' || cycle.status === 'running')
    if (hasActive && !pollTimer) {
      pollTimer = setInterval(() => { loadHistoryWithPoll() }, 4000)
    } else if (!hasActive && pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  } catch { /* ignore */ }
}

async function triggerManualAvailabilityCheck() {
  triggerLoading.value = true
  emit('update:settingsMsg', '')
  emit('update:settingsError', false)
  try {
    const res = await triggerAvailabilityCheck()
    emit('update:settingsMsg', `Cycle #${res.data.cycleId} queued — history updates below`)
    timers.push(setTimeout(() => { emit('update:settingsMsg', '') }, 12000))
    timers.push(setTimeout(() => { loadHistoryWithPoll() }, 1000))
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } })?.response?.status
    emit('update:settingsMsg', status === 409
      ? 'A manual availability run is already in progress'
      : 'Failed to queue availability check')
    emit('update:settingsError', true)
  } finally {
    triggerLoading.value = false
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
    const parsed = new URL(url)
    const path = parsed.pathname.length > 20 ? parsed.pathname.substring(0, 17) + '...' : parsed.pathname
    return parsed.hostname + path
  } catch {
    if (url.length <= 35) return url
    return url.substring(0, 32) + '...'
  }
}

onMounted(() => {
  window.addEventListener('resize', onResize)
  loadHistoryWithPoll()
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  if (pollTimer) clearInterval(pollTimer)
  timers.forEach(clearTimeout)
})
</script>
