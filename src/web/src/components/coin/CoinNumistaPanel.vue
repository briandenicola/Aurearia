<template>
  <div class="numista-section">
    <div class="numista-header">
      <h3>Numista Lookup</h3>
      <button
        class="btn btn-secondary btn-sm"
        :disabled="searching"
        @click="handleSearch"
      >
        {{ searching ? 'Searching...' : 'Search' }}
      </button>
    </div>
    <p v-if="error" class="numista-error">{{ error }}</p>
    <div v-if="results.length" class="numista-results">
      <a
        v-for="item in results"
        :key="item.id"
        :href="`https://en.numista.com/catalogue/pieces${item.id}.html`"
        target="_blank"
        rel="noopener"
        class="numista-card"
      >
        <img v-if="item.obverse_thumbnail" :src="item.obverse_thumbnail" class="numista-thumb" />
        <div class="numista-card-info">
          <span class="numista-card-title">{{ item.title }}</span>
          <span class="numista-card-meta">
            <template v-if="item.issuer?.name">{{ item.issuer.name }}</template>
            <template v-if="item.min_year"> · {{ item.min_year }}<template v-if="item.max_year && item.max_year !== item.min_year">–{{ item.max_year }}</template></template>
          </span>
        </div>
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { searchNumista } from '@/api/client'
import type { NumistaType } from '@/types'

const props = defineProps<{
  coinName: string
  coinRuler: string
  coinDenomination: string
}>()

const searching = ref(false)
const results = ref<NumistaType[]>([])
const error = ref('')

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

<style scoped>
.numista-section {
  margin-bottom: 1.5rem;
}

.numista-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.numista-header h3 {
  font-size: 1rem;
  margin: 0;
}

.numista-error {
  font-size: 0.85rem;
  color: #e67e22;
  font-style: italic;
}

.numista-results {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 0.75rem;
}

.numista-card {
  display: flex;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  text-decoration: none;
  color: inherit;
  transition: border-color var(--transition-fast);
}

.numista-card:hover {
  border-color: var(--accent-gold);
}

.numista-thumb {
  width: 48px;
  height: 48px;
  object-fit: contain;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}

.numista-card-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
}

.numista-card-title {
  font-size: 0.85rem;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.numista-card-meta {
  font-size: 0.75rem;
  color: var(--text-muted);
}
</style>
