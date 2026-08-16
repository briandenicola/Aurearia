<template>
  <CoinDetailSectionPageShell section-title="Actions">
    <template #default="{ coin, refresh }">
      <CoinActionsPanel
        :coin-id="coin.id"
        :coin-name="coin.name"
        :coin-ruler="coin.ruler ?? ''"
        :coin-denomination="coin.denomination ?? ''"
        :coin-mint="coin.mint ?? ''"
        :coin-date-range="coin.dateRange ?? ''"
        :coin-material="coin.material"
        :coin-obverse-inscription="coin.obverseInscription ?? ''"
        :coin-reverse-inscription="coin.reverseInscription ?? ''"
        :image-count="coin.images?.length ?? 0"
        :coin-has-obverse-image="coin.images?.some((img) => img.imageType === 'obverse') ?? false"
        :coin-has-reverse-image="coin.images?.some((img) => img.imageType === 'reverse') ?? false"
        :is-pwa="isPwa"
        @images-changed="refresh"
        @estimate-applied="handleEstimateApplied"
      />
    </template>
  </CoinDetailSectionPageShell>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import CoinDetailSectionPageShell from '@/components/coin/CoinDetailSectionPageShell.vue'
import CoinActionsPanel from '@/components/coin/CoinActionsPanel.vue'
import { usePwa } from '@/composables/usePwa'
import { getJournalEntries } from '@/api/client'
import type { CoinJournal } from '@/types'
import { ref } from 'vue'

const { isPwa } = usePwa()
const route = useRoute()

const journalEntries = ref<CoinJournal[]>([])

async function handleEstimateApplied() {
  const coinId = Number(route.params.id)
  // Reload journal entries after estimate is applied (it creates a journal entry)
  try {
    const res = await getJournalEntries(coinId)
    journalEntries.value = res.data || []
  } catch {
    journalEntries.value = []
  }
}
</script>
