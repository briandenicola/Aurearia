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
        <span class="set-count-label">{{ set.coinCount === 1 ? 'Coin' : 'Coins' }}</span>
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
  return props.set.coinCount > 0 ? 'Curated group' : 'Ready for coins'
})
</script>

<style scoped>
.set-dashboard-card {
  width: 100%;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 1.15rem;
  min-height: 5rem;
  padding: 0.75rem 1.1rem;
  background: var(--bg-card);
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-md);
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
  width: 0.45rem;
  height: 4rem;
  border-radius: var(--radius-full);
  box-shadow: 0 0 16px var(--accent-gold-glow);
}

.set-card-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.set-name {
  color: var(--text-heading);
  font-family: 'Cinzel', serif;
  font-size: 1.35rem;
  font-weight: 500;
  letter-spacing: 0.02em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.set-description {
  color: var(--text-muted);
  font-size: 0.95rem;
}

.set-card-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.85rem;
}

.set-count {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0;
  white-space: nowrap;
  min-width: 4.25rem;
}

.set-count-value {
  color: var(--accent-gold);
  font-size: 2.75rem;
  font-weight: 700;
  line-height: 0.85;
  font-family: 'Inter', sans-serif;
}

.set-count-label {
  color: var(--accent-gold);
  font-size: 0.9rem;
  font-weight: 600;
}

@media (max-width: 560px) {
  .set-dashboard-card {
    gap: 0.75rem;
    min-height: 4.5rem;
    padding: 0.65rem 0.85rem;
  }

  .set-accent {
    height: 3.5rem;
  }

  .set-name {
    font-size: 1.1rem;
  }

  .set-description {
    font-size: 0.85rem;
  }

  .set-count {
    min-width: 3.5rem;
  }

  .set-count-value {
    font-size: 2.25rem;
  }

  .set-count-label {
    font-size: 0.75rem;
  }
}
</style>
