<template>
  <div class="metadata-table">
    <div
      v-for="row in rows"
      :key="row.key"
      class="metadata-row"
    >
      <span class="row-label">{{ row.label }}</span>
      <span class="row-value" :class="row.valueClass">{{ row.value }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CoinDetailMetadataRow } from '@/types'

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
</style>
