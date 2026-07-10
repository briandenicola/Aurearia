<template>
  <section class="mb-6 rounded-md border border-border-subtle bg-card p-6">
    <div class="mb-4 flex items-center justify-between gap-4">
      <div>
        <h2>Completion</h2>
        <p class="m-0 text-body text-text-secondary">
          {{ completion.completedTargets }} of {{ completion.totalTargets }} targets complete
        </p>
      </div>
      <span class="text-xl font-semibold text-gold">{{ completion.completionPercentage.toFixed(1) }}%</span>
    </div>
    <div class="h-[0.6rem] overflow-hidden rounded-full bg-input">
      <div class="h-full bg-gold transition-[width] duration-300" :style="{ width: `${Math.min(completion.completionPercentage, 100)}%` }"></div>
    </div>
    <div v-if="completion.missingTargets.length" class="mt-4">
      <span class="section-label">Missing targets</span>
      <div class="mt-2 flex flex-wrap gap-[0.35rem]">
        <span v-for="target in completion.missingTargets" :key="target.id || target.label" class="chip-sm">
          {{ target.label }}
        </span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { CoinSetCompletion } from '@/types'

defineProps<{
  completion: CoinSetCompletion
}>()
</script>
