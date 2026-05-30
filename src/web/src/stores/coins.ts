import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Coin, StatsResponse, ValueSnapshot, CollectionHealthSummary, CoinHealthItem } from '@/types'
import * as api from '@/api/client'

export const useCoinsStore = defineStore('coins', () => {
  const coins = ref<Coin[]>([])
  const currentCoin = ref<Coin | null>(null)
  const total = ref(0)
  const loading = ref(false)
  const stats = ref<StatsResponse | null>(null)
  const valueHistory = ref<ValueSnapshot[]>([])
  const selectedCategory = ref('')
  const searchQuery = ref('')
  const activeSortKey = ref('')
  const galleryIndex = ref(0)
  const collectionHealth = ref<CollectionHealthSummary | null>(null)
  const coinHealthList = ref<CoinHealthItem[]>([])
  const healthLoading = ref(false)

  async function fetchCoins(params?: {
    category?: string
    search?: string
    wishlist?: string
    sold?: string
    tag?: string
    page?: number
    sort?: string
    order?: string
    seed?: number
  }) {
    loading.value = true
    try {
      const res = await api.getCoins(params)
      coins.value = res.data.coins || []
      total.value = res.data.total
    } finally {
      loading.value = false
    }
  }

  async function fetchCoin(id: number) {
    loading.value = true
    try {
      const res = await api.getCoin(id)
      currentCoin.value = res.data
    } finally {
      loading.value = false
    }
  }

  async function addCoin(coin: Partial<Coin>) {
    const res = await api.createCoin(coin)
    return res.data
  }

  async function editCoin(id: number, coin: Partial<Coin>) {
    const res = await api.updateCoin(id, coin)
    return res.data
  }

  async function removeCoin(id: number) {
    await api.deleteCoin(id)
    coins.value = coins.value.filter((c) => c.id !== id)
  }

  async function fetchStats() {
    const res = await api.getStats()
    stats.value = res.data
  }

  async function fetchValueHistory() {
    const res = await api.getValueHistory()
    valueHistory.value = res.data
  }

  async function fetchCollectionHealth() {
    healthLoading.value = true
    try {
      const res = await api.getCollectionHealthSummary()
      collectionHealth.value = res.data
    } catch (error) {
      collectionHealth.value = null
      throw error
    } finally {
      healthLoading.value = false
    }
  }

  async function fetchCoinHealthList(scope?: 'all' | 'needs_attention', page = 1, limit = 25) {
    healthLoading.value = true
    try {
      const res = await api.getCoinHealthList({ scope, page, limit })
      coinHealthList.value = res.data.coins
      return res.data
    } finally {
      healthLoading.value = false
    }
  }

  return {
    coins,
    currentCoin,
    total,
    loading,
    stats,
    valueHistory,
    selectedCategory,
    searchQuery,
    activeSortKey,
    galleryIndex,
    collectionHealth,
    coinHealthList,
    healthLoading,
    fetchCoins,
    fetchCoin,
    addCoin,
    editCoin,
    removeCoin,
    fetchStats,
    fetchValueHistory,
    fetchCollectionHealth,
    fetchCoinHealthList,
  }
})
