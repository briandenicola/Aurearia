<template>
  <div class="flex shrink items-center overflow-y-auto p-4 text-center text-text-secondary">
    <div class="flex w-full flex-col items-center gap-3">
      <Bot :size="32" />
      <p class="max-w-[300px] leading-relaxed">
        Ask about your collection, research coins, or turn an answer into a markdown note.
      </p>
      <div class="mt-2 flex w-full flex-col gap-2">
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send', 'Which coins in my collection are missing an era?')"
        >
          Coins missing an era
        </button>
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send', 'Summarize my most valuable coins and explain why they stand out')"
        >
          Summarize valuable coins
        </button>
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send', 'Research Athenian owl tetradrachms and include key attribution notes')"
        >
          Research Athenian owls
        </button>
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send', 'What ancient coin shows are coming up near me?')"
        >
          Upcoming coin shows near me
        </button>
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send-portfolio')"
        >
          Analyze my portfolio
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bot } from 'lucide-vue-next'

defineEmits<{
  send: [message: string]
  'send-portfolio': []
}>()
</script>
