<template>
  <section class="settings-section card">
    <h2>Saved Conversations</h2>
    <p class="setting-desc" style="margin-bottom: 1rem">
      Your saved AI coin search conversations. Open one to continue the search or review results.
    </p>

    <div v-if="loading" class="loading-inline">Loading...</div>

    <div v-else-if="conversations.length" class="apikey-list">
      <div v-for="conv in conversations" :key="conv.id" class="apikey-item">
        <div class="apikey-item-info" style="cursor: pointer" @click="$emit('open', conv.id)">
          <span class="apikey-item-name">{{ conv.title }}</span>
          <span class="apikey-item-meta">{{ formatDate(conv.updatedAt) }}</span>
        </div>
        <div class="conv-actions">
          <button class="btn btn-secondary btn-sm" @click="$emit('open', conv.id)">Open</button>
          <button class="btn btn-danger btn-sm" @click="$emit('delete', conv.id)">Delete</button>
        </div>
      </div>
    </div>
    <p v-else class="setting-desc" style="margin-top: 0.5rem">No saved conversations yet. Use the Save button in the coin search chat to save a conversation.</p>
  </section>
</template>

<script setup lang="ts">
import type { ConversationSummary } from '@/api/client'

defineProps<{
  conversations: ConversationSummary[]
  loading: boolean
}>()

defineEmits<{
  open: [id: number]
  delete: [id: number]
}>()

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
  })
}
</script>

<style scoped>
.settings-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.loading-inline {
  color: var(--text-muted);
  font-style: italic;
  padding: 0.5rem 0;
}

.apikey-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.apikey-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.6rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 0.75rem;
}

.apikey-item:last-child {
  border-bottom: none;
}

.apikey-item-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.apikey-item-name {
  font-size: 0.9rem;
  font-weight: 500;
}

.apikey-item-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.conv-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.btn-danger {
  background: #e74c3c;
  color: #fff;
  border: none;
  cursor: pointer;
}

.btn-danger:hover {
  background: #c0392b;
}
</style>
