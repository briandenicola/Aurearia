<template>
  <div class="card">
    <div>
      <h3 class="mb-5 text-lg text-heading">Collection Health</h3>
    </div>
    <div class="flex flex-col gap-6">
      <div class="flex flex-col items-center gap-2 rounded-sm bg-input p-4">
        <div class="font-display text-[3rem] font-semibold text-gold max-md:text-[2.5rem]">{{ summary.score }}</div>
        <div
          class="rounded-full border px-[0.8rem] py-1 text-base font-semibold uppercase tracking-[0.08em]"
          :class="summary.grade === 'A'
            ? 'border-[rgba(39,174,96,0.3)] bg-[rgba(39,174,96,0.15)] text-green-400'
            : summary.grade === 'B'
              ? 'border-[rgba(52,152,219,0.3)] bg-[rgba(52,152,219,0.15)] text-sky-400'
              : summary.grade === 'C'
                ? 'border-[rgba(243,156,18,0.3)] bg-[rgba(243,156,18,0.15)] text-amber-400'
                : summary.grade === 'D'
                  ? 'border-[rgba(230,126,34,0.3)] bg-[rgba(230,126,34,0.15)] text-orange-400'
                  : 'border-[rgba(231,76,60,0.3)] bg-[rgba(231,76,60,0.15)] text-red-400'"
        >
          Grade {{ summary.grade }}
        </div>
      </div>

      <div class="flex flex-col gap-3">
        <div
          v-for="(value, key) in summary.dimensions"
          :key="key"
          class="grid grid-cols-[auto_minmax(0,1fr)_50px] items-center gap-3 max-md:grid-cols-[auto_minmax(0,1fr)_45px] max-md:gap-2"
        >
          <div class="whitespace-nowrap text-body font-medium text-text-secondary">{{ formatDimensionLabel(key) }}</div>
          <div class="overflow-hidden rounded-full bg-input">
            <div
              class="h-2 rounded-full transition-[width] duration-300"
              :class="key === 'metadata'
                ? 'bg-[linear-gradient(90deg,var(--accent-gold),var(--accent-bronze))]'
                : key === 'imageCoverage'
                  ? 'bg-[linear-gradient(90deg,#3498db,#2980b9)]'
                  : key === 'valuationFreshness'
                    ? 'bg-[linear-gradient(90deg,#27ae60,#229954)]'
                    : 'bg-[linear-gradient(90deg,#9b59b6,#8e44ad)]'"
              :style="{ width: `${value}%` }"
            ></div>
          </div>
          <div class="text-right text-body font-semibold text-text-primary">{{ value }}%</div>
        </div>
      </div>

      <div class="border-t border-border-subtle pt-3">
        <div class="section-label mb-2">Scoring Weights</div>
        <div class="flex flex-wrap gap-[0.35rem]">
          <span
            v-for="(value, key) in summary.weights"
            :key="key"
            class="rounded-full border border-border-subtle bg-[var(--accent-gold-glow)] px-2 py-[0.15rem] text-sm text-text-secondary"
          >
            {{ formatDimensionLabel(key) }}: {{ value }}%
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CollectionHealthSummary } from '@/types'

defineProps<{
  summary: CollectionHealthSummary
}>()

function formatDimensionLabel(key: string): string {
  const labels: Record<string, string> = {
    metadata: 'Metadata',
    imageCoverage: 'Image Coverage',
    valuationFreshness: 'Valuation Freshness',
    aiCoverage: 'AI Coverage',
  }
  return labels[key] || key
}

function getDimensionFillClass(key: string): string {
  const cssMap: Record<string, string> = {
    metadata: 'fill-metadata',
    imageCoverage: 'fill-images',
    valuationFreshness: 'fill-valuation',
    aiCoverage: 'fill-ai',
  }
  return cssMap[key] || `fill-${key}`
}
</script>
