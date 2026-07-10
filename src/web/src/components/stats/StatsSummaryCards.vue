<template>
  <div class="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(200px,1fr))]">
    <div class="flex flex-col gap-1 rounded-md border border-border-subtle bg-card p-6 text-center">
      <span class="font-display text-2xl font-semibold text-text-primary">{{ stats.totalCoins }}</span>
      <span class="text-sm uppercase tracking-[0.08em] text-text-muted">Coins Owned</span>
    </div>
    <div class="flex flex-col gap-1 rounded-md border border-border-subtle bg-card p-6 text-center">
      <span class="font-display text-2xl font-semibold text-text-primary">{{ stats.totalWishlist }}</span>
      <span class="text-sm uppercase tracking-[0.08em] text-text-muted">On Wishlist</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { StatsResponse } from '@/types'

defineProps<{
  stats: StatsResponse
}>()
</script>
