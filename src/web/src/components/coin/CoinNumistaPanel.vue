<template>
  <div class="mb-6">
    <div class="mb-3 flex items-center justify-between gap-3">
      <h3 class="m-0 text-base font-medium text-text-primary">Numista Lookup</h3>
      <button
        class="btn btn-secondary btn-sm"
        :disabled="searching"
        @click="handleSearch"
      >
        {{ searching ? 'Searching...' : 'Search' }}
      </button>
    </div>
    <p v-if="error" class="text-body italic text-warning">{{ error }}</p>
    <div v-if="results.length" class="grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(280px,1fr))]">
      <SafeExternalLink
        v-for="item in results"
        :key="item.id"
        :href="numistaPieceUrl(item.id)"
        target="_blank"
        rel="noopener"
        class="flex gap-3 rounded-sm border border-border-subtle bg-surface px-3 py-3 text-inherit no-underline transition-colors hover:border-gold"
      >
        <img v-if="item.obverse_thumbnail" :src="item.obverse_thumbnail" class="h-12 w-12 shrink-0 rounded-sm object-contain" />
        <div class="flex min-w-0 flex-col gap-[0.15rem]">
          <span class="line-clamp-2 text-body font-medium text-text-primary">{{ item.title }}</span>
          <span class="text-sm text-text-muted">
            <template v-if="item.issuer?.name">{{ item.issuer.name }}</template>
            <template v-if="item.min_year"> · {{ item.min_year }}<template v-if="item.max_year && item.max_year !== item.min_year">–{{ item.max_year }}</template></template>
          </span>
        </div>
      </SafeExternalLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { searchNumista } from '@/api/client'
import type { NumistaType } from '@/types'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

const props = defineProps<{
  coinName: string
  coinRuler: string
  coinDenomination: string
}>()

const searching = ref(false)
const results = ref<NumistaType[]>([])
const error = ref('')

function numistaPieceUrl(pieceId: number): string {
  return `https://en.numista.com/catalogue/pieces${pieceId}.html`
}

async function handleSearch() {
  searching.value = true
  error.value = ''
  results.value = []
  try {
    const q = [props.coinName, props.coinDenomination, props.coinRuler].filter(Boolean).join(' ')
    const res = await searchNumista(q)
    results.value = res.data.types || []
    if (!results.value.length) {
      error.value = 'No results found on Numista'
    }
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Numista search failed'
    if (typeof err === 'object' && err !== null && 'response' in err) {
      const axiosErr = err as { response?: { data?: { error?: string } } }
      error.value = axiosErr.response?.data?.error || error.value
    }
  } finally {
    searching.value = false
  }
}
</script>
