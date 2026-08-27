<template>
  <div class="container">
    <header class="page-header">
      <h1>Availability Run History</h1>
      <RouterLink class="btn btn-secondary btn-sm" to="/wishlist">
        <ArrowLeft :size="16" /> Back to Wishlist
      </RouterLink>
    </header>

    <div v-if="unauthorized" class="empty-state card">
      <h3>Sign in required</h3>
      <p>Please sign in to view your availability run history.</p>
    </div>

    <!-- Deep-linked single-run detail (e.g. from a notification) -->
    <template v-else-if="routeRunId !== null">
      <div v-if="detailLoading" class="loading-overlay"><div class="spinner"></div></div>
      <div v-else-if="detailError" class="empty-state card">
        <h3>Run not found</h3>
        <p>{{ detailError }}</p>
      </div>
      <section v-else-if="detailRun" class="card">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div>
            <span class="section-label mb-1 block">Run #{{ detailRun.id }}</span>
            <strong class="text-lg text-text-primary">{{ formatDate(detailRun.startedAt) }}</strong>
          </div>
          <span class="chip-sm" :class="statusClass(detailRun.status)">{{ detailRun.status === 'completed' ? 'done' : detailRun.status }}</span>
        </div>

        <div class="mb-4 flex flex-wrap items-center gap-4 text-body text-text-secondary">
          <span>{{ detailRun.coinsChecked }} checked</span>
          <span class="font-semibold text-[var(--color-positive)]">{{ detailRun.available }} available</span>
          <span class="font-semibold text-[var(--color-negative)]">{{ detailRun.unavailable }} unavailable</span>
          <span class="font-semibold text-warning">{{ detailRun.unknown }} unknown</span>
          <span v-if="detailRun.failMessage" class="text-[var(--color-negative)]">{{ detailRun.failMessage }}</span>
        </div>

        <p v-if="!detailRun.results || detailRun.results.length === 0" class="py-4 text-center text-text-muted">No results for this run.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full border-collapse text-body [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-2 [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-2 [&_td]:text-left">
            <thead>
              <tr>
                <th>Coin</th>
                <th>URL</th>
                <th>Status</th>
                <th>Reason</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="result in detailRun.results" :key="result.id">
                <td>{{ result.coinName }}</td>
                <td>
                  <SafeExternalLink v-if="safeResultUrl(result.url)" :href="result.url" target="_blank" rel="noopener" class="text-gold no-underline hover:underline">
                    {{ truncateUrl(result.url) }}
                  </SafeExternalLink>
                  <span v-else class="text-text-muted">--</span>
                </td>
                <td><span class="chip-sm" :class="resultStatusClass(result.status)">{{ result.status }}</span></td>
                <td class="max-w-[220px] overflow-hidden text-ellipsis whitespace-nowrap">{{ result.reason || '--' }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <RouterLink class="btn btn-secondary btn-sm mt-4" to="/wishlist/availability-runs">Back to all runs</RouterLink>
      </section>
    </template>

    <template v-else>
      <div v-if="loading" class="loading-overlay"><div class="spinner"></div></div>
      <div v-else-if="loadError" class="empty-state card">
        <h3>Unable to load run history</h3>
        <p>{{ loadError }}</p>
      </div>
      <div v-else-if="runs.length === 0" class="empty-state card">
        <h3>No availability runs yet</h3>
        <p>Runs appear here whenever your wishlist URLs are checked for availability.</p>
      </div>

      <template v-else>
        <div class="flex flex-col gap-2">
          <div
            v-for="run in runs"
            :key="run.id"
            class="card !p-0 cursor-pointer !px-4 !py-3 transition-colors hover:bg-card-hover"
            @click="toggleExpand(run.id)"
          >
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="min-w-0">
                <span class="section-label mb-0.5 block">{{ run.triggerType }}</span>
                <strong class="text-body text-text-primary">{{ formatDate(run.startedAt) }}</strong>
              </div>
              <div class="flex flex-wrap items-center gap-3 text-body text-text-secondary">
                <span>{{ run.coinsChecked }} checked</span>
                <span class="font-semibold text-[var(--color-negative)]">{{ run.unavailable }} unavail</span>
                <span class="chip-sm" :class="statusClass(run.status)">{{ run.status === 'completed' ? 'done' : run.status }}</span>
              </div>
            </div>

            <div v-if="expandedRunId === run.id" class="mt-3 border-t border-border-subtle pt-3" @click.stop>
              <div v-if="expandedLoading" class="flex justify-center py-4"><div class="spinner"></div></div>
              <template v-else>
                <div class="mb-3 flex flex-wrap items-center gap-4 text-body text-text-secondary">
                  <span class="font-semibold text-[var(--color-positive)]">{{ run.available }} available</span>
                  <span class="font-semibold text-warning">{{ run.unknown }} unknown</span>
                  <span v-if="run.failMessage" class="text-[var(--color-negative)]">{{ run.failMessage }}</span>
                </div>
                <p v-if="expandedResults.length === 0" class="py-4 text-center text-text-muted">No results for this run.</p>
                <div v-else class="overflow-x-auto">
                  <table class="w-full border-collapse text-body [&_th]:border-b [&_th]:border-border-subtle [&_th]:px-2 [&_th]:py-2 [&_th]:text-left [&_th]:text-label [&_th]:font-semibold [&_th]:uppercase [&_th]:tracking-[0.05em] [&_th]:text-text-muted [&_td]:border-b [&_td]:border-border-subtle [&_td]:px-2 [&_td]:py-2 [&_td]:text-left">
                    <thead>
                      <tr>
                        <th>Coin</th>
                        <th>URL</th>
                        <th>Status</th>
                        <th>Reason</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="result in expandedResults" :key="result.id">
                        <td>{{ result.coinName }}</td>
                        <td>
                          <SafeExternalLink v-if="safeResultUrl(result.url)" :href="result.url" target="_blank" rel="noopener" class="text-gold no-underline hover:underline" @click.stop>
                            {{ truncateUrl(result.url) }}
                          </SafeExternalLink>
                          <span v-else class="text-text-muted">--</span>
                        </td>
                        <td><span class="chip-sm" :class="resultStatusClass(result.status)">{{ result.status }}</span></td>
                        <td class="max-w-[220px] overflow-hidden text-ellipsis whitespace-nowrap">{{ result.reason || '--' }}</td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </template>
            </div>
          </div>
        </div>

        <div class="mt-6 flex items-center justify-center gap-4">
          <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="prevPage()">&larr; Previous</button>
          <span class="text-body text-text-secondary">Page {{ page }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="page * limit >= total" @click="nextPage()">Next &rarr;</button>
        </div>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft } from 'lucide-vue-next'
import { getMyAvailabilityRunDetail, listMyAvailabilityRuns } from '@/api/client'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'
import type { AvailabilityResult, AvailabilityRun } from '@/types'

const route = useRoute()

const runs = ref<AvailabilityRun[]>([])
const total = ref(0)
const page = ref(1)
const limit = 20
const loading = ref(false)
const loadError = ref('')
const unauthorized = ref(false)

const expandedRunId = ref<number | null>(null)
const expandedResults = ref<AvailabilityResult[]>([])
const expandedLoading = ref(false)

const routeRunId = computed(() => {
  const raw = route.params.id
  const id = Array.isArray(raw) ? raw[0] : raw
  const parsed = id ? Number(id) : NaN
  return Number.isFinite(parsed) ? parsed : null
})
const detailRun = ref<AvailabilityRun | null>(null)
const detailLoading = ref(false)
const detailError = ref('')

function isUnauthorizedError(err: unknown): boolean {
  return (err as { response?: { status?: number } })?.response?.status === 401
}

function statusClass(status: string): string {
  if (status === 'queued') return 'text-gold'
  if (status === 'running') return 'text-[var(--accent-bronze)]'
  if (status === 'failed' || status === 'partial_failure') return 'text-[var(--color-negative)]'
  return 'text-[var(--color-positive)]'
}

function resultStatusClass(status: string): string {
  if (status === 'available') return 'text-[var(--color-positive)]'
  if (status === 'unavailable') return 'text-[var(--color-negative)]'
  return 'text-warning'
}

function safeResultUrl(url: string | null | undefined): string | null {
  return sanitizeExternalUrl(url)
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleString()
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

async function loadRuns() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await listMyAvailabilityRuns(page.value, limit)
    runs.value = res.data?.runs ?? []
    total.value = res.data?.total ?? 0
  } catch (err) {
    if (isUnauthorizedError(err)) {
      unauthorized.value = true
    } else {
      loadError.value = 'Failed to load availability run history. Please try again.'
    }
    runs.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

async function prevPage() {
  page.value = Math.max(1, page.value - 1)
  await loadRuns()
}

async function nextPage() {
  page.value++
  await loadRuns()
}

async function toggleExpand(runId: number) {
  if (expandedRunId.value === runId) {
    expandedRunId.value = null
    expandedResults.value = []
    return
  }
  expandedRunId.value = runId
  expandedResults.value = []
  expandedLoading.value = true
  try {
    const res = await getMyAvailabilityRunDetail(runId)
    expandedResults.value = res.data?.results ?? []
  } catch (err) {
    if (isUnauthorizedError(err)) unauthorized.value = true
    expandedResults.value = []
  } finally {
    expandedLoading.value = false
  }
}

async function loadDetail(runId: number) {
  detailLoading.value = true
  detailError.value = ''
  try {
    const res = await getMyAvailabilityRunDetail(runId)
    detailRun.value = res.data ?? null
  } catch (err) {
    if (isUnauthorizedError(err)) {
      unauthorized.value = true
    } else {
      detailError.value = 'This run could not be found or is no longer available.'
    }
    detailRun.value = null
  } finally {
    detailLoading.value = false
  }
}

onMounted(async () => {
  if (routeRunId.value !== null) {
    await loadDetail(routeRunId.value)
    return
  }
  await loadRuns()
})
</script>
