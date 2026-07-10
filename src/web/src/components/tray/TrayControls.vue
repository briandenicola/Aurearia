<template>
  <div
    class="z-[1200] flex w-[min(calc(100vw-2rem),420px)] flex-col"
    :class="fixed
      ? 'fixed bottom-[calc(1rem+env(safe-area-inset-bottom))] left-1/2 -translate-x-1/2'
      : 'static mx-auto w-[min(100%,420px)] translate-x-0'"
  >
    <div class="flex flex-nowrap items-center justify-center gap-4 rounded-full border border-border-subtle bg-card p-[0.6rem] shadow-card max-[575px]:gap-2 max-[575px]:p-[0.45rem]">
      <button
        class="inline-flex items-center gap-[0.35rem] whitespace-nowrap rounded-full border border-border-accent bg-card px-5 py-2 text-body text-text-secondary transition-all hover:bg-[var(--accent-gold-dim)] hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-30 max-[575px]:px-3 max-[575px]:py-[0.45rem]"
        :disabled="drawerIndex === 0"
        @click="emit('prev')"
      >
        <ChevronLeft :size="16" />
        Prev
      </button>
      <span class="min-w-[120px] flex-none text-center text-base text-text-primary">
        Tray {{ drawerIndex + 1 }} of {{ totalDrawers }}
      </span>
      <button
        class="inline-flex items-center gap-[0.35rem] whitespace-nowrap rounded-full border border-border-accent bg-card px-5 py-2 text-body text-text-secondary transition-all hover:bg-[var(--accent-gold-dim)] hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-30 max-[575px]:px-3 max-[575px]:py-[0.45rem]"
        :disabled="drawerIndex >= totalDrawers - 1"
        @click="emit('next')"
      >
        Next
        <ChevronRight :size="16" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

interface Props {
  drawerIndex: number
  totalDrawers: number
  fixed?: boolean
}

withDefaults(defineProps<Props>(), {
  fixed: true,
})
const emit = defineEmits<{
  prev: []
  next: []
}>()
</script>
