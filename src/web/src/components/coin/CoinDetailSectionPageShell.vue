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

      <div class="mb-4 grid grid-cols-2 gap-4">
        <div class="relative aspect-square w-full overflow-hidden rounded-md border border-border-subtle bg-card">
          <AuthenticatedImage
            v-if="obverseImage"
            :media-path="obverseImage.filePath"
            alt="Obverse"
            class="h-full w-full object-contain"
          />
          <div v-else class="flex h-full w-full flex-col items-center justify-center gap-2">
            <span class="text-body font-medium text-text-secondary">Obverse</span>
            <span class="text-sm italic text-text-muted">No image</span>
          </div>
        </div>
        <div class="relative aspect-square w-full overflow-hidden rounded-md border border-border-subtle bg-card">
          <AuthenticatedImage
            v-if="reverseImage"
            :media-path="reverseImage.filePath"
            alt="Reverse"
            class="h-full w-full object-contain"
          />
          <div v-else class="flex h-full w-full flex-col items-center justify-center gap-2">
            <span class="text-body font-medium text-text-secondary">Reverse</span>
            <span class="text-sm italic text-text-muted">No image</span>
          </div>
        </div>
      </div>

      <!-- Coin identity banner -->
      <div class="mb-6 border-b border-border-subtle pb-4">
        <h1 class="mb-1 text-2xl font-medium text-heading">{{ coin.name }}</h1>
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
import { computed } from 'vue'
import { ChevronLeft } from 'lucide-vue-next'
import { useCoinDetailContext } from '@/composables/useCoinDetailContext'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

defineProps<{
  sectionTitle: string
}>()

const { coin, loading, refreshCoin, navigateToOverview } = useCoinDetailContext()
const obverseImage = computed(() => coin.value?.images?.find((image) => image.imageType === 'obverse') ?? null)
const reverseImage = computed(() => coin.value?.images?.find((image) => image.imageType === 'reverse') ?? null)
</script>
