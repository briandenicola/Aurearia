<template>
  <Transition name="drawer-slide">
    <aside v-if="open && group" class="mint-drawer" role="dialog" aria-modal="true" :aria-labelledby="titleId">
      <header class="drawer-header">
        <div>
          <p class="section-label">Selected Mint</p>
          <h2 :id="titleId">{{ group.mint.displayName }}</h2>
          <p class="drawer-summary">{{ group.count }} {{ group.count === 1 ? 'coin' : 'coins' }} in this view</p>
        </div>
        <button class="btn btn-sm btn-ghost" type="button" aria-label="Close mint drawer" @click="$emit('close')">
          <X :size="16" />
        </button>
      </header>

      <ul class="coin-list">
        <li v-for="coin in group.coins" :key="coin.id" class="coin-item">
          <router-link class="coin-link" :to="`/coin/${coin.id}`">
            <span class="coin-name">{{ coin.name }}</span>
            <span class="coin-meta">{{ coin.ruler || 'Unknown ruler' }} · {{ coin.denomination || 'Unknown denomination' }}</span>
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
.mint-drawer {
  position: fixed;
  right: 1rem;
  top: 5rem;
  bottom: 1rem;
  z-index: 300;
  width: min(380px, calc(100vw - 2rem));
  overflow-y: auto;
  padding: 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
}

.drawer-header {
  display: flex;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.drawer-header h2 {
  margin: 0.2rem 0;
}

.drawer-summary {
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.coin-list {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin: 0;
  padding: 0;
}

.coin-item {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
}

.coin-link {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.75rem;
  color: var(--text-primary);
  text-decoration: none;
}

.coin-link:hover,
.coin-link:focus-visible {
  color: var(--accent-gold);
}

.coin-name {
  font-weight: 600;
}

.coin-meta {
  color: var(--text-secondary);
  font-size: 0.8rem;
}

.drawer-slide-enter-active,
.drawer-slide-leave-active {
  transition: transform var(--transition-med), opacity var(--transition-med);
}

.drawer-slide-enter-from,
.drawer-slide-leave-to {
  opacity: 0;
  transform: translateX(1rem);
}

@media (max-width: 768px) {
  .mint-drawer {
    left: 0.75rem;
    right: 0.75rem;
    top: auto;
    bottom: 0.75rem;
    width: auto;
    max-height: 70vh;
  }
}
</style>
