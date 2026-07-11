<template>
  <div class="flex shrink-0 items-center justify-between border-b border-border-subtle px-4 py-3">
    <h1 class="m-0 flex items-center gap-2 text-lg text-gold">
      <Bot :size="20" />
      Coin Search Agent
    </h1>
    <div class="flex items-center gap-2">
      <button
        v-if="hasMessages"
        class="btn btn-ghost btn-xs disabled:cursor-default disabled:opacity-50"
        :disabled="saving"
        @click="$emit('save')"
        :title="conversationId ? 'Update saved conversation' : 'Save conversation'"
      >
        <Save :size="16" />
        {{ saveLabel }}
      </button>
      <button
        class="inline-flex cursor-pointer items-center rounded-sm p-1 text-text-muted transition-colors hover:bg-card hover:text-text-primary"
        @click="$emit('close')"
      >
        <X :size="18" />
      </button>
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
