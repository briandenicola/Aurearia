<template>
  <div class="card !px-8 !py-12 flex flex-col items-center gap-4 text-center max-md:!px-6 max-md:!py-8">
    <div class="mb-2 text-text-muted opacity-60">
      <FileQuestion :size="48" />
    </div>
    <h3 class="m-0 text-lg text-heading">No Active Coins to Score</h3>
    <p class="m-0 max-w-[500px] text-base leading-[1.6] text-text-secondary">
      Collection health scoring requires active coins (not wishlist or sold items).
    </p>
    <p class="m-0 max-w-[500px] text-body leading-[1.6] text-text-muted">
      Add coins to your collection or mark wishlist items as acquired to start tracking
      collection health metrics.
    </p>
    <router-link to="/add" class="btn btn-primary mt-2">
      <CirclePlus :size="16" /> Add Your First Coin
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { FileQuestion, CirclePlus } from 'lucide-vue-next'
</script>
