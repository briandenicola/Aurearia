<template>
  <Teleport to="body">
    <div class="fixed right-3 bottom-3 z-[1200] flex max-w-[calc(100vw-1.5rem)] flex-col gap-2 sm:right-4 sm:bottom-4 sm:max-w-[min(22rem,calc(100vw-2rem))]" aria-live="polite" aria-atomic="true">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="flex items-center justify-between gap-3 rounded-sm border bg-card px-4 py-3 text-body text-text-primary shadow-card"
        :class="toast.kind === 'success' ? 'border-gold' : toast.kind === 'info' ? 'border-border-accent' : 'border-loss'"
        role="status"
      >
        <span>{{ toast.message }}</span>
        <button class="inline-flex shrink-0 items-center justify-center p-[0.15rem] text-text-secondary transition-colors hover:text-text-primary" type="button" aria-label="Dismiss notification" @click="removeToast(toast.id)">
          <X :size="14" />
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { X } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'

const { toasts, removeToast } = useToast()
</script>
