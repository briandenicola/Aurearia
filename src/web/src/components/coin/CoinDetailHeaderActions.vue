<template>
  <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 max-md:mb-4">
    <button class="btn btn-ghost btn-xs justify-self-start whitespace-nowrap" @click="router.push('/')">
      <ArrowLeft :size="14" />
      Back to Gallery
    </button>
    <div class="flex min-w-0 items-center justify-end gap-[0.45rem]">
      <AppIconButton
        :disabled="sharing"
        :title="sharing ? 'Sharing...' : 'Share'"
        :aria-label="sharing ? 'Sharing...' : 'Share'"
        @click="$emit('share')"
      >
        <Share2 :size="24" />
      </AppIconButton>
      <AppIconButton title="Edit" aria-label="Edit" @click="$emit('edit')">
        <Pencil :size="24" />
      </AppIconButton>
      <AppIconButton title="Delete" aria-label="Delete" @click="$emit('delete')">
        <Trash2 :size="24" />
      </AppIconButton>
      <CoinDetailOverflowMenu
        :coin-id="coinId"
        :is-wishlist="isWishlist"
        :is-sold="isSold"
        :duplicating="duplicating"
        @sell="emit('sell')"
        @duplicate="emit('duplicate')"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowLeft, Pencil, Share2, Trash2 } from 'lucide-vue-next'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import CoinDetailOverflowMenu from '@/components/coin/CoinDetailOverflowMenu.vue'

withDefaults(defineProps<{
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
  edit: []
  delete: []
}>()

const router = useRouter()
</script>
