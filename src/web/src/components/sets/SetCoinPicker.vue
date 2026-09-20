<template>
  <div class="form-group">
    <label :for="searchId" class="form-label">Search coins</label>
    <input :id="searchId" v-model="search" type="search" class="form-input" placeholder="Search by name, ruler, denomination, or mint" />
  </div>
  <div class="form-group">
    <label :for="selectId" class="form-label">Coin</label>
    <select :id="selectId" :value="modelValue ?? ''" class="form-select" required @change="selectCoin">
      <option value="" disabled>Select a coin...</option>
      <option v-for="coin in options" :key="coin.id" :value="coin.id">
        {{ coin.name }}<template v-if="coin.ruler"> - {{ coin.ruler }}</template>
      </option>
    </select>
    <p v-if="loading" class="mt-2 text-chip text-text-secondary" role="status">Loading coins...</p>
    <div v-else-if="error" class="mt-2" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="btn btn-secondary btn-sm" @click="load">Retry</button>
    </div>
    <p v-else-if="pageCoins.length === 0" class="mt-2 text-chip text-text-secondary">No eligible coins on this page.</p>
    <div class="mt-3 flex items-center justify-between gap-2" aria-label="Coin search pages">
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading || page <= 1" @click="page--">&lt; Previous</button>
      <span class="text-chip text-text-secondary">Page {{ page }} of {{ pages }} ({{ total }} matches)</span>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading || page >= pages" @click="page++">Next &gt;</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onScopeDispose, ref, watch } from 'vue'
import { getCoins } from '@/api/client'
import type { Coin } from '@/types'

const props = defineProps<{
  modelValue: number | null
  selectedCoin?: Coin
  excludedIds: number[]
  allowWishlist: boolean
  searchId: string
  selectId: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: number | null] }>()
const search = ref('')
const page = ref(1)
const total = ref(0)
const results = ref<Coin[]>([])
const selected = ref<Coin | undefined>(props.selectedCoin)
const loading = ref(false)
const error = ref('')
const perPage = 50
const pages = computed(() => Math.max(1, Math.ceil(total.value / perPage)))
const pageCoins = computed(() => results.value.filter(eligible))
const options = computed(() => {
  const coin = selected.value
  return coin && coin.id === props.modelValue && eligible(coin) && !pageCoins.value.some(item => item.id === coin.id)
    ? [coin, ...pageCoins.value] : pageCoins.value
})
let request = 0

function eligible(coin: Coin) {
  return !coin.isSold && (props.allowWishlist || !coin.isWishlist) && !props.excludedIds.includes(coin.id)
}

function selectCoin(event: Event) {
  const target = event.target
  if (!(target instanceof HTMLSelectElement)) return
  selected.value = options.value.find(coin => coin.id === Number(target.value))
  emit('update:modelValue', selected.value?.id ?? null)
}

async function load() {
  const current = ++request
  loading.value = true
  error.value = ''
  results.value = []
  try {
    const response = await getCoins({
      wishlist: props.allowWishlist ? undefined : 'false',
      sold: 'false', page: page.value, limit: perPage, sort: 'name', order: 'asc', search: search.value.trim(),
    })
    if (current !== request) return
    results.value = response.data.coins
    total.value = response.data.total
  } catch {
    if (current === request) error.value = 'Unable to load coins. Please try again.'
  } finally {
    if (current === request) loading.value = false
  }
}

watch(search, () => { page.value = 1 })
watch([search, page, () => props.allowWishlist], () => { void load() }, { immediate: true })
onScopeDispose(() => { request += 1 })
</script>
