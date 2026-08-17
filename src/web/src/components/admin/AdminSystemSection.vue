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

      <section
        class="mt-6 min-w-0 rounded-md border border-border-subtle bg-input p-4 md:p-6"
        aria-labelledby="ocre-section-heading"
        data-testid="ocre-section"
      >
        <div class="mb-5">
          <p class="section-label">Deep Analysis Provider</p>
          <h3 id="ocre-section-heading" class="m-0 text-lg font-medium text-heading">OCRE / Deep Analysis</h3>
          <p class="mt-1 text-sm text-text-muted">
            Online Coins of the Roman Empire (Nomisma.org). Off by default; when enabled, Deep Analysis may
            consult OCRE for Roman Imperial coin-type identification within a bounded per-job call budget.
          </p>
        </div>

        <div class="form-group min-w-0">
          <label class="flex items-center gap-3">
            <input
              id="deep-identification-enabled"
              v-model="localDeepIdentificationEnabled"
              type="checkbox"
              name="DeepIdentificationEnabled"
              data-testid="deep-identification-enabled-toggle"
              aria-describedby="deep-identification-enabled-help"
            />
            <span class="form-label m-0">Enable Deep Analysis</span>
          </label>
          <span id="deep-identification-enabled-help" class="mt-1 block text-sm text-text-muted">
            Shows the Deep Analysis action in Identify Coin and on saved coins, and allows new background analysis jobs.
          </span>
        </div>

        <fieldset class="mb-6 min-w-0">
          <legend class="section-label mb-3">Deep Analysis Limits</legend>
          <div class="grid min-w-0 gap-4 md:grid-cols-2">
            <div v-for="setting in deepIdentificationSettingFields" :key="setting.name" class="form-group min-w-0">
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

        <div class="form-group min-w-0">
          <label class="flex items-center gap-3">
            <input
              id="ocre-enabled"
              v-model="localOCREEnabled"
              type="checkbox"
              name="DeepIdentificationOCREEnabled"
              data-testid="ocre-enabled-toggle"
              aria-describedby="ocre-enabled-help"
            />
            <span class="form-label m-0">Enable OCRE Deep Analysis provider</span>
          </label>
          <span id="ocre-enabled-help" class="mt-1 block text-sm text-text-muted">
            When off, no OCRE requests are ever made. Rollback at any time by disabling this toggle.
          </span>
        </div>

        <fieldset class="mb-2 min-w-0">
          <legend class="section-label mb-3">OCRE Limits</legend>
          <div class="grid min-w-0 gap-4 md:grid-cols-2">
            <div class="form-group min-w-0">
              <label class="form-label" for="ocre-call-budget">Per-job call budget</label>
              <input
                id="ocre-call-budget"
                v-model="localOCRECallBudget"
                class="form-input"
                type="number"
                name="DeepIdentificationOCRECallBudget"
                :min="1"
                :max="20"
                step="1"
                required
              />
              <span class="mt-1 block text-sm text-text-muted">Requests per job; 1–20. Default 3.</span>
            </div>
          </div>
        </fieldset>

        <section class="border-t border-border-subtle pt-6" aria-labelledby="ocre-health-heading">
          <div class="mb-4 flex items-center justify-between gap-3">
            <h3 id="ocre-health-heading" class="m-0 text-lg font-medium text-heading">OCRE Health</h3>
            <button type="button" class="btn btn-secondary btn-xs" :disabled="ocreHealthLoading" @click="loadOCREHealth">
              {{ ocreHealthLoading ? 'Refreshing...' : 'Refresh' }}
            </button>
          </div>

          <div v-if="ocreHealthLoading" class="flex items-center gap-3 py-6 text-sm text-text-secondary" role="status">
            <span class="sr-only">Loading OCRE health</span>
            Loading OCRE health
          </div>
          <p v-else-if="ocreHealthError" class="rounded-sm border border-[var(--color-negative)] p-3 text-sm text-[var(--color-negative)]" role="alert">
            OCRE health is temporarily unavailable. Try again.
          </p>
          <div v-else-if="ocreHealth" class="grid gap-4 md:grid-cols-3" data-testid="ocre-health">
            <div class="rounded-sm border border-border-subtle bg-card p-4">
              <p class="section-label m-0">Provider</p>
              <div class="mt-1 font-semibold text-text-primary">{{ ocreHealth.enabled ? 'Enabled' : 'Disabled' }}</div>
              <div class="mt-1 text-sm text-text-secondary">{{ ocreHealth.gateValidated ? 'Configuration valid' : 'Configuration invalid' }}</div>
            </div>
            <div class="rounded-sm border border-border-subtle bg-card p-4">
              <p class="section-label m-0">Call budget</p>
              <div class="mt-1 text-sm text-text-primary">{{ ocreHealth.callBudget }} per job</div>
            </div>
            <div class="rounded-sm border border-border-subtle bg-card p-4">
              <p class="section-label m-0">Last outcome</p>
              <div class="mt-1 font-semibold text-text-primary">{{ formatOCREOutcome(ocreHealth.lastOutcome) }}</div>
              <div class="mt-1 text-sm text-text-secondary">{{ formatTimestamp(ocreHealth.lastCheckedAt) }}</div>
            </div>
          </div>
          <p v-else class="py-6 text-sm text-text-muted">No OCRE health data is available yet.</p>
        </section>

        <section class="mt-6 border-t border-border-subtle pt-6" aria-labelledby="deep-observability-heading">
          <div class="mb-4 flex items-center justify-between gap-3">
            <h3 id="deep-observability-heading" class="m-0 text-lg font-medium text-heading">Deep Analysis Operations</h3>
            <button type="button" class="btn btn-secondary btn-xs min-h-[44px]" :disabled="deepMetricsLoading" @click="loadDeepMetrics">
              {{ deepMetricsLoading ? 'Refreshing...' : 'Refresh' }}
            </button>
          </div>
          <div v-if="deepMetricsLoading" class="py-6 text-sm text-text-secondary" role="status">Loading Deep Analysis operations</div>
          <p v-else-if="deepMetricsError" class="rounded-sm border border-[var(--color-negative)] p-3 text-sm text-[var(--color-negative)]" role="alert">
            Deep Analysis operations are temporarily unavailable. Try again.
          </p>
          <div v-else-if="deepMetrics" class="grid gap-4" data-testid="deep-observability">
            <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <p class="section-label m-0">Terminal jobs</p>
                <div class="mt-1 text-sm text-text-primary">{{ terminalJobCount }}</div>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <p class="section-label m-0">Partial success</p>
                <div class="mt-1 text-sm text-text-primary">{{ formatRate(deepMetrics.partialSuccessRate) }}</div>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <p class="section-label m-0">Job duration</p>
                <div class="mt-1 text-sm text-text-primary">p50 {{ formatDuration(deepMetrics.duration.p50Ms) }} · p95 {{ formatDuration(deepMetrics.duration.p95Ms) }}</div>
              </div>
              <div class="rounded-sm border border-border-subtle bg-card p-4">
                <p class="section-label m-0">Queue / live streams</p>
                <div class="mt-1 text-sm text-text-primary">{{ deepMetrics.queueDepth }} queued · {{ deepMetrics.activeSseStreams }} live</div>
              </div>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full min-w-[540px] text-left text-sm">
                <thead class="text-text-muted">
                  <tr><th class="p-2">Provider</th><th class="p-2">Outcomes</th><th class="p-2">Latency</th></tr>
                </thead>
                <tbody>
                  <tr v-for="(metrics, provider) in deepMetrics.providers" :key="provider" class="border-t border-border-subtle">
                    <td class="p-2 font-semibold uppercase text-text-primary">{{ provider }}</td>
                    <td class="p-2 text-text-secondary">{{ formatStatusCounts(metrics.statusCounts) }}</td>
                    <td class="p-2 text-text-secondary">p50 {{ formatDuration(metrics.latency.p50Ms) }} · p95 {{ formatDuration(metrics.latency.p95Ms) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p class="m-0 text-sm text-text-secondary">
              SSE reconnects {{ deepMetrics.reconnectCount }}, truncations {{ deepMetrics.truncationCount }};
              hint cleanup {{ deepMetrics.hintDeletion.success }} succeeded / {{ deepMetrics.hintDeletion.failure }} failed;
              janitor {{ deepMetrics.janitor.recoverySweeps }} recovery / {{ deepMetrics.janitor.retentionSweeps }} retention sweeps,
              {{ deepMetrics.janitor.failures }} failures.
            </p>
          </div>
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
import { getAdminDeepIdentificationObservability, getAdminNumistaHealth, getAdminOCREHealth } from '@/api/client'
import type { DeepIdentificationObservabilitySummary, NumistaHealthSummary, NumistaLookupStatus, OCREHealthSummary } from '@/types'

const props = withDefaults(defineProps<{
  numistaApiKey: string
  numistaSearchTTLHours: string
  numistaDetailTTLHours: string
  numistaEnrichmentLimit: string
  numistaSearchResultLimit: string
  numistaSearchTimeoutSeconds: string
  numistaDetailTimeoutSeconds: string
  deepIdentificationEnabled?: string
  deepIdentificationWorkerCount?: string
  deepIdentificationMaxActivePerUser?: string
  deepIdentificationQueueDepth?: string
  deepIdentificationHardTimeoutSeconds?: string
  deepIdentificationEventRetentionHours?: string
  deepIdentificationResultRetentionDays?: string
  deepIdentificationMaxProviders?: string
  deepIdentificationNumistaCallBudget?: string
  deepIdentificationOCREEnabled: string
  deepIdentificationOCRECallBudget: string
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
}>(), {
  deepIdentificationEnabled: 'false',
  deepIdentificationWorkerCount: '2',
  deepIdentificationMaxActivePerUser: '1',
  deepIdentificationQueueDepth: '32',
  deepIdentificationHardTimeoutSeconds: '300',
  deepIdentificationEventRetentionHours: '24',
  deepIdentificationResultRetentionDays: '90',
  deepIdentificationMaxProviders: '4',
  deepIdentificationNumistaCallBudget: '4',
})

const emit = defineEmits<{
  save: [settings: {
    numistaApiKey: string
    numistaSearchTTLHours: string
    numistaDetailTTLHours: string
    numistaEnrichmentLimit: string
    numistaSearchResultLimit: string
    numistaSearchTimeoutSeconds: string
    numistaDetailTimeoutSeconds: string
    deepIdentificationEnabled: string
    deepIdentificationWorkerCount: string
    deepIdentificationMaxActivePerUser: string
    deepIdentificationQueueDepth: string
    deepIdentificationHardTimeoutSeconds: string
    deepIdentificationEventRetentionHours: string
    deepIdentificationResultRetentionDays: string
    deepIdentificationMaxProviders: string
    deepIdentificationNumistaCallBudget: string
    deepIdentificationOCREEnabled: string
    deepIdentificationOCRECallBudget: string
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
const localDeepIdentificationEnabled = ref((props.deepIdentificationEnabled || 'false') === 'true')
const localDeepIdentificationWorkerCount = ref(props.deepIdentificationWorkerCount)
const localDeepIdentificationMaxActivePerUser = ref(props.deepIdentificationMaxActivePerUser)
const localDeepIdentificationQueueDepth = ref(props.deepIdentificationQueueDepth)
const localDeepIdentificationHardTimeoutSeconds = ref(props.deepIdentificationHardTimeoutSeconds)
const localDeepIdentificationEventRetentionHours = ref(props.deepIdentificationEventRetentionHours)
const localDeepIdentificationResultRetentionDays = ref(props.deepIdentificationResultRetentionDays)
const localDeepIdentificationMaxProviders = ref(props.deepIdentificationMaxProviders)
const localDeepIdentificationNumistaCallBudget = ref(props.deepIdentificationNumistaCallBudget)
const localOCREEnabled = ref((props.deepIdentificationOCREEnabled || 'false') === 'true')
const localOCRECallBudget = ref(props.deepIdentificationOCRECallBudget || '3')
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
const ocreHealth = ref<OCREHealthSummary | null>(null)
const ocreHealthLoading = ref(true)
const ocreHealthError = ref(false)
const deepMetrics = ref<DeepIdentificationObservabilitySummary | null>(null)
const deepMetricsLoading = ref(true)
const deepMetricsError = ref(false)
const terminalJobCount = computed(() =>
  Object.values(deepMetrics.value?.jobsByTerminalStatus ?? {}).reduce((total, count) => total + count, 0),
)
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

const deepIdentificationSettingFields = [
  { name: 'DeepIdentificationWorkerCount', label: 'Worker count', min: 1, max: 32, fallback: 2, model: localDeepIdentificationWorkerCount, hint: 'Concurrent job workers; 1–32. Default 2.' },
  { name: 'DeepIdentificationMaxActivePerUser', label: 'Active jobs per user', min: 1, max: 10, fallback: 1, model: localDeepIdentificationMaxActivePerUser, hint: 'Per-user active job limit; 1–10. Default 1.' },
  { name: 'DeepIdentificationQueueDepth', label: 'Queue depth', min: 1, max: 1000, fallback: 32, model: localDeepIdentificationQueueDepth, hint: 'Queued jobs; 1–1000. Default 32.' },
  { name: 'DeepIdentificationHardTimeoutSeconds', label: 'Hard timeout', min: 1, max: 900, fallback: 300, model: localDeepIdentificationHardTimeoutSeconds, hint: 'Seconds per job; 1–900. Default 300.' },
  { name: 'DeepIdentificationEventRetentionHours', label: 'Event retention', min: 1, max: 720, fallback: 24, model: localDeepIdentificationEventRetentionHours, hint: 'Hours; 1–720. Default 24.' },
  { name: 'DeepIdentificationResultRetentionDays', label: 'Result retention', min: 1, max: 3650, fallback: 90, model: localDeepIdentificationResultRetentionDays, hint: 'Days; 1–3650. Default 90.' },
  { name: 'DeepIdentificationMaxProviders', label: 'Provider limit', min: 1, max: 10, fallback: 4, model: localDeepIdentificationMaxProviders, hint: 'Providers per job; 1–10. Default 4.' },
  { name: 'DeepIdentificationNumistaCallBudget', label: 'Numista call budget', min: 1, max: 20, fallback: 4, model: localDeepIdentificationNumistaCallBudget, hint: 'Requests per job; 1–20. Default 4.' },
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
  for (const setting of deepIdentificationSettingFields) {
    setting.model.value = boundedValue(setting.model.value, setting.min, setting.max, setting.fallback)
  }
  localOCRECallBudget.value = boundedValue(localOCRECallBudget.value, 1, 20, 3)
  emit('save', {
    numistaApiKey: localNumistaApiKey.value,
    numistaSearchTTLHours: localNumistaSearchTTLHours.value,
    numistaDetailTTLHours: localNumistaDetailTTLHours.value,
    numistaEnrichmentLimit: localNumistaEnrichmentLimit.value,
    numistaSearchResultLimit: localNumistaSearchResultLimit.value,
    numistaSearchTimeoutSeconds: localNumistaSearchTimeoutSeconds.value,
    numistaDetailTimeoutSeconds: localNumistaDetailTimeoutSeconds.value,
    deepIdentificationEnabled: localDeepIdentificationEnabled.value ? 'true' : 'false',
    deepIdentificationWorkerCount: localDeepIdentificationWorkerCount.value,
    deepIdentificationMaxActivePerUser: localDeepIdentificationMaxActivePerUser.value,
    deepIdentificationQueueDepth: localDeepIdentificationQueueDepth.value,
    deepIdentificationHardTimeoutSeconds: localDeepIdentificationHardTimeoutSeconds.value,
    deepIdentificationEventRetentionHours: localDeepIdentificationEventRetentionHours.value,
    deepIdentificationResultRetentionDays: localDeepIdentificationResultRetentionDays.value,
    deepIdentificationMaxProviders: localDeepIdentificationMaxProviders.value,
    deepIdentificationNumistaCallBudget: localDeepIdentificationNumistaCallBudget.value,
    deepIdentificationOCREEnabled: localOCREEnabled.value ? 'true' : 'false',
    deepIdentificationOCRECallBudget: localOCRECallBudget.value,
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

async function loadOCREHealth() {
  ocreHealthLoading.value = true
  ocreHealthError.value = false
  try {
    const response = await getAdminOCREHealth()
    ocreHealth.value = response.data
  } catch {
    ocreHealth.value = null
    ocreHealthError.value = true
  } finally {
    ocreHealthLoading.value = false
  }
}

async function loadDeepMetrics() {
  deepMetricsLoading.value = true
  deepMetricsError.value = false
  try {
    const response = await getAdminDeepIdentificationObservability()
    deepMetrics.value = response.data
  } catch {
    deepMetrics.value = null
    deepMetricsError.value = true
  } finally {
    deepMetricsLoading.value = false
  }
}

function formatRate(value: number) {
  return `${(Math.min(1, Math.max(0, value)) * 100).toFixed(1)}%`
}

function formatDuration(milliseconds: number) {
  if (milliseconds <= 0) return 'not observed'
  if (milliseconds < 1000) return `${milliseconds} ms`
  return `${(milliseconds / 1000).toFixed(1)} s`
}

function formatStatusCounts(counts: Record<string, number>) {
  const entries = Object.entries(counts).filter(([, count]) => count > 0)
  return entries.length
    ? entries.map(([status, count]) => `${formatOCREOutcome(status)} ${count}`).join(', ')
    : 'No attempts'
}

function formatOCREOutcome(status?: string | null) {
  if (!status) return 'No recent outcome'
  if (status === 'not_automated') return 'Manual Verification'
  return status
    .split('_')
    .map((part) => (part ? part.charAt(0).toUpperCase() + part.slice(1) : part))
    .join(' ')
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
watch(() => props.deepIdentificationEnabled, (value) => { localDeepIdentificationEnabled.value = (value || 'false') === 'true' })
watch(() => props.deepIdentificationWorkerCount, (value) => { localDeepIdentificationWorkerCount.value = value })
watch(() => props.deepIdentificationMaxActivePerUser, (value) => { localDeepIdentificationMaxActivePerUser.value = value })
watch(() => props.deepIdentificationQueueDepth, (value) => { localDeepIdentificationQueueDepth.value = value })
watch(() => props.deepIdentificationHardTimeoutSeconds, (value) => { localDeepIdentificationHardTimeoutSeconds.value = value })
watch(() => props.deepIdentificationEventRetentionHours, (value) => { localDeepIdentificationEventRetentionHours.value = value })
watch(() => props.deepIdentificationResultRetentionDays, (value) => { localDeepIdentificationResultRetentionDays.value = value })
watch(() => props.deepIdentificationMaxProviders, (value) => { localDeepIdentificationMaxProviders.value = value })
watch(() => props.deepIdentificationNumistaCallBudget, (value) => { localDeepIdentificationNumistaCallBudget.value = value })
watch(() => props.deepIdentificationOCREEnabled, (value) => { localOCREEnabled.value = (value || 'false') === 'true' })
watch(() => props.deepIdentificationOCRECallBudget, (value) => { localOCRECallBudget.value = value || '3' })
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

onMounted(() => {
  loadHealth()
  loadOCREHealth()
  loadDeepMetrics()
})
</script>
