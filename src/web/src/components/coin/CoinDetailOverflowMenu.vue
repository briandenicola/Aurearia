<template>
  <AppOverflowMenu backdrop-aria-label="Close overflow menu">
    <template #trigger="{ open, toggle }">
      <AppIconButton title="More actions" aria-label="Open overflow actions" :active="open" @click="toggle">
        <Menu :size="24" />
      </AppIconButton>
    </template>
    <template #default="{ close }">
      <button
        v-if="!isWishlist && !isSold"
        class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
        aria-label="Sell Coin"
        @click="handleSell(close)"
      >
        <CircleDollarSign :size="16" />
        Sell Coin
      </button>
      <button
        class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-55"
        :disabled="duplicating"
        :aria-label="duplicating ? 'Copying coin...' : 'Copy Coin'"
        @click="handleDuplicate(close)"
      >
        <Copy :size="16" />
        {{ duplicating ? 'Copying...' : 'Copy Coin' }}
      </button>
      <router-link
        v-for="section in sections"
        :key="section.id"
        :to="section.route(coinId)"
        class="inline-flex w-full items-center rounded-sm px-3 py-2 text-body text-text-secondary no-underline transition-all hover:bg-card-hover hover:text-text-primary"
        @click="close"
      >
        {{ section.title }}
      </router-link>
    </template>
  </AppOverflowMenu>
</template>

<script setup lang="ts">
import { CircleDollarSign, Copy, Menu } from 'lucide-vue-next'
import { COIN_DETAIL_SECTIONS, SECTION_ORDER, type CoinDetailSection } from '@/constants/coinDetailSections'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import AppOverflowMenu from '@/components/ui/AppOverflowMenu.vue'

const props = withDefaults(defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
  duplicating?: boolean
}>(), {
  duplicating: false,
})

const emit = defineEmits<{
  sell: []
  duplicate: []
}>()

const sections: CoinDetailSection[] = SECTION_ORDER.map(id => COIN_DETAIL_SECTIONS[id]) as CoinDetailSection[]

function handleSell(close: () => void) {
  close()
  emit('sell')
}

function handleDuplicate(close: () => void) {
  if (props.duplicating) return
  close()
  emit('duplicate')
}
</script>
