<template>
  <section class="mint-list-panel" aria-label="Mint locations list">
    <header class="mint-list-header">
      <p class="section-label">Locations</p>
      <span class="mint-list-total">{{ groups.length }}</span>
    </header>

    <ul class="mint-list" role="list">
      <li v-for="group in groups" :key="group.mint.id">
        <button
          class="mint-list-item"
          :class="{ 'mint-list-item--selected': selectedMintId === group.mint.id }"
          type="button"
          :aria-pressed="selectedMintId === group.mint.id"
          :aria-label="`${group.mint.displayName}: ${group.count} ${group.count === 1 ? 'coin' : 'coins'}`"
          @click="$emit('select-mint', group)"
        >
          <div class="mint-list-item-top">
            <span class="mint-list-name">{{ group.mint.displayName }}</span>
            <span class="mint-list-badge">{{ group.count }}</span>
          </div>
          <p v-if="group.mint.region" class="mint-list-region">{{ group.mint.region }}</p>
          <p v-if="coinPreview(group)" class="mint-list-preview">{{ coinPreview(group) }}</p>
        </button>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import type { MintGroup } from '@/utils/mintMap'

const PREVIEW_MAX = 2

const props = defineProps<{
  groups: MintGroup[]
  selectedMintId?: number | null
}>()

defineEmits<{
  'select-mint': [group: MintGroup]
}>()

function coinPreview(group: MintGroup): string {
  const names = group.coins
    .slice(0, PREVIEW_MAX)
    .map((coin) => coin.name)
    .filter(Boolean)
  if (!names.length) return ''
  const more = group.count - names.length
  return more > 0 ? `${names.join(', ')} +${more} more` : names.join(', ')
}
</script>

<style scoped>
.mint-list-panel {
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.mint-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
  flex-shrink: 0;
}

.mint-list-total {
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  padding: 0.1rem 0.5rem;
  line-height: 1.4;
}

.mint-list {
  list-style: none;
  margin: 0;
  padding: 0.5rem;
  overflow-y: auto;
  flex: 1;
  min-height: 0; /* required: allows flex child to shrink and scroll within panel */
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  /* desktop: no max-height — grid row height bounds the panel naturally */
}

.mint-list-item {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  width: 100%;
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  padding: 0.6rem 0.75rem;
  cursor: pointer;
  transition: background var(--transition-fast), border-color var(--transition-fast);
  color: var(--text-primary);
}

.mint-list-item:hover,
.mint-list-item:focus-visible {
  background: var(--bg-card-hover);
  border-color: var(--border-subtle);
}

.mint-list-item--selected {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.mint-list-item--selected .mint-list-region,
.mint-list-item--selected .mint-list-preview {
  color: var(--accent-gold);
  opacity: 0.75;
}

.mint-list-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.mint-list-name {
  font-size: 0.9rem;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.mint-list-badge {
  flex-shrink: 0;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  padding: 0.1rem 0.45rem;
  line-height: 1.4;
}

.mint-list-item--selected .mint-list-badge {
  background: var(--accent-gold);
  color: var(--bg-card);
  border-color: var(--accent-gold);
}

.mint-list-region {
  margin: 0;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mint-list-preview {
  margin: 0;
  font-size: 0.8rem;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

@media (max-width: 768px) {
  .mint-list {
    /* on mobile the panel is full-width below the map; limit its own height */
    max-height: 280px;
  }
}
</style>
