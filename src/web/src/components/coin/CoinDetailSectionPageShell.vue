<template>
  <div class="container">
    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="coin" class="mx-auto max-w-[900px]">
      <!-- Back link -->
      <div class="mb-4">
        <button class="btn btn-ghost btn-sm inline-flex items-center gap-1" @click="navigateToOverview">
          <ChevronLeft :size="16" />
          Back to Overview
        </button>
      </div>

      <!-- Coin identity banner -->
      <div class="mb-6 border-b border-border-subtle pb-4">
        <h1 class="mb-1 text-xl font-medium text-heading">{{ coin.name }}</h1>
        <p v-if="coin.ruler" class="text-base text-text-secondary">{{ coin.ruler }}</p>
      </div>

      <!-- Section title -->
      <h2 class="mb-6 text-lg font-medium text-heading">{{ sectionTitle }}</h2>

      <!-- Content slot -->
      <div>
        <slot :coin="coin" :refresh="refreshCoin"></slot>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeft } from 'lucide-vue-next'
import { useCoinDetailContext } from '@/composables/useCoinDetailContext'

defineProps<{
  sectionTitle: string
}>()

const { coin, loading, refreshCoin, navigateToOverview } = useCoinDetailContext()
</script>
