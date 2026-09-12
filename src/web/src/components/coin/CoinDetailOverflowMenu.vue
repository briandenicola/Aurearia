<template>
  <AppOverflowMenu backdrop-aria-label="Close overflow menu">
    <template #trigger="{ open, toggle }">
      <AppIconButton title="More actions" aria-label="Open overflow actions" :active="open" @click="toggle">
        <Menu :size="24" />
      </AppIconButton>
    </template>
    <template #default="{ close }">
      <button
        v-if="showShare"
        class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-55"
        :disabled="sharing"
        :aria-label="sharing ? 'Sharing...' : 'Share'"
        @click="handleShare(close)"
      >
        <Share2 :size="16" />
        {{ sharing ? 'Sharing...' : 'Share' }}
      </button>
      <button
        v-if="showReminder"
        class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
        :aria-label="reminderActive ? 'Edit Reminder' : 'Set Reminder'"
        @click="handleReminder(close)"
      >
        <BellRing :size="16" />
        {{ reminderActive ? 'Edit Reminder' : 'Set Reminder' }}
      </button>
      <button
        v-if="showPin"
        class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-55"
        :disabled="pinBusy"
        :aria-label="pinBusy ? 'Updating coin Quick Access pin' : pinLabel"
        :aria-busy="pinBusy"
        :aria-pressed="pinned"
        @click="handlePin(close)"
      >
        <PinOff v-if="pinned" :size="16" />
        <Pin v-else :size="16" />
        {{ pinBusy ? 'Updating...' : pinLabel }}
      </button>
      <div v-if="showShare || showReminder || showPin" class="my-1 border-t border-border-subtle"></div>
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
import { computed } from 'vue'
import { BellRing, CircleDollarSign, Copy, Menu, Pin, PinOff, Share2 } from 'lucide-vue-next'
import { COIN_DETAIL_SECTIONS, SECTION_ORDER, type CoinDetailSection } from '@/constants/coinDetailSections'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import AppOverflowMenu from '@/components/ui/AppOverflowMenu.vue'

const props = withDefaults(defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
  duplicating?: boolean
  showShare?: boolean
  sharing?: boolean
  showReminder?: boolean
  reminderActive?: boolean
  showPin?: boolean
  pinned?: boolean
  pinBusy?: boolean
}>(), {
  duplicating: false,
  showShare: false,
  sharing: false,
  showReminder: false,
  reminderActive: false,
  showPin: false,
  pinned: false,
  pinBusy: false,
})

const emit = defineEmits<{
  sell: []
  duplicate: []
  share: []
  reminder: []
  pin: []
}>()

const sections: CoinDetailSection[] = SECTION_ORDER.map(id => COIN_DETAIL_SECTIONS[id]) as CoinDetailSection[]

const pinLabel = computed(() => props.pinned ? 'Unpin coin from Quick Access' : 'Pin coin to Quick Access')

function handleSell(close: () => void) {
  close()
  emit('sell')
}

function handleDuplicate(close: () => void) {
  if (props.duplicating) return
  close()
  emit('duplicate')
}

function handleShare(close: () => void) {
  if (props.sharing) return
  close()
  emit('share')
}

function handleReminder(close: () => void) {
  close()
  emit('reminder')
}

function handlePin(close: () => void) {
  if (props.pinBusy) return
  close()
  emit('pin')
}
</script>
