<template>
  <section class="mb-6 rounded-md border border-border-subtle bg-card p-6">
    <div class="mb-4 flex items-center justify-between gap-4">
      <div>
        <h2>Completion</h2>
        <p class="m-0 text-body text-text-secondary">
          {{ collectionItems }} collection item<span v-if="collectionItems !== 1">s</span> of {{ totalItems }} total item<span v-if="totalItems !== 1">s</span>
        </p>
        <p class="m-0 text-chip text-text-muted">{{ wishlistItems }} wishlist item<span v-if="wishlistItems !== 1">s</span></p>
      </div>
      <span class="text-xl font-semibold text-gold">{{ completionPercent.toFixed(1) }}%</span>
    </div>
    <div class="h-[0.6rem] overflow-hidden rounded-full bg-input">
      <div class="h-full bg-gold transition-[width] duration-300" :style="{ width: `${Math.min(completionPercent, 100)}%` }"></div>
    </div>
    <div v-if="completion.missingTargets.length" class="mt-4">
      <span class="section-label">Wishlist items still needed</span>
      <div class="mt-2 flex flex-wrap gap-[0.35rem]">
        <span v-for="target in completion.missingTargets" :key="target.id || target.label" class="chip-sm">
          {{ target.label }}
        </span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CoinSetCompletion } from '@/types'

const props = defineProps<{
  completion: CoinSetCompletion
}>()

const collectionItems = computed(() => props.completion.collectionItems ?? props.completion.completedTargets)
const totalItems = computed(() => props.completion.totalTargets || (collectionItems.value + wishlistItems.value))
const wishlistItems = computed(() => {
  if (props.completion.wishlistItems != null) return props.completion.wishlistItems
  const total = props.completion.totalTargets ?? 0
  return Math.max(total - collectionItems.value, 0)
})
const completionPercent = computed(() => {
  if (totalItems.value <= 0) return 0
  if (props.completion.collectionItems != null || props.completion.wishlistItems != null) {
    return (collectionItems.value / totalItems.value) * 100
  }
  return props.completion.completionPercentage
})
</script>
