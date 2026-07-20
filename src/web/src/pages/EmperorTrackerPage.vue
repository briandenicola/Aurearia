<template>
  <div class="container flex flex-col gap-6">
    <header class="page-header flex flex-nowrap items-center justify-between gap-4">
      <div>
        <p class="section-label">Collection Insights</p>
        <h1>Emperors</h1>
        <p class="mt-[0.35rem] text-base text-text-secondary">
          Your collection's progress toward every Western and Eastern Roman Emperor.
        </p>
      </div>
      <router-link
        class="inline-flex shrink-0 items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
        to="/stats"
        aria-label="Back to Stats"
      >
        <ArrowLeft :size="20" />
      </router-link>
    </header>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="!enabled" class="card px-8 py-12 text-center">
      <Crown :size="48" :stroke-width="1" class="mx-auto mb-4 text-text-muted" />
      <h2 class="mb-2 text-xl text-heading">Emperor Tracker isn't enabled yet</h2>
      <p class="mb-6 text-text-secondary">
        Turn it on in Settings to track your collection's progress toward every Roman Emperor.
      </p>
      <router-link to="/settings" class="btn btn-primary">Go to Settings</router-link>
    </div>

    <div v-else-if="errorMessage" class="card px-8 py-12 text-center">
      <p class="text-text-secondary">{{ errorMessage }}</p>
    </div>

    <template v-else-if="result">
      <section class="card flex flex-col gap-2 p-5">
        <h2 class="m-0 font-display text-lg font-medium text-gold">Commonly Accepted Augustuses</h2>
        <p class="m-0 text-2xl font-semibold text-heading">
          {{ result.emperor.owned }} of {{ result.emperor.total }}
          <span class="text-base font-normal text-text-secondary">({{ formatPct(result.emperor.percentage) }}%)</span>
        </p>
      </section>

      <section v-if="result.suggestions.length" class="card flex flex-col gap-3 p-5">
        <h2 class="m-0 font-display text-lg font-medium text-gold">What to Pursue Next</h2>
        <ul class="m-0 flex flex-col gap-2 p-0">
          <li
            v-for="figure in result.suggestions"
            :key="figure.id"
            class="flex items-center justify-between gap-3 border-b border-border-subtle pb-2 text-body last:border-0 last:pb-0"
          >
            <span>{{ figure.name }} <span class="text-text-muted">— {{ figure.dynasty }}</span></span>
            <span class="rounded-full border border-border-subtle px-2 py-0.5 text-xs text-text-muted">{{ rarityLabel(figure.rarityTier) }}</span>
          </li>
        </ul>
      </section>

      <section
        v-for="dynasty in result.emperor.dynasties"
        :key="dynasty.dynasty"
        class="card flex flex-col gap-3 p-5"
      >
        <h3 class="m-0 text-base text-heading">
          {{ dynasty.dynasty }} — {{ dynasty.owned }} of {{ dynasty.total }}
          ({{ formatPct(dynasty.total ? (dynasty.owned / dynasty.total) * 100 : 0) }}%)
        </h3>
        <ImperialFigureWellGrid :slots="dynasty.figures" @highlight-updated="load" />
      </section>

      <section v-if="result.usurpers" class="flex flex-col gap-3">
        <h2 class="m-0 font-display text-lg font-medium text-gold">
          Usurpers — {{ result.usurpers.owned }} of {{ result.usurpers.total }} ({{ formatPct(result.usurpers.percentage) }}%)
        </h2>
        <section
          v-for="dynasty in result.usurpers.dynasties"
          :key="dynasty.dynasty"
          class="card flex flex-col gap-3 p-5"
        >
          <h3 class="m-0 text-base text-heading">{{ dynasty.dynasty }} — {{ dynasty.owned }} of {{ dynasty.total }}</h3>
          <ImperialFigureWellGrid :slots="dynasty.figures" @highlight-updated="load" />
        </section>
      </section>

      <section v-if="result.empresses" class="flex flex-col gap-3">
        <h2 class="m-0 font-display text-lg font-medium text-gold">
          Empresses — {{ result.empresses.owned }} of {{ result.empresses.total }} ({{ formatPct(result.empresses.percentage) }}%)
        </h2>
        <section
          v-for="dynasty in result.empresses.dynasties"
          :key="dynasty.dynasty"
          class="card flex flex-col gap-3 p-5"
        >
          <h3 class="m-0 text-base text-heading">{{ dynasty.dynasty }} — {{ dynasty.owned }} of {{ dynasty.total }}</h3>
          <ImperialFigureWellGrid :slots="dynasty.figures" @highlight-updated="load" />
        </section>
      </section>

      <section v-if="result.other" class="flex flex-col gap-3">
        <h2 class="m-0 font-display text-lg font-medium text-gold">
          Other Figures — {{ result.other.owned }} of {{ result.other.total }} ({{ formatPct(result.other.percentage) }}%)
        </h2>
        <section
          v-for="dynasty in result.other.dynasties"
          :key="dynasty.dynasty"
          class="card flex flex-col gap-3 p-5"
        >
          <h3 class="m-0 text-base text-heading">{{ dynasty.dynasty }} — {{ dynasty.owned }} of {{ dynasty.total }}</h3>
          <ImperialFigureWellGrid :slots="dynasty.figures" @highlight-updated="load" />
        </section>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ArrowLeft, Crown } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { getEmperorTrackerProgress, getApiErrorMessage } from '@/api/client'
import ImperialFigureWellGrid from '@/components/emperor-tracker/ImperialFigureWellGrid.vue'
import type { EmperorTrackerResult, RarityTier } from '@/types'

const auth = useAuthStore()

const loading = ref(true)
const enabled = ref(true)
const errorMessage = ref('')
const result = ref<EmperorTrackerResult | null>(null)

const RARITY_LABELS: Record<RarityTier, string> = {
  common: 'Common',
  scarce: 'Scarce',
  rare: 'Rare',
  very_rare: 'Very Rare',
}

function rarityLabel(tier: RarityTier): string {
  return RARITY_LABELS[tier] ?? tier
}

function formatPct(value: number): string {
  return Math.round(value).toString()
}

async function load() {
  loading.value = true
  errorMessage.value = ''
  try {
    const res = await getEmperorTrackerProgress()
    result.value = res.data
    enabled.value = true
  } catch (err) {
    const status = (err as { response?: { status?: number } })?.response?.status
    if (status === 403) {
      enabled.value = auth.user?.emperorTrackerEnabled ?? false
    } else {
      errorMessage.value = getApiErrorMessage(err) || 'Failed to load emperor tracker progress.'
    }
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
