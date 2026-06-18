<template>
  <div class="tray-controls">
    <div class="drawer-navigation">
      <button
        class="btn btn-sm"
        :disabled="drawerIndex === 0"
        @click="emit('prev')"
      >
        &lt; Previous
      </button>
      <span class="drawer-label">
        Tray {{ drawerIndex + 1 }} of {{ totalDrawers }}
      </span>
      <button
        class="btn btn-sm"
        :disabled="drawerIndex >= totalDrawers - 1"
        @click="emit('next')"
      >
        Next &gt;
      </button>
    </div>
    <div class="felt-theme-selector">
      <span class="theme-label">Felt Color:</span>
      <div class="theme-chips">
        <button
          class="chip theme-chip theme-chip-red"
          :class="{ active: feltTheme === 'red' }"
          @click="emit('update:feltTheme', 'red')"
          aria-label="Red felt"
        >
          Red
        </button>
        <button
          class="chip theme-chip theme-chip-green"
          :class="{ active: feltTheme === 'green' }"
          @click="emit('update:feltTheme', 'green')"
          aria-label="Green felt"
        >
          Green
        </button>
        <button
          class="chip theme-chip theme-chip-navy"
          :class="{ active: feltTheme === 'navy' }"
          @click="emit('update:feltTheme', 'navy')"
          aria-label="Navy felt"
        >
          Navy
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { FeltColor } from '@/composables/useTrayPreference'

interface Props {
  drawerIndex: number
  totalDrawers: number
  feltTheme: FeltColor
}

defineProps<Props>()
const emit = defineEmits<{
  prev: []
  next: []
  'update:feltTheme': [theme: FeltColor]
}>()
</script>

<style scoped>
.tray-controls {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.drawer-navigation {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: nowrap;
  gap: 1rem;
}

.drawer-navigation .btn {
  white-space: nowrap;
}

.drawer-label {
  font-size: 0.9rem;
  color: var(--text-primary);
  flex: 0 0 auto;
  min-width: 120px;
  text-align: center;
}

.felt-theme-selector {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.theme-label {
  font-size: 0.85rem;
  color: var(--text-secondary);
}

.theme-chips {
  display: flex;
  gap: 0.35rem;
}

.theme-chip {
  font-size: 0.8rem;
  padding: 0.35rem 0.85rem;
  border: 1px solid var(--border-subtle);
  background: var(--bg-card);
  color: var(--text-secondary);
  transition: var(--transition-fast);
}

.theme-chip:hover {
  border-color: var(--border-accent);
  background: var(--bg-card-hover);
}

.theme-chip.active {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.theme-chip-red.active {
  background: var(--felt-red-dim);
  border-color: var(--felt-red-base);
  color: var(--felt-red-bright);
}

.theme-chip-green.active {
  background: var(--felt-green-dim);
  border-color: var(--felt-green-base);
  color: var(--felt-green-bright);
}

.theme-chip-navy.active {
  background: var(--felt-navy-dim);
  border-color: var(--felt-navy-base);
  color: var(--felt-navy-bright);
}

@media (max-width: 575px) {
  .tray-controls {
    gap: 0.75rem;
  }

  .drawer-navigation {
    gap: 0.5rem;
  }

  .felt-theme-selector {
    flex-direction: column;
    gap: 0.5rem;
  }
}
</style>
