<template>
  <section class="admin-section card">
    <h2>Application Logs</h2>
    <div class="logs-toolbar">
      <select :value="filter" class="form-select logs-filter" @change="$emit('update:filter', ($event.target as HTMLSelectElement).value); $emit('load')">
        <option value="">All Levels</option>
        <option v-for="level in ['TRACE','DEBUG','INFO','WARN','ERROR']" :key="level" :value="level">{{ level }}</option>
      </select>
      <button class="btn btn-secondary btn-sm" @click="$emit('load')" :disabled="loading">
        {{ loading ? 'Loading...' : 'Refresh' }}
      </button>
      <button
        class="btn btn-sm"
        :class="autoRefresh ? 'btn-primary' : 'btn-secondary'"
        @click="$emit('toggle-auto-refresh')"
      >
        {{ autoRefresh ? 'Auto ●' : 'Auto ○' }}
      </button>
      <button
        class="btn btn-secondary btn-sm"
        @click="$emit('export')"
        :disabled="logs.length === 0"
        title="Export logs as text file"
      >
        <Download :size="14" /> Export
      </button>
    </div>
    <div class="logs-container">
      <div v-if="logs.length === 0 && !loading" class="logs-empty">
        No log entries. Click Refresh to load.
      </div>
      <div
        v-for="(entry, i) in logs"
        :key="i"
        class="log-entry"
        :class="logLevelClass(entry.level)"
      >
        <span class="log-time">{{ entry.timestamp.substring(11, 19) }}</span>
        <span class="log-level-badge">{{ entry.level }}</span>
        <span class="log-msg">{{ entry.message }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { LogEntry } from '@/types'
import { Download } from 'lucide-vue-next'

defineProps<{
  logs: LogEntry[]
  loading: boolean
  filter: string
  autoRefresh: boolean
}>()

defineEmits<{
  load: []
  'toggle-auto-refresh': []
  export: []
  'update:filter': [val: string]
}>()

function logLevelClass(level: string) {
  switch (level) {
    case 'ERROR': return 'log-error'
    case 'WARN': return 'log-warn'
    case 'DEBUG': return 'log-debug'
    case 'TRACE': return 'log-trace'
    default: return 'log-info'
  }
}
</script>

<style scoped>
.logs-toolbar {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 1rem;
}

.logs-filter {
  width: auto;
  min-width: 120px;
}

.logs-container {
  max-height: 500px;
  overflow-y: auto;
  background: var(--bg-body);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.5rem;
  font-family: 'Courier New', Courier, monospace;
  font-size: 0.78rem;
  line-height: 1.5;
}

.logs-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-family: 'Inter', sans-serif;
}

.log-entry {
  display: flex;
  gap: 0.5rem;
  padding: 0.15rem 0.25rem;
  border-radius: 2px;
}

.log-entry:hover {
  background: var(--bg-card);
}

.log-time {
  color: var(--text-muted);
  flex-shrink: 0;
}

.log-level-badge {
  flex-shrink: 0;
  min-width: 48px;
  text-align: center;
  font-weight: 600;
  border-radius: 2px;
  padding: 0 4px;
}

.log-msg {
  word-break: break-word;
}

.log-error .log-level-badge { color: #e74c3c; }
.log-error .log-msg { color: #e74c3c; }
.log-warn .log-level-badge { color: #f39c12; }
.log-debug .log-level-badge { color: #3498db; }
.log-trace .log-level-badge { color: #7f8c8d; }
.log-info .log-level-badge { color: #2ecc71; }
</style>
