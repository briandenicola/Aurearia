<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <span
      :class="[
        'inline-flex rounded-full border px-2 py-0.5 text-chip',
        alert.isActive
          ? 'border-[rgba(74,222,128,0.35)] bg-[rgba(74,222,128,0.08)] text-green-400'
          : 'border-border-subtle text-text-muted',
      ]"
    >
      {{ alert.isActive ? 'Active' : 'Disabled' }}
    </span>
    <span class="inline-flex rounded-full border border-border-subtle px-2 py-0.5 text-chip text-gold">{{ cadenceLabel }}</span>
    <span
      v-for="item in criteriaItems"
      :key="item"
      class="inline-flex rounded-full border border-border-subtle bg-card px-2 py-0.5 text-chip text-text-secondary"
    >
      {{ item }}
    </span>
    <span v-if="!criteriaItems.length" class="text-chip text-text-muted">No criteria</span>
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
