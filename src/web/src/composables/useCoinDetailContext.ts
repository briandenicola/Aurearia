// T007: Shared coin detail context composable for #219
// This composable provides shared coin loading logic for the overview page and all section pages

import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useCoinsStore } from '@/stores/coins'
import type { Coin } from '@/types'

export function useCoinDetailContext() {
  const route = useRoute()
  const router = useRouter()
  const store = useCoinsStore()

  const coinId = computed(() => Number(route.params.id))
  const coin = computed<Coin | null>(() => store.currentCoin)
  const loading = computed(() => store.loading && !coin.value)

  async function loadCoin() {
    if (coinId.value) {
      await store.fetchCoin(coinId.value)
    }
  }

  async function refreshCoin() {
    if (coinId.value) {
      await store.fetchCoin(coinId.value)
    }
  }

  function navigateToOverview() {
    if (coinId.value) {
      router.push(`/coin/${coinId.value}`)
    }
  }

  onMounted(() => {
    loadCoin()
  })

  return {
    coinId,
    coin,
    loading,
    loadCoin,
    refreshCoin,
    navigateToOverview,
  }
}
