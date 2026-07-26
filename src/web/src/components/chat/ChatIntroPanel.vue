<template>
  <div class="flex shrink items-center overflow-y-auto p-4 text-center text-text-secondary">
    <div class="flex w-full flex-col items-center gap-3">
      <Bot :size="32" />
      <p class="max-w-[300px] leading-relaxed">
        Search for coins, find upcoming shows, or get a portfolio analysis -- ask me anything about collecting.
      </p>
      <div class="mt-2 flex w-full flex-col gap-2">
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send', 'Find me Roman silver denarii of Julius Caesar')"
        >
          Roman denarii of Julius Caesar
        </button>
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send', 'I\'m looking for Byzantine gold solidi under $1000')"
        >
          Byzantine gold solidi under $1000
        </button>
        <button
          class="w-full rounded-sm border border-border-subtle bg-card px-3 py-2 text-left text-chip text-text-secondary transition-colors hover:border-border-accent hover:text-gold"
          @click="$emit('send', 'Show me ancient Greek tetradrachms from Athens')"
        >
          Greek tetradrachms from Athens
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
