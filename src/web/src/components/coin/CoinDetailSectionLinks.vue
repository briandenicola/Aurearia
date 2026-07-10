<template>
  <div class="flex flex-col gap-2">
    <router-link
      v-for="section in sections"
      :key="section.id"
      :to="section.route(coinId)"
      class="section-link-row"
    >
      <div class="section-link-content">
        <span class="section-link-title">{{ section.title }}</span>
        <span class="section-link-description">{{ section.description }}</span>
      </div>
      <ChevronRight :size="18" class="section-link-chevron" />
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { ChevronRight } from 'lucide-vue-next'
import { COIN_DETAIL_SECTIONS, SECTION_ORDER, type CoinDetailSection } from '@/constants/coinDetailSections'

defineProps<{
  coinId: number
}>()

const sections: CoinDetailSection[] = SECTION_ORDER.map(id => COIN_DETAIL_SECTIONS[id]) as CoinDetailSection[]
</script>
