<template>
  <div class="chat-header">
    <h1><Bot :size="20" /> Coin Search Agent</h1>
    <div class="chat-header-actions">
      <button
        v-if="hasMessages"
        class="chat-save"
        :disabled="saving"
        @click="$emit('save')"
        :title="conversationId ? 'Update saved conversation' : 'Save conversation'"
      >
        <Save :size="16" />
        {{ saveLabel }}
      </button>
      <button class="chat-close" @click="$emit('close')"><X :size="18" /></button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bot, Save, X } from 'lucide-vue-next'

defineProps<{
  hasMessages: boolean
  saving: boolean
  conversationId: number | null
  saveLabel: string
}>()

defineEmits<{
  save: []
  close: []
}>()
</script>

<style scoped>
.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.chat-header h1 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.4rem;
  margin: 0;
  color: var(--accent-gold);
}

.chat-header-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.chat-save {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  background: none;
  border: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.3rem 0.6rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  transition: all var(--transition-fast);
}

.chat-save:hover:not(:disabled) {
  color: var(--accent-gold);
  border-color: var(--accent-gold);
}

.chat-save:disabled {
  opacity: 0.5;
  cursor: default;
}

.chat-close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: all var(--transition-fast);
}

.chat-close:hover {
  color: var(--text-primary);
  background: var(--bg-card);
}
</style>
