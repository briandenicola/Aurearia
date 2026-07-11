<template>
  <div class="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 max-md:mb-4 max-md:gap-[0.35rem]">
    <button class="btn btn-ghost btn-xs justify-self-start whitespace-nowrap" @click="router.push('/')">
      <ArrowLeft :size="14" />
      Back to Gallery
    </button>
    <div class="flex min-w-0 items-center justify-end gap-[0.35rem] max-md:gap-[0.2rem]">
      <button
        class="inline-flex items-center justify-center rounded-sm p-1.5 text-text-secondary transition-colors hover:bg-gold-glow hover:text-gold disabled:cursor-not-allowed disabled:opacity-55 max-md:p-[0.35rem]"
        :disabled="sharing"
        :title="sharing ? 'Sharing...' : 'Share'"
        :aria-label="sharing ? 'Sharing...' : 'Share'"
        @click="$emit('share')"
      >
        <Share2 :size="18" />
      </button>
      <button
        v-if="!isWishlist && !isSold"
        class="inline-flex items-center justify-center rounded-sm p-1.5 text-text-secondary transition-colors hover:bg-gold-glow hover:text-gold max-md:p-[0.35rem]"
        title="Sell"
        aria-label="Sell"
        @click="$emit('sell')"
      >
        <CircleDollarSign :size="18" />
      </button>
      <router-link
        :to="`/edit/${coinId}`"
        class="inline-flex items-center justify-center rounded-sm p-1.5 text-text-secondary no-underline transition-colors hover:bg-gold-glow hover:text-gold max-md:p-[0.35rem]"
        title="Edit"
        aria-label="Edit"
      >
        <Pencil :size="18" />
      </router-link>
      <button
        class="inline-flex items-center justify-center rounded-sm p-1.5 text-text-secondary transition-colors hover:bg-gold-glow hover:text-gold disabled:cursor-not-allowed disabled:opacity-55 max-md:p-[0.35rem]"
        :disabled="duplicating"
        :title="duplicating ? 'Duplicating...' : 'Duplicate'"
        :aria-label="duplicating ? 'Duplicating...' : 'Duplicate'"
        @click="$emit('duplicate')"
      >
        <Copy :size="18" />
      </button>
      <button
        class="inline-flex items-center justify-center rounded-sm p-1.5 text-text-secondary transition-colors hover:bg-gold-glow hover:text-gold max-md:p-[0.35rem]"
        title="Delete"
        aria-label="Delete"
        @click="$emit('delete')"
      >
        <Trash2 :size="18" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowLeft, CircleDollarSign, Copy, Pencil, Share2, Trash2 } from 'lucide-vue-next'

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

defineEmits<{
  share: []
  sell: []
  duplicate: []
  delete: []
}>()

const router = useRouter()
</script>
