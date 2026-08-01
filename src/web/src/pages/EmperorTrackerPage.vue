<template>
  <div class="container flex flex-col gap-4">
    <header class="page-header flex flex-nowrap items-center justify-between gap-4">
      <div>
        <p class="section-label">Sets</p>
        <h1>Emperors</h1>
        <p class="mt-[0.35rem] text-base text-text-secondary">
          Your collection's progress toward every Western and Eastern Roman Emperor.
        </p>
      </div>
      <div class="relative flex shrink-0 items-center gap-2">
        <router-link
          class="inline-flex items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
          to="/sets"
          aria-label="Back to Sets"
        >
          <ArrowLeft :size="20" />
        </router-link>
        <button
          class="inline-flex items-center justify-center rounded-sm border border-border-subtle bg-transparent p-[0.4rem] text-text-secondary transition hover:border-border-accent hover:bg-gold-glow hover:text-gold"
          :class="menuOpen ? 'border-border-accent bg-gold-glow text-gold' : ''"
          aria-label="Emperor actions"
          @click="menuOpen = !menuOpen"
        >
          <Menu :size="20" />
        </button>
        <div v-if="menuOpen && result" class="absolute right-0 top-[calc(100%+0.5rem)] z-20 w-[220px] rounded-md border border-border-subtle bg-card p-2 shadow-[0_10px_26px_rgba(0,0,0,0.45)]">
          <router-link
            to="/sets/emperors/stats"
            class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-body text-text-secondary no-underline transition-all hover:bg-card-hover hover:text-text-primary"
            @click="menuOpen = false"
          >
            <ChartNoAxesColumn :size="15" />
            Stats
          </router-link>
          <router-link
            v-if="result.suggestions.length"
            to="/sets/emperors/pursuits"
            class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-body text-text-secondary no-underline transition-all hover:bg-card-hover hover:text-text-primary"
            @click="menuOpen = false"
          >
            <Crown :size="15" />
            What to Pursue Next
          </router-link>
        </div>
      </div>
    </header>
    <button v-if="menuOpen" class="fixed inset-0 z-10 bg-transparent" aria-label="Close emperor actions menu" @click="menuOpen = false"></button>

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
import { ArrowLeft, ChartNoAxesColumn, Crown, Menu } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { getEmperorTrackerProgress, getApiErrorMessage } from '@/api/client'
import ImperialFigureWellGrid from '@/components/emperor-tracker/ImperialFigureWellGrid.vue'
import type { EmperorTrackerResult } from '@/types'

const auth = useAuthStore()

const loading = ref(true)
const enabled = ref(true)
const errorMessage = ref('')
const result = ref<EmperorTrackerResult | null>(null)
const menuOpen = ref(false)

function formatPct(value: number): string {
  return Math.round(value).toString()
}

async function load() {
  loading.value = true
  menuOpen.value = false
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
