<template>
  <section class="mb-6 rounded-md border border-border-subtle bg-card p-6">
    <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <h2>Value Trend</h2>
      <select :value="range" class="rounded-sm border border-border-subtle bg-input px-[0.6rem] py-[0.4rem] text-base text-text-primary" @change="$emit('update:range', ($event.target as HTMLSelectElement).value)">
        <option value="1m">1 month</option>
        <option value="3m">3 months</option>
        <option value="1y">1 year</option>
        <option value="all">All</option>
      </select>
    </div>
    <div v-if="snapshots.length" class="flex flex-col gap-2">
      <div
        v-for="snapshot in snapshots"
        :key="snapshot.snapshotDate"
        class="flex justify-between border-b border-border-subtle pb-[0.4rem] text-body text-text-secondary last:border-b-0"
      >
        <span>{{ formatDate(snapshot.snapshotDate) }}</span>
        <strong class="font-semibold text-gold">${{ snapshot.totalValue.toFixed(2) }}</strong>
      </div>
    </div>
    <p v-else class="m-0 text-body text-text-secondary">No snapshots yet. Capture one to start tracking trends.</p>
  </section>
</template>

<script setup lang="ts">
import type { CoinSetSnapshot } from '@/types'

defineProps<{
  snapshots: CoinSetSnapshot[]
  range: string
}>()

defineEmits<{
  'update:range': [value: string]
}>()

function formatDate(value: string) {
  return new Date(value).toLocaleDateString()
}
</script>
