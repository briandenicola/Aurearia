<template>
  <div class="mb-6">
    <div class="mb-3 flex gap-2">
      <input
        v-model="input"
        type="text"
        class="form-input flex-1"
        placeholder="e.g. Cleaned, sent to grading, displayed at show..."
        @keyup.enter="handleAdd"
      />
      <button class="btn btn-primary btn-sm" :disabled="!input.trim()" @click="handleAdd">Add</button>
    </div>
    <div v-if="entries.length" class="flex flex-col gap-[0.4rem]" :class="{ 'max-h-[16.5rem] overflow-y-auto pr-1': entries.length > 3 }">
      <div v-for="entry in entries" :key="entry.id" class="flex items-start justify-between gap-2 rounded-sm border border-border-subtle bg-card px-3 py-2">
        <div class="flex min-w-0 flex-col gap-[0.1rem]">
          <span class="text-body text-text-primary">{{ entry.entry }}</span>
          <span class="text-label text-text-muted">{{ formatJournalDate(entry.createdAt) }}</span>
        </div>
        <button class="btn btn-ghost btn-xs" @click="$emit('delete', entry.id)">✕</button>
      </div>
    </div>
    <p v-else class="text-body italic text-text-muted">No activity recorded yet.</p>
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
