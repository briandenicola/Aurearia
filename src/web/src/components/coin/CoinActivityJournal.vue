<template>
  <div class="journal-section">
    <h3>Activity Journal</h3>
    <div class="journal-add">
      <input
        v-model="input"
        type="text"
        class="form-input journal-input"
        placeholder="e.g. Cleaned, sent to grading, displayed at show..."
        @keyup.enter="handleAdd"
      />
      <button class="btn btn-primary btn-sm" :disabled="!input.trim()" @click="handleAdd">Add</button>
    </div>
    <div v-if="entries.length" class="journal-list">
      <div v-for="entry in entries" :key="entry.id" class="journal-entry">
        <div class="journal-entry-content">
          <span class="journal-entry-text">{{ entry.entry }}</span>
          <span class="journal-entry-date">{{ formatJournalDate(entry.createdAt) }}</span>
        </div>
        <button class="btn btn-ghost btn-xs" @click="$emit('delete', entry.id)">✕</button>
      </div>
    </div>
    <p v-else class="journal-empty">No activity recorded yet.</p>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { CoinJournal } from '@/types'

defineProps<{
  entries: CoinJournal[]
  coinId: number
}>()

const emit = defineEmits<{
  add: [entry: string]
  delete: [entryId: number]
}>()

const input = ref('')

function handleAdd() {
  if (!input.value.trim()) return
  emit('add', input.value.trim())
  input.value = ''
}

function formatJournalDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: 'short', day: 'numeric', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}
</script>

<style scoped>
.journal-section {
  margin-bottom: 1.5rem;
}

.journal-section h3 {
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.journal-add {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
}

.journal-input {
  flex: 1;
}

.journal-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.journal-entry {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 0.5rem;
  padding: 0.5rem 0.75rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
}

.journal-entry-content {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.journal-entry-text {
  font-size: 0.85rem;
}

.journal-entry-date {
  font-size: 0.7rem;
  color: var(--text-muted);
}

.journal-empty {
  font-size: 0.85rem;
  color: var(--text-muted);
  font-style: italic;
}

.btn-ghost {
  background: transparent;
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.btn-ghost:hover {
  color: #e74c3c;
  border-color: #e74c3c;
}

.btn-xs {
  padding: 0.15rem 0.45rem;
  font-size: 0.7rem;
  border-radius: var(--radius-sm);
}
</style>
