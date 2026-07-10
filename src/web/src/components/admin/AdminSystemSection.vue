<template>
  <section class="card">
    <h2 class="text-xl font-medium mb-5 pb-3 border-b border-border-subtle">System Settings</h2>
    <form @submit.prevent="$emit('save', { numistaApiKey: localNumistaApiKey, logLevel: localLogLevel, pushoverAppToken: localPushoverAppToken, publicAppUrl: localPublicAppUrl })">
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
  logLevel: string
  logLevels: readonly string[]
  saving: boolean
  msg: string
  error: boolean
  appVersion: string
  buildDate: string
}>()

defineEmits<{
  save: [settings: { numistaApiKey: string; logLevel: string; pushoverAppToken: string; publicAppUrl: string }]
}>()

const localNumistaApiKey = ref(props.numistaApiKey)
const localPushoverAppToken = ref(props.pushoverAppToken)
const localPublicAppUrl = ref(props.publicAppUrl)
const localLogLevel = ref(props.logLevel)

watch(() => props.numistaApiKey, (v) => { localNumistaApiKey.value = v })
watch(() => props.pushoverAppToken, (v) => { localPushoverAppToken.value = v })
watch(() => props.publicAppUrl, (v) => { localPublicAppUrl.value = v })
watch(() => props.logLevel, (v) => { localLogLevel.value = v })
</script>
