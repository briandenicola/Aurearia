<template>
  <section class="card">
    <h2 class="text-[1.1rem] font-medium mb-5 pb-3 border-b border-border-subtle">Saved Conversations</h2>
    <p class="text-sm text-text-muted mb-4">
      Your saved AI coin search conversations. Open one to continue the search or review results.
    </p>

    <div v-if="loading" class="text-text-muted italic py-2">Loading...</div>

    <div v-else-if="conversations.length" class="flex flex-col gap-2 mt-4">
      <div
        v-for="conv in conversations"
        :key="conv.id"
        class="flex justify-between items-center py-[0.6rem] border-b border-border-subtle last:border-0 gap-3"
      >
        <div class="flex flex-col gap-[0.1rem] min-w-0 cursor-pointer" @click="$emit('open', conv.id)">
          <span class="text-base font-medium">{{ conv.title }}</span>
          <span class="text-sm text-text-muted">{{ formatDate(conv.updatedAt) }}</span>
        </div>
        <div class="flex gap-2 shrink-0">
          <button class="btn btn-secondary btn-sm" @click="$emit('open', conv.id)">Open</button>
          <button class="btn btn-danger btn-sm" @click="$emit('delete', conv.id)">Delete</button>
        </div>
      </div>
    </div>
    <p v-else class="text-sm text-text-muted mt-2">
      No saved conversations yet. Use the Save button in the coin search chat to save a conversation.
    </p>
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

