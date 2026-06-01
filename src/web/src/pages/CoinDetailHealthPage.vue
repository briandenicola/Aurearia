<template>
  <CoinDetailSectionPageShell section-title="Metadata Health">
    <template #default="{ coin }">
      <CoinHealthChecklist
        v-if="coinHealth && !coin.isWishlist && !coin.isSold"
        :score="coinHealth.score"
        :grade="coinHealth.grade"
        :missing-items="coinHealth.missingItems"
        @quick-action="handleHealthQuickAction(coin.id, $event)"
      />
      <div v-else class="health-empty card">
        <p v-if="coin.isWishlist || coin.isSold">
          Metadata health scoring is only available for active coins in your collection.
        </p>
        <p v-else>No health data available for this coin yet.</p>
      </div>
    </template>
  </CoinDetailSectionPageShell>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CoinDetailSectionPageShell from '@/components/coin/CoinDetailSectionPageShell.vue'
import CoinHealthChecklist from '@/components/coin/CoinHealthChecklist.vue'
import { getCoinHealth } from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import type { CoinHealthItem, HealthQuickAction } from '@/types'

const route = useRoute()
const router = useRouter()
const { showAlert } = useDialog()

const coinHealth = ref<CoinHealthItem | null>(null)

onMounted(() => {
  loadCoinHealth()
})

async function loadCoinHealth() {
  const coinId = Number(route.params.id)
  try {
    const res = await getCoinHealth(coinId)
    coinHealth.value = res.data
  } catch (err) {
    console.error('Failed to load coin health:', err)
    coinHealth.value = null
  }
}

function handleHealthQuickAction(coinId: number, action: HealthQuickAction) {
  switch (action) {
    case 'edit_metadata':
      router.push(`/edit/${coinId}`)
      break
    case 'upload_images':
      router.push(`/edit/${coinId}`)
      break
    case 'run_valuation':
      router.push(`/coin/${coinId}/actions`)
      break
    case 'run_ai_analysis':
      router.push(`/coin/${coinId}/analysis`)
      break
    default:
      showAlert('Action unavailable')
  }
}
</script>

<style scoped>
.health-empty {
  padding: 1.5rem;
  text-align: center;
  color: var(--text-secondary);
  font-size: 0.9rem;
}
</style>
