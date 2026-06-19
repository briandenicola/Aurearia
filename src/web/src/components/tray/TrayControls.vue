<template>
  <div class="tray-controls">
    <div class="drawer-navigation">
      <button
        class="nav-btn"
        :disabled="drawerIndex === 0"
        @click="emit('prev')"
      >
        <ChevronLeft :size="16" />
        Prev
      </button>
      <span class="drawer-label">
        Tray {{ drawerIndex + 1 }} of {{ totalDrawers }}
      </span>
      <button
        class="nav-btn"
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
}

defineProps<Props>()
const emit = defineEmits<{
  prev: []
  next: []
}>()
</script>

<style scoped>
.tray-controls {
  position: fixed;
  left: 50%;
  bottom: calc(1rem + env(safe-area-inset-bottom));
  z-index: 1200;
  display: flex;
  flex-direction: column;
  width: min(calc(100vw - 2rem), 420px);
  transform: translateX(-50%);
}

.drawer-navigation {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: nowrap;
  gap: 1rem;
  padding: 0.6rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
}

.drawer-label {
  font-size: 0.9rem;
  color: var(--text-primary);
  flex: 0 0 auto;
  min-width: 120px;
  text-align: center;
}

.nav-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.5rem 1.25rem;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  background: var(--bg-card);
  color: var(--text-secondary);
  font-size: 0.85rem;
  cursor: pointer;
  white-space: nowrap;
  transition: all var(--transition-fast);
}

.nav-btn:hover:not(:disabled) {
  background: var(--accent-gold-dim);
  color: var(--text-primary);
}

.nav-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

@media (max-width: 575px) {
  .drawer-navigation {
    gap: 0.5rem;
    padding: 0.45rem;
  }

  .nav-btn {
    padding: 0.45rem 0.8rem;
  }
}
</style>
