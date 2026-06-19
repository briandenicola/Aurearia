<template>
  <button class="set-dashboard-card" type="button" @click="$emit('click')">
    <span class="set-accent" :style="{ backgroundColor: set.color }" aria-hidden="true"></span>
    <span class="set-card-main">
      <span class="set-name">{{ set.name }}</span>
      <span v-if="setDescription" class="set-description">{{ setDescription }}</span>
    </span>
    <span class="set-card-meta">
      <span class="set-count">
        <span class="set-count-value">{{ set.coinCount }}</span>
        <span class="set-count-label">{{ set.coinCount === 1 ? 'coin' : 'coins' }}</span>
      </span>
      <span v-if="set.completionPercentage != null" class="completion-meter" aria-label="Completion">
        <span class="completion-track">
          <span class="completion-fill" :style="{ width: `${set.completionPercentage}%` }"></span>
        </span>
        <span class="completion-label">{{ set.completionPercentage }}%</span>
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CoinSetSummary } from '@/types'

const props = defineProps<{
  set: CoinSetSummary
}>()

defineEmits<{
  (e: 'click'): void
}>()

const setDescription = computed(() => {
  if (props.set.completionPercentage != null) return 'Completion set'
  return props.set.coinCount > 0 ? 'Curated group' : 'Ready for coins'
})
</script>

<style scoped>
.set-dashboard-card {
  width: 100%;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.85rem;
  padding: 0.85rem 1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: inherit;
  cursor: pointer;
  text-align: left;
  transition: border-color var(--transition-fast), background var(--transition-fast), box-shadow var(--transition-fast);
}

.set-dashboard-card:hover {
  border-color: var(--border-accent);
  background: var(--bg-card-hover);
  box-shadow: var(--shadow-card);
}

.set-accent {
  width: 0.25rem;
  height: 2.8rem;
  border-radius: var(--radius-full);
  box-shadow: 0 0 16px var(--accent-gold-glow);
}

.set-card-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.set-name {
  color: var(--text-heading);
  font-family: 'Cinzel', serif;
  font-size: 1rem;
  font-weight: 500;
  letter-spacing: 0.02em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.set-description {
  color: var(--text-muted);
  font-size: 0.8rem;
}

.set-card-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.85rem;
}

.set-count {
  display: flex;
  align-items: baseline;
  gap: 0.25rem;
  white-space: nowrap;
}

.set-count-value {
  color: var(--accent-gold);
  font-size: 1rem;
  font-weight: 600;
}

.set-count-label,
.completion-label {
  color: var(--text-muted);
  font-size: 0.75rem;
}

.completion-meter {
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.completion-track {
  width: 3.5rem;
  height: 0.25rem;
  overflow: hidden;
  border-radius: var(--radius-full);
  background: var(--bg-input);
}

.completion-fill {
  display: block;
  height: 100%;
  border-radius: var(--radius-full);
  background: var(--accent-gold);
}

@media (max-width: 560px) {
  .set-dashboard-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .set-card-meta {
    grid-column: 2;
    justify-content: space-between;
  }
}
</style>
