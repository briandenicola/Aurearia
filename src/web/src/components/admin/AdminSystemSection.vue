<template>
  <section class="admin-section card">
    <h2>System Settings</h2>
    <form @submit.prevent="$emit('save', { numistaApiKey: localNumistaApiKey, logLevel: localLogLevel })">
      <div class="form-group">
        <label class="form-label">Numista API Key</label>
        <input v-model="localNumistaApiKey" class="form-input" type="password" placeholder="Enter your Numista API key" />
        <span class="form-hint">Get a free key at <a href="https://en.numista.com/api/" target="_blank" rel="noopener">numista.com/api</a> (2,000 requests/month free)</span>
      </div>
      <div class="form-group">
        <label class="form-label">Log Level</label>
        <select v-model="localLogLevel" class="form-select">
          <option v-for="level in logLevels" :key="level" :value="level">{{ level }}</option>
        </select>
      </div>
      <p v-if="msg" class="msg" :class="{ error }">{{ msg }}</p>
      <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">
        {{ saving ? 'Saving...' : 'Save System Settings' }}
      </button>
    </form>
    <div class="version-info">
      <span class="version-label">Version</span>
      <span class="version-value">{{ appVersion }}</span>
      <span v-if="buildDate" class="version-date">Built {{ buildDate }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  numistaApiKey: string
  logLevel: string
  logLevels: readonly string[]
  saving: boolean
  msg: string
  error: boolean
  appVersion: string
  buildDate: string
}>()

defineEmits<{
  save: [settings: { numistaApiKey: string; logLevel: string }]
}>()

const localNumistaApiKey = ref(props.numistaApiKey)
const localLogLevel = ref(props.logLevel)

watch(() => props.numistaApiKey, (v) => { localNumistaApiKey.value = v })
watch(() => props.logLevel, (v) => { localLogLevel.value = v })
</script>

<style scoped>
.msg {
  font-size: 0.85rem;
  color: var(--accent-gold);
  margin: 0.5rem 0;
}

.msg.error {
  color: #e74c3c;
}

.version-info {
  margin-top: 1.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.78rem;
  color: var(--text-muted);
}

.version-label {
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.version-value {
  font-family: 'Courier New', Courier, monospace;
  color: var(--text-secondary);
}

.version-date {
  margin-left: 0.25rem;
}
</style>
