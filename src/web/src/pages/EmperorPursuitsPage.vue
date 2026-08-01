<template>
  <div class="container flex flex-col gap-6">
    <header class="page-header flex flex-nowrap items-center justify-between gap-4">
      <div>
        <p class="section-label">Emperors</p>
        <h1>What to Pursue Next</h1>
        <p class="mt-[0.35rem] text-base text-text-secondary">
          Suggested rulers to target next based on your current Emperor tracker progress.
        </p>
      </div>
      <router-link
        class="inline-flex shrink-0 items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
        to="/sets/emperors"
        aria-label="Back to Emperors"
      >
        <ArrowLeft :size="20" />
      </router-link>
    </header>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="errorMessage" class="card px-8 py-12 text-center">
      <p class="text-text-secondary">{{ errorMessage }}</p>
    </div>

    <section v-else-if="suggestions.length" class="card flex flex-col gap-3 p-5">
      <ul class="m-0 flex flex-col gap-2 p-0">
        <li
          v-for="figure in suggestions"
          :key="figure.id"
          class="flex items-center justify-between gap-3 border-b border-border-subtle pb-2 text-body last:border-0 last:pb-0"
        >
          <span class="min-w-0">
            <span class="font-medium text-text-primary">{{ figure.name }}</span>
            <span class="text-text-muted"> — {{ figure.dynasty }}</span>
          </span>
          <span class="flex shrink-0 items-center gap-2">
            <button
              type="button"
              class="btn btn-xs btn-ghost whitespace-nowrap"
              :aria-label="`Ask the agent to search for ${figure.name} coins`"
              @click="searchForFigure(figure.name)"
            >
              Search Agent
            </button>
            <span class="rounded-full border border-border-subtle px-2 py-0.5 text-xs text-text-muted">{{ rarityLabel(figure.rarityTier) }}</span>
          </span>
        </li>
      </ul>
    </section>

    <section v-else class="card px-8 py-12 text-center">
      <p class="text-text-secondary">No pursuit suggestions are available right now.</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { getApiErrorMessage, getEmperorTrackerProgress } from '@/api/client'
import type { RarityTier, RomanImperialFigure } from '@/types'

const loading = ref(true)
const errorMessage = ref('')
const suggestions = ref<RomanImperialFigure[]>([])

const RARITY_LABELS: Record<RarityTier, string> = {
  common: 'Common',
  scarce: 'Scarce',
  rare: 'Rare',
  very_rare: 'Very Rare',
}

function rarityLabel(tier: RarityTier): string {
  return RARITY_LABELS[tier] ?? tier
}

function searchForFigure(name: string) {
  window.dispatchEvent(new window.CustomEvent('open-agent-chat', {
    detail: {
      prompt: `Look for available ${name} coins for my collection. Focus on reputable dealer and auction listings, include prices and links, and only show coins that appear available.`,
    },
  }))
}

async function load() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await getEmperorTrackerProgress()
    suggestions.value = res.data.suggestions ?? []
  } catch (error) {
    errorMessage.value = getApiErrorMessage(error) || 'Failed to load pursuit suggestions.'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
