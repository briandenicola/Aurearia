<template>
  <div class="criteria-summary">
    <span class="status" :class="{ inactive: !alert.isActive }">{{ alert.isActive ? 'Active' : 'Disabled' }}</span>
    <span class="cadence">{{ cadenceLabel }}</span>
    <span v-for="item in criteriaItems" :key="item" class="criterion">{{ item }}</span>
    <span v-if="!criteriaItems.length" class="muted">No criteria</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WishlistSearchAlert } from '@/types'

const props = defineProps<{ alert: WishlistSearchAlert }>()

const cadenceLabel = computed(() => `Cadence: ${props.alert.cadence}`)
const criteriaItems = computed(() => {
  const a = props.alert
  const items = [
    a.rulerOrIssuer,
    a.coinType,
    a.mint,
    a.material,
    a.gradeOrCondition,
    a.keywords,
    ...a.sourceFilters,
  ].filter(Boolean)
  if (a.priceMin != null || a.priceMax != null) items.push(`Price ${a.priceMin ?? 0}–${a.priceMax ?? 'any'} ${a.currency}`)
  if (a.dateFrom != null || a.dateTo != null) items.push(`Date ${a.dateFrom ?? 'any'}–${a.dateTo ?? 'any'}`)
  return items
})
</script>

<style scoped>
.criteria-summary { display: flex; flex-wrap: wrap; gap: .4rem; align-items: center; }
.status, .cadence, .criterion { border: 1px solid var(--border-subtle); border-radius: var(--radius-full); padding: .15rem .55rem; font-size: 0.8rem; }
.status { color: var(--success); }
.status.inactive { color: var(--text-muted); }
.cadence { color: var(--accent-gold); }
.criterion { color: var(--text-secondary); background: var(--bg-card); }
.muted { color: var(--text-muted); }
</style>
