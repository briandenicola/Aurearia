<template>
  <section class="card">
    <h2 class="mb-5 border-b border-border-subtle pb-3 text-xl font-medium">System Settings</h2>
    <form @submit.prevent="save">
      <div class="form-group">
        <label class="form-label">Pushover API Token</label>
        <input v-model="localPushoverAppToken" class="form-input" type="password" placeholder="Enter your Pushover application API token" />
        <span class="mt-1 block text-sm text-text-muted">Create an app at <a href="https://pushover.net/apps" target="_blank" rel="noopener">pushover.net/apps</a> to get a token. Users provide their own User Key in Account Settings.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Public App URL</label>
        <input v-model="localPublicAppUrl" class="form-input" type="url" placeholder="https://coins.example.com" />
        <span class="mt-1 block text-sm text-text-muted">Full browser URL for this app. Used to make Pushover Coin of the Day links open directly to a coin; leave blank to send the alert without an external link.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Log Level</label>
        <select v-model="localLogLevel" class="form-select">
          <option v-for="level in logLevels" :key="level" :value="level">{{ level }}</option>
        </select>
      </div>

      <section
        class="mt-6 min-w-0 rounded-md border border-border-subtle bg-input p-4 md:p-6"
        aria-labelledby="numista-section-heading"
        data-testid="numista-section"
      >
        <div class="mb-5">
          <p class="section-label">Catalog Integration</p>
          <h3 id="numista-section-heading" class="m-0 text-lg font-medium text-heading">Numista</h3>
        </div>

        <div class="form-group">
          <label class="form-label" for="numista-api-key">Numista API Key</label>
          <input
            id="numista-api-key"
            v-model="localNumistaApiKey"
            class="form-input"
            type="password"
            autocomplete="off"
            aria-describedby="numista-api-key-help"
            placeholder="Enter your Numista API key"
          />
          <span id="numista-api-key-help" class="mt-1 block text-sm text-text-muted">Get a free key at <a href="https://en.numista.com/api/" target="_blank" rel="noopener">numista.com/api</a> (2,000 requests/month free)</span>
        </div>

        <fieldset class="mb-6 min-w-0">
          <legend class="section-label mb-3">Numista Lookup Limits</legend>
          <div class="grid min-w-0 gap-4 md:grid-cols-2">
            <div v-for="setting in numistaSettingFields" :key="setting.name" class="form-group min-w-0">
              <label class="form-label" :for="setting.name">{{ setting.label }}</label>
              <input
                :id="setting.name"
                v-model="setting.model.value"
                class="form-input"
                type="number"
                :name="setting.name"
                :min="setting.min"
                :max="setting.max"
                step="1"
                required
              />
              <span class="mt-1 block text-sm text-text-muted">{{ setting.hint }}</span>
            </div>
          </div>
        </fieldset>

        <section class="border-t border-border-subtle pt-6" aria-labelledby="numista-health-heading">
          <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
            <h3 id="numista-health-heading" class="m-0 text-lg font-medium text-heading">Numista Health</h3>
            <button type="button" class="btn btn-secondary btn-xs" :disabled="healthLoading" @click="loadHealth">
              {{ healthLoading ? 'Refreshing...' : 'Refresh' }}
            </button>
          </div>

          <div v-if="healthLoading" class="flex items-center gap-3 py-6 text-sm text-text-secondary" role="status">
            <span class="spinner"></span>
            Loading Numista health
          </div>
          <p v-else-if="healthError" class="rounded-sm border border-[var(--color-negative)] p-3 text-sm text-[var(--color-negative)]" role="alert">
            Numista health is temporarily unavailable. Try again.
          </p>
          <div v-else-if="health && !hasHealthEvents" class="rounded-sm border border-border-subtle bg-card p-4" role="status">
            <h4 class="section-label mb-2">No Recent Activity</h4>
            <p class="m-0 text-sm text-text-primary">No Numista lookup health events have been recorded yet.</p>
            <p class="mb-0 mt-1 text-sm text-text-secondary">
              {{ health.configured ? 'Numista is configured.' : 'Numista is not configured.' }}
              Refresh after a lookup to view operational metrics.
            </p>
          </div>
          <div v-else-if="health" class="grid gap-4">
            <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <div class="rounded-sm border border-border-subtle bg-card p-3">
                <div class="section-label">Configuration</div>
                <div class="mt-1 font-semibold text-text-primary">{{ health.configured ? 'Configured' : 'Not configured' }}</div>
                <div class="mt-1 text-sm text-text-secondary">{{ health.configurationValid ? 'Configuration valid' : 'Configuration invalid' }}</div>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-3">
                <div class="section-label">Requests</div>
                <div class="mt-1 text-sm text-text-primary">{{ health.broadRequestCount }} broad</div>
                <div class="mt-1 text-sm text-text-secondary">{{ health.detailRequestCount }} detail</div>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-3">
                <div class="section-label">Latency</div>
                <div class="mt-1 text-sm text-text-primary">p50 {{ health.p50ElapsedMs }} ms</div>
                <div class="mt-1 text-sm text-text-secondary">p95 {{ health.p95ElapsedMs }} ms</div>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-3">
                <div class="section-label">Latest Outcome</div>
                <div class="mt-1 font-semibold text-text-primary">{{ formatStatus(health.lastOutcome) }}</div>
                <div class="mt-1 text-sm text-text-secondary">{{ formatTimestamp(health.lastCheckedAt) }}</div>
              </div>
            </div>

            <div class="rounded-sm border border-border-subtle bg-card p-4">
              <h4 class="section-label mb-3">Status Counts</h4>
              <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
                <div v-for="status in statusRows" :key="status.key" class="flex items-center justify-between gap-2 rounded-sm bg-input px-3 py-2">
                  <span class="text-sm text-text-secondary">{{ status.label }}</span>
                  <strong class="text-gold">{{ health.statusCounts[status.key] ?? 0 }}</strong>
                </div>
              </div>
            </div>

            <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <h4 class="section-label mb-2">Cache</h4>
                <p class="m-0 text-sm text-text-primary">{{ health.freshCacheHitCount }} fresh cache hits</p>
                <p class="mb-0 mt-1 text-sm text-text-secondary">{{ health.coalescedRequestCount }} coalesced requests · {{ formatPercent(health.freshCacheHitRate) }}</p>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <h4 class="section-label mb-2">Provider Loads</h4>
                <p class="m-0 text-sm text-text-primary">{{ health.providerLoadCount }} loads</p>
                <p class="mb-0 mt-1 text-sm text-text-secondary">{{ health.providerFailureCount }} failed · {{ health.cancelledRequestCount }} cancelled</p>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <h4 class="section-label mb-2">Enrichment</h4>
                <p class="m-0 text-sm text-text-primary">{{ health.enrichmentAttempted }} attempted</p>
                <p class="mb-0 mt-1 text-sm text-text-secondary">{{ health.enrichmentSucceeded }} succeeded · {{ health.enrichmentFailed }} failed</p>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <h4 class="section-label mb-2">Quota Signals</h4>
                <p class="m-0 text-sm text-text-primary">Last limited: {{ formatTimestamp(health.lastQuotaLimitedAt) }}</p>
                <p class="mb-0 mt-1 text-sm text-text-secondary">Retry after: {{ formatRetryAfter(health.lastRetryAfterSeconds) }}</p>
              </div>
            </div>
          </div>
          <p v-else class="py-6 text-sm text-text-muted">No Numista health data is available yet.</p>
        </section>
      </section>

      <p v-if="msg" class="my-2 text-body" :class="error ? 'text-[var(--color-negative)]' : 'text-gold'">{{ msg }}</p>
      <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">
        {{ saving ? 'Saving...' : 'Save System Settings' }}
      </button>
    </form>

    <div class="mt-6 flex items-center gap-2 border-t border-border-subtle pt-4 text-[0.78rem] text-text-muted">
      <span class="font-semibold uppercase tracking-[0.05em]">Version</span>
      <span class="font-mono text-text-secondary">{{ appVersion }}</span>
      <span v-if="buildDate" class="ml-1">Built {{ buildDate }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { getAdminNumistaHealth } from '@/api/client'
import type { NumistaHealthSummary, NumistaLookupStatus } from '@/types'

const props = defineProps<{
  numistaApiKey: string
  numistaSearchTTLHours: string
  numistaDetailTTLHours: string
  numistaEnrichmentLimit: string
  numistaSearchResultLimit: string
  numistaSearchTimeoutSeconds: string
  numistaDetailTimeoutSeconds: string
  pushoverAppToken: string
  publicAppUrl: string
  uspsApiBaseUrl: string
  uspsApiKey: string
  uspsApiKeyHeader: string
  upsApiBaseUrl: string
  upsTokenUrl: string
  upsClientId: string
  upsClientSecret: string
  upsScope: string
  fedexApiBaseUrl: string
  fedexTokenUrl: string
  fedexClientId: string
  fedexClientSecret: string
  fedexScope: string
  logLevel: string
  logLevels: readonly string[]
  saving: boolean
  msg: string
  error: boolean
  appVersion: string
  buildDate: string
}>()

const emit = defineEmits<{
  save: [settings: {
    numistaApiKey: string
    numistaSearchTTLHours: string
    numistaDetailTTLHours: string
    numistaEnrichmentLimit: string
    numistaSearchResultLimit: string
    numistaSearchTimeoutSeconds: string
    numistaDetailTimeoutSeconds: string
    logLevel: string
    pushoverAppToken: string
    publicAppUrl: string
    uspsApiBaseUrl: string
    uspsApiKey: string
    uspsApiKeyHeader: string
    upsApiBaseUrl: string
    upsTokenUrl: string
    upsClientId: string
    upsClientSecret: string
    upsScope: string
    fedexApiBaseUrl: string
    fedexTokenUrl: string
    fedexClientId: string
    fedexClientSecret: string
    fedexScope: string
  }]
}>()

const localNumistaApiKey = ref(props.numistaApiKey)
const localNumistaSearchTTLHours = ref(props.numistaSearchTTLHours || '24')
const localNumistaDetailTTLHours = ref(props.numistaDetailTTLHours || '168')
const localNumistaEnrichmentLimit = ref(props.numistaEnrichmentLimit || '5')
const localNumistaSearchResultLimit = ref(props.numistaSearchResultLimit || '20')
const localNumistaSearchTimeoutSeconds = ref(props.numistaSearchTimeoutSeconds || '4')
const localNumistaDetailTimeoutSeconds = ref(props.numistaDetailTimeoutSeconds || '3')
const localPushoverAppToken = ref(props.pushoverAppToken)
const localPublicAppUrl = ref(props.publicAppUrl)
const localUSPSAPIBaseURL = ref(props.uspsApiBaseUrl)
const localUSPSAPIKey = ref(props.uspsApiKey)
const localUSPSAPIKeyHeader = ref(props.uspsApiKeyHeader)
const localUPSAPIBaseURL = ref(props.upsApiBaseUrl)
const localUPSTokenURL = ref(props.upsTokenUrl)
const localUPSClientID = ref(props.upsClientId)
const localUPSClientSecret = ref(props.upsClientSecret)
const localUPSScope = ref(props.upsScope)
const localFedExAPIBaseURL = ref(props.fedexApiBaseUrl)
const localFedExTokenURL = ref(props.fedexTokenUrl)
const localFedExClientID = ref(props.fedexClientId)
const localFedExClientSecret = ref(props.fedexClientSecret)
const localFedExScope = ref(props.fedexScope)
const localLogLevel = ref(props.logLevel)
const health = ref<NumistaHealthSummary | null>(null)
const healthLoading = ref(true)
const healthError = ref(false)
const hasHealthEvents = computed(() => {
  if (!health.value) return false
  const counters = [
    health.value.broadRequestCount,
    health.value.detailRequestCount,
    health.value.freshCacheHitCount,
    health.value.coalescedRequestCount,
    health.value.providerLoadCount,
    health.value.providerFailureCount,
    health.value.cancelledRequestCount,
    health.value.enrichmentAttempted,
    health.value.enrichmentSucceeded,
    health.value.enrichmentFailed,
  ]
  return counters.some((count) => count > 0) ||
    Object.values(health.value.statusCounts).some((count) => count > 0) ||
    Boolean(health.value.lastOutcome || health.value.lastCheckedAt ||
      health.value.lastQuotaLimitedAt || (health.value.lastRetryAfterSeconds ?? 0) > 0)
})

const numistaSettingFields = [
  { name: 'NumistaSearchTTLHours', label: 'Search cache TTL', min: 1, max: 720, fallback: 24, model: localNumistaSearchTTLHours, hint: 'Hours; 1–720. Default 24.' },
  { name: 'NumistaDetailTTLHours', label: 'Detail cache TTL', min: 1, max: 2160, fallback: 168, model: localNumistaDetailTTLHours, hint: 'Hours; 1–2160. Default 168.' },
  { name: 'NumistaEnrichmentLimit', label: 'Enrichment limit', min: 1, max: 10, fallback: 5, model: localNumistaEnrichmentLimit, hint: 'Candidates per lookup; 1–10. Default 5.' },
  { name: 'NumistaSearchResultLimit', label: 'Search result limit', min: 1, max: 50, fallback: 20, model: localNumistaSearchResultLimit, hint: 'Broad candidates; 1–50. Default 20.' },
  { name: 'NumistaSearchTimeoutSeconds', label: 'Search timeout', min: 1, max: 10, fallback: 4, model: localNumistaSearchTimeoutSeconds, hint: 'Seconds; 1–10. Default 4.' },
  { name: 'NumistaDetailTimeoutSeconds', label: 'Detail timeout', min: 1, max: 10, fallback: 3, model: localNumistaDetailTimeoutSeconds, hint: 'Seconds; 1–10. Default 3.' },
] as const

const statusRows: { key: NumistaLookupStatus; label: string }[] = [
  { key: 'success', label: 'Success' },
  { key: 'empty', label: 'Empty' },
  { key: 'unconfigured', label: 'Unconfigured' },
  { key: 'quota-limited', label: 'Quota limited' },
  { key: 'timeout', label: 'Timeout' },
  { key: 'unavailable', label: 'Unavailable' },
]

function boundedValue(value: string, min: number, max: number, fallback: number) {
  const parsed = Number(value)
  if (!Number.isInteger(parsed)) return String(fallback)
  return String(Math.min(max, Math.max(min, parsed)))
}

function save() {
  for (const setting of numistaSettingFields) {
    setting.model.value = boundedValue(setting.model.value, setting.min, setting.max, setting.fallback)
  }
  emit('save', {
    numistaApiKey: localNumistaApiKey.value,
    numistaSearchTTLHours: localNumistaSearchTTLHours.value,
    numistaDetailTTLHours: localNumistaDetailTTLHours.value,
    numistaEnrichmentLimit: localNumistaEnrichmentLimit.value,
    numistaSearchResultLimit: localNumistaSearchResultLimit.value,
    numistaSearchTimeoutSeconds: localNumistaSearchTimeoutSeconds.value,
    numistaDetailTimeoutSeconds: localNumistaDetailTimeoutSeconds.value,
    logLevel: localLogLevel.value,
    pushoverAppToken: localPushoverAppToken.value,
    publicAppUrl: localPublicAppUrl.value,
    uspsApiBaseUrl: localUSPSAPIBaseURL.value,
    uspsApiKey: localUSPSAPIKey.value,
    uspsApiKeyHeader: localUSPSAPIKeyHeader.value,
    upsApiBaseUrl: localUPSAPIBaseURL.value,
    upsTokenUrl: localUPSTokenURL.value,
    upsClientId: localUPSClientID.value,
    upsClientSecret: localUPSClientSecret.value,
    upsScope: localUPSScope.value,
    fedexApiBaseUrl: localFedExAPIBaseURL.value,
    fedexTokenUrl: localFedExTokenURL.value,
    fedexClientId: localFedExClientID.value,
    fedexClientSecret: localFedExClientSecret.value,
    fedexScope: localFedExScope.value,
  })
}

async function loadHealth() {
  healthLoading.value = true
  healthError.value = false
  try {
    const response = await getAdminNumistaHealth()
    health.value = response.data
  } catch {
    health.value = null
    healthError.value = true
  } finally {
    healthLoading.value = false
  }
}

function formatStatus(status?: NumistaLookupStatus | null) {
  if (!status) return 'No recent outcome'
  return status === 'quota-limited' ? 'Quota limited' : status.charAt(0).toUpperCase() + status.slice(1)
}

function formatTimestamp(value?: string | null) {
  if (!value) return 'Not observed'
  const timestamp = new Date(value)
  return Number.isNaN(timestamp.getTime()) ? 'Not observed' : timestamp.toLocaleString()
}

function formatPercent(value: number) {
  const bounded = Math.min(1, Math.max(0, value))
  return `${(bounded * 100).toFixed(1)}% fresh hit rate`
}

function formatRetryAfter(value?: number | null) {
  return value && value > 0 ? `${value} seconds` : 'Not observed'
}

watch(() => props.numistaApiKey, (value) => { localNumistaApiKey.value = value })
watch(() => props.numistaSearchTTLHours, (value) => { localNumistaSearchTTLHours.value = value || '24' })
watch(() => props.numistaDetailTTLHours, (value) => { localNumistaDetailTTLHours.value = value || '168' })
watch(() => props.numistaEnrichmentLimit, (value) => { localNumistaEnrichmentLimit.value = value || '5' })
watch(() => props.numistaSearchResultLimit, (value) => { localNumistaSearchResultLimit.value = value || '20' })
watch(() => props.numistaSearchTimeoutSeconds, (value) => { localNumistaSearchTimeoutSeconds.value = value || '4' })
watch(() => props.numistaDetailTimeoutSeconds, (value) => { localNumistaDetailTimeoutSeconds.value = value || '3' })
watch(() => props.pushoverAppToken, (value) => { localPushoverAppToken.value = value })
watch(() => props.publicAppUrl, (value) => { localPublicAppUrl.value = value })
watch(() => props.uspsApiBaseUrl, (value) => { localUSPSAPIBaseURL.value = value })
watch(() => props.uspsApiKey, (value) => { localUSPSAPIKey.value = value })
watch(() => props.uspsApiKeyHeader, (value) => { localUSPSAPIKeyHeader.value = value })
watch(() => props.upsApiBaseUrl, (value) => { localUPSAPIBaseURL.value = value })
watch(() => props.upsTokenUrl, (value) => { localUPSTokenURL.value = value })
watch(() => props.upsClientId, (value) => { localUPSClientID.value = value })
watch(() => props.upsClientSecret, (value) => { localUPSClientSecret.value = value })
watch(() => props.upsScope, (value) => { localUPSScope.value = value })
watch(() => props.fedexApiBaseUrl, (value) => { localFedExAPIBaseURL.value = value })
watch(() => props.fedexTokenUrl, (value) => { localFedExTokenURL.value = value })
watch(() => props.fedexClientId, (value) => { localFedExClientID.value = value })
watch(() => props.fedexClientSecret, (value) => { localFedExClientSecret.value = value })
watch(() => props.fedexScope, (value) => { localFedExScope.value = value })
watch(() => props.logLevel, (value) => { localLogLevel.value = value })

onMounted(loadHealth)
</script>
