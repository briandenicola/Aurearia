<template>
  <div class="metadata-table">
    <div
      v-for="row in rows"
      :key="row.key"
      class="metadata-row"
      :class="{ 'full-width': row.fullWidth }"
    >
      <span v-if="!row.fullWidth" class="row-label">{{ row.label }}</span>
      <span v-if="!row.url" class="row-value" :class="row.valueClass">{{ row.value }}</span>
      <SafeExternalLink v-else :href="row.url" class="row-value row-link" :class="row.valueClass">
        {{ row.value }}
      </SafeExternalLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CoinDetailMetadataRow } from '@/types'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

defineProps<{
  rows: CoinDetailMetadataRow[]
}>()
</script>

<style scoped>
.metadata-table {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 0.75rem 1rem;
}

.metadata-row.full-width {
  grid-column: 1 / -1;
}

.metadata-row.full-width .row-value {
  font-style: italic;
  color: var(--text-secondary);
}

.metadata-row.full-width .row-link {
  color: var(--accent-gold);
  transition: color var(--transition-fast);
}

.metadata-row.full-width .row-link:hover {
  color: var(--accent-bronze);
}
</style>
