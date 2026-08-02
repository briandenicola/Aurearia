<template>
  <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 max-md:mb-4">
    <button class="btn btn-ghost btn-xs justify-self-start whitespace-nowrap" @click="router.push('/')">
      <ArrowLeft :size="14" />
      Back to Gallery
    </button>
    <div class="relative flex min-w-0 items-center justify-end gap-[0.45rem]">
      <button
        class="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-transparent text-text-secondary transition-all hover:border-border-accent hover:bg-gold-glow hover:text-gold disabled:cursor-not-allowed disabled:opacity-55"
        :disabled="sharing"
        :title="sharing ? 'Sharing...' : 'Share'"
        :aria-label="sharing ? 'Sharing...' : 'Share'"
        @click="$emit('share')"
      >
        <Share2 :size="24" />
      </button>
      <router-link
        :to="`/edit/${coinId}`"
        class="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-transparent text-text-secondary no-underline transition-all hover:border-border-accent hover:bg-gold-glow hover:text-gold"
        title="Edit"
        aria-label="Edit"
      >
        <Pencil :size="24" />
      </router-link>
      <button
        class="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-transparent text-text-secondary transition-all hover:border-border-accent hover:bg-gold-glow hover:text-gold"
        title="Delete"
        aria-label="Delete"
        @click="$emit('delete')"
      >
        <Trash2 :size="24" />
      </button>
      <button
        class="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-transparent text-text-secondary transition-all hover:border-border-accent hover:bg-gold-glow hover:text-gold"
        :class="menuOpen ? 'border-border-accent bg-gold-glow text-gold' : ''"
        title="More actions"
        aria-label="Open overflow actions"
        @click="menuOpen = !menuOpen"
      >
        <Menu :size="24" />
      </button>
      <div v-if="menuOpen" class="absolute right-0 top-[calc(100%+0.45rem)] z-20 w-[260px] rounded-md border border-border-subtle bg-card p-2 shadow-[0_10px_26px_rgba(0,0,0,0.45)]">
        <button
          v-if="!isWishlist && !isSold"
          class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
          aria-label="Sell Coin"
          @click="handleSell"
        >
          <CircleDollarSign :size="16" />
          Sell Coin
        </button>
        <button
          class="inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-55"
          :disabled="duplicating"
          :aria-label="duplicating ? 'Copying coin...' : 'Copy Coin'"
          @click="handleDuplicate"
        >
          <Copy :size="16" />
          {{ duplicating ? 'Copying...' : 'Copy Coin' }}
        </button>
        <router-link
          v-for="section in sections"
          :key="section.id"
          :to="section.route(coinId)"
          class="inline-flex w-full items-center rounded-sm px-3 py-2 text-body text-text-secondary no-underline transition-all hover:bg-card-hover hover:text-text-primary"
          @click="menuOpen = false"
        >
          {{ section.title }}
        </router-link>
      </div>
    </div>
    <button v-if="menuOpen" class="fixed inset-0 z-10 bg-transparent" aria-label="Close overflow menu" @click="menuOpen = false"></button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, CircleDollarSign, Copy, Menu, Pencil, Share2, Trash2 } from 'lucide-vue-next'
import { COIN_DETAIL_SECTIONS, SECTION_ORDER, type CoinDetailSection } from '@/constants/coinDetailSections'

const props = withDefaults(defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
  sharing?: boolean
  duplicating?: boolean
}>(), {
  sharing: false,
  duplicating: false,
})

const emit = defineEmits<{
  share: []
  sell: []
  duplicate: []
  delete: []
}>()

const router = useRouter()
const menuOpen = ref(false)
const sections: CoinDetailSection[] = SECTION_ORDER.map(id => COIN_DETAIL_SECTIONS[id]) as CoinDetailSection[]

function handleSell() {
  menuOpen.value = false
  emit('sell')
}

function handleDuplicate() {
  if (props.duplicating) return
  menuOpen.value = false
  emit('duplicate')
}
</script>
