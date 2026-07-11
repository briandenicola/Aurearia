<template>
  <Transition name="drawer-slide">
    <aside
      v-if="open && group"
      class="fixed inset-x-3 bottom-3 z-[1100] max-h-[70vh] overflow-y-auto rounded-md border border-border-subtle bg-card p-4 shadow-card md:top-20 md:right-4 md:bottom-4 md:left-auto md:max-h-none md:w-[min(380px,calc(100vw-2rem))]"
      role="dialog"
      aria-modal="true"
      :aria-labelledby="titleId"
    >
      <header class="mb-4 flex items-start justify-between gap-3">
        <div>
          <p class="section-label">Selected Mint</p>
          <h2 :id="titleId" class="mt-1">{{ group.mint.displayName }}</h2>
          <p class="m-0 text-body text-text-secondary">{{ group.count }} {{ group.count === 1 ? 'coin' : 'coins' }} in this view</p>
        </div>
        <button class="btn btn-sm btn-ghost" type="button" aria-label="Close mint drawer" @click="$emit('close')">
          <X :size="16" />
        </button>
      </header>

      <ul class="m-0 flex list-none flex-col gap-3 p-0">
        <li v-for="coin in group.coins" :key="coin.id" class="rounded-sm border border-border-subtle bg-input">
          <router-link :to="`/coin/${coin.id}`" class="flex flex-col gap-1 rounded-sm p-3 text-text-primary transition-colors hover:text-gold focus-visible:text-gold">
            <span class="font-semibold">{{ coin.name }}</span>
            <span class="text-chip text-text-secondary">{{ coin.ruler || 'Unknown ruler' }} · {{ coin.denomination || 'Unknown denomination' }}</span>
          </router-link>
        </li>
      </ul>
    </aside>
  </Transition>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue'
import { X } from 'lucide-vue-next'
import type { MintGroup } from '@/utils/mintMap'

const props = defineProps<{
  group: MintGroup | null
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const titleId = computed(() => `mint-drawer-${props.group?.mint.id ?? 'empty'}`)

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}

watch(() => props.open, (open) => {
  if (open) {
    document.addEventListener('keydown', handleKeydown)
  } else {
    document.removeEventListener('keydown', handleKeydown)
  }
}, { immediate: true })

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<style scoped>
.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform var(--transition-med), opacity var(--transition-med);
}

.drawer-slide-enter-from,
.drawer-slide-leave-to {
  opacity: 0;
  transform: translateX(1rem);
}
</style>
