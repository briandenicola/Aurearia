<template>
  <section class="card">
    <h2 class="text-xl font-medium mb-5 pb-3 border-b border-border-subtle">System Settings</h2>
    <form @submit.prevent="$emit('save', {
      numistaApiKey: localNumistaApiKey,
      logLevel: localLogLevel,
      pushoverAppToken: localPushoverAppToken,
      publicAppUrl: localPublicAppUrl,
      uspsApiBaseUrl: localUSPSAPIBaseURL,
      uspsApiKey: localUSPSAPIKey,
      uspsApiKeyHeader: localUSPSAPIKeyHeader,
      upsApiBaseUrl: localUPSAPIBaseURL,
      upsTokenUrl: localUPSTokenURL,
      upsClientId: localUPSClientID,
      upsClientSecret: localUPSClientSecret,
      upsScope: localUPSScope,
      fedexApiBaseUrl: localFedExAPIBaseURL,
      fedexTokenUrl: localFedExTokenURL,
      fedexClientId: localFedExClientID,
      fedexClientSecret: localFedExClientSecret,
      fedexScope: localFedExScope,
    })">
      <div class="form-group">
        <label class="form-label">Numista API Key</label>
        <input v-model="localNumistaApiKey" class="form-input" type="password" placeholder="Enter your Numista API key" />
        <span class="form-hint text-sm text-text-muted mt-1 block">Get a free key at <a href="https://en.numista.com/api/" target="_blank" rel="noopener">numista.com/api</a> (2,000 requests/month free)</span>
      </div>
      <div class="form-group">
        <label class="form-label">Pushover API Token</label>
        <input v-model="localPushoverAppToken" class="form-input" type="password" placeholder="Enter your Pushover application API token" />
        <span class="form-hint text-sm text-text-muted mt-1 block">Create an app at <a href="https://pushover.net/apps" target="_blank" rel="noopener">pushover.net/apps</a> to get a token. Users provide their own User Key in Account Settings.</span>
      </div>
      <div class="form-group">
        <label class="form-label">Public App URL</label>
        <input v-model="localPublicAppUrl" class="form-input" type="url" placeholder="https://coins.example.com" />
        <span class="form-hint text-sm text-text-muted mt-1 block">Full browser URL for this app. Used to make Pushover Coin of the Day links open directly to a coin; leave blank to send the alert without an external link.</span>
      </div>
      <hr class="my-6 border-0 border-t border-border-subtle" />
      <h3 class="mb-4 text-base font-semibold text-text-primary">Shipment Carrier Integrations</h3>
      <div class="form-group">
        <label class="form-label">USPS API Base URL</label>
        <input v-model="localUSPSAPIBaseURL" class="form-input" type="url" placeholder="https://api.usps.com" />
      </div>
      <div class="form-group">
        <label class="form-label">USPS API Key</label>
        <input v-model="localUSPSAPIKey" class="form-input" type="password" placeholder="USPS API key" />
      </div>
      <div class="form-group">
        <label class="form-label">USPS API Key Header</label>
        <input v-model="localUSPSAPIKeyHeader" class="form-input" placeholder="X-API-Key" />
      </div>
      <div class="form-group">
        <label class="form-label">UPS API Base URL</label>
        <input v-model="localUPSAPIBaseURL" class="form-input" type="url" placeholder="https://onlinetools.ups.com" />
      </div>
      <div class="form-group">
        <label class="form-label">UPS Token URL</label>
        <input v-model="localUPSTokenURL" class="form-input" type="url" placeholder="https://onlinetools.ups.com/security/v1/oauth/token" />
      </div>
      <div class="form-group">
        <label class="form-label">UPS Client ID</label>
        <input v-model="localUPSClientID" class="form-input" placeholder="UPS client ID" />
      </div>
      <div class="form-group">
        <label class="form-label">UPS Client Secret</label>
        <input v-model="localUPSClientSecret" class="form-input" type="password" placeholder="UPS client secret" />
      </div>
      <div class="form-group">
        <label class="form-label">UPS Scope</label>
        <input v-model="localUPSScope" class="form-input" placeholder="Optional OAuth scope" />
      </div>
      <div class="form-group">
        <label class="form-label">FedEx API Base URL</label>
        <input v-model="localFedExAPIBaseURL" class="form-input" type="url" placeholder="https://apis.fedex.com" />
      </div>
      <div class="form-group">
        <label class="form-label">FedEx Token URL</label>
        <input v-model="localFedExTokenURL" class="form-input" type="url" placeholder="https://apis.fedex.com/oauth/token" />
      </div>
      <div class="form-group">
        <label class="form-label">FedEx Client ID</label>
        <input v-model="localFedExClientID" class="form-input" placeholder="FedEx client ID" />
      </div>
      <div class="form-group">
        <label class="form-label">FedEx Client Secret</label>
        <input v-model="localFedExClientSecret" class="form-input" type="password" placeholder="FedEx client secret" />
      </div>
      <div class="form-group">
        <label class="form-label">FedEx Scope</label>
        <input v-model="localFedExScope" class="form-input" placeholder="Optional OAuth scope" />
      </div>
      <div class="form-group">
        <label class="form-label">Log Level</label>
        <select v-model="localLogLevel" class="form-select">
          <option v-for="level in logLevels" :key="level" :value="level">{{ level }}</option>
        </select>
      </div>
      <p
        v-if="msg"
        class="text-body my-2"
        :class="error ? 'text-[#e74c3c]' : 'text-gold'"
      >{{ msg }}</p>
      <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">
        {{ saving ? 'Saving...' : 'Save System Settings' }}
      </button>
    </form>
    <div class="flex items-center gap-2 mt-6 pt-4 border-t border-border-subtle text-[0.78rem] text-text-muted">
      <span class="font-semibold uppercase tracking-[0.05em]">Version</span>
      <span class="font-mono text-text-secondary">{{ appVersion }}</span>
      <span v-if="buildDate" class="ml-1">Built {{ buildDate }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  numistaApiKey: string
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

defineEmits<{
  save: [settings: {
    numistaApiKey: string
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

watch(() => props.numistaApiKey, (v) => { localNumistaApiKey.value = v })
watch(() => props.pushoverAppToken, (v) => { localPushoverAppToken.value = v })
watch(() => props.publicAppUrl, (v) => { localPublicAppUrl.value = v })
watch(() => props.uspsApiBaseUrl, (v) => { localUSPSAPIBaseURL.value = v })
watch(() => props.uspsApiKey, (v) => { localUSPSAPIKey.value = v })
watch(() => props.uspsApiKeyHeader, (v) => { localUSPSAPIKeyHeader.value = v })
watch(() => props.upsApiBaseUrl, (v) => { localUPSAPIBaseURL.value = v })
watch(() => props.upsTokenUrl, (v) => { localUPSTokenURL.value = v })
watch(() => props.upsClientId, (v) => { localUPSClientID.value = v })
watch(() => props.upsClientSecret, (v) => { localUPSClientSecret.value = v })
watch(() => props.upsScope, (v) => { localUPSScope.value = v })
watch(() => props.fedexApiBaseUrl, (v) => { localFedExAPIBaseURL.value = v })
watch(() => props.fedexTokenUrl, (v) => { localFedExTokenURL.value = v })
watch(() => props.fedexClientId, (v) => { localFedExClientID.value = v })
watch(() => props.fedexClientSecret, (v) => { localFedExClientSecret.value = v })
watch(() => props.fedexScope, (v) => { localFedExScope.value = v })
watch(() => props.logLevel, (v) => { localLogLevel.value = v })
</script>
