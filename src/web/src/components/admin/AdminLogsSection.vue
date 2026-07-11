<template>
  <section class="card">
    <h2 class="text-xl font-medium mb-5 pb-3 border-b border-border-subtle">Application Logs</h2>
    <div class="flex items-center gap-2 mb-4">
      <select
        :value="filter"
        class="form-select w-auto min-w-[120px]"
        @change="$emit('update:filter', ($event.target as HTMLSelectElement).value); $emit('load')"
      >
        <option value="">All Levels</option>
        <option v-for="level in ['TRACE','DEBUG','INFO','WARN','ERROR']" :key="level" :value="level">{{ level }}</option>
      </select>
      <button class="btn btn-secondary btn-sm" :disabled="loading" @click="$emit('load')">
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
        class="btn btn-secondary btn-sm inline-flex items-center gap-1"
        :disabled="logs.length === 0"
        title="Export logs as text file"
        @click="$emit('export')"
      >
        <Download :size="14" /> Export
      </button>
    </div>
    <div class="max-h-[500px] overflow-y-auto bg-surface border border-border-subtle rounded-sm p-2 font-mono text-chip leading-relaxed">
      <div v-if="logs.length === 0 && !loading" class="text-center py-8 text-text-muted font-sans">
        No log entries. Click Refresh to load.
      </div>
      <div
        v-for="(entry, i) in logs"
        :key="i"
        class="flex gap-2 px-1 py-[0.15rem] rounded-[2px] hover:bg-card"
        :class="logLevelClass(entry.level)"
      >
        <span class="text-text-muted shrink-0">{{ entry.timestamp.substring(11, 19) }}</span>
        <span class="shrink-0 min-w-[48px] text-center font-semibold rounded-[2px] px-1 log-level-badge">{{ entry.level }}</span>
        <span class="break-words">{{ entry.message }}</span>
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
/* Log level badge colors — too many color variants to express cleanly as Tailwind
   arbitrary values inside :class bindings; keep as scoped CSS for clarity. */
.log-error .log-level-badge,
.log-error .break-words { color: #e74c3c; }
.log-warn .log-level-badge { color: #f39c12; }
.log-debug .log-level-badge { color: #3498db; }
.log-trace .log-level-badge { color: #7f8c8d; }
.log-info .log-level-badge { color: #2ecc71; }
</style>
