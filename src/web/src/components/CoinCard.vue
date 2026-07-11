<template>
  <div
    class="card group flex cursor-pointer flex-col overflow-hidden p-0"
    :class="selectable && selected ? 'outline-2 outline-gold outline-offset-[-2px]' : ''"
    @click="handleClick"
  >
    <div class="relative flex aspect-square w-full items-center justify-center overflow-hidden bg-[radial-gradient(ellipse_at_center,var(--bg-secondary)_0%,var(--bg-primary)_100%)] [@media(display-mode:standalone)]:aspect-[5/6]">
      <AuthenticatedImage v-if="primaryImage" :media-path="primaryImage" :alt="coin.name" class="h-full w-full object-contain transition duration-300 group-hover:scale-[1.02] group-hover:brightness-110" loading="lazy">
        <template #fallback>
          <div class="text-text-primary/30"><Coins :size="48" :stroke-width="1" /></div>
        </template>
      </AuthenticatedImage>
      <div v-else class="text-text-primary/30"><Coins :size="48" :stroke-width="1" /></div>
      <div v-if="wishlist && coin.listingStatus === 'unavailable'" class="pointer-events-none absolute inset-0 z-[2] bg-black/50"></div>
      <span v-if="wishlist && coin.listingStatus === 'unavailable'" class="absolute top-2 right-2 z-[3] rounded-full bg-red-600/85 px-2 py-[0.2rem] text-label font-semibold uppercase tracking-[0.08em] text-white">Unavailable</span>
      <button
        v-if="wishlist && coin.listingStatus === 'unavailable'"
        class="absolute right-2 bottom-2 z-[3] rounded-sm border border-border-subtle bg-black/70 px-2 py-[0.15rem] text-[0.65rem] text-text-secondary transition-colors hover:bg-black/85 hover:text-text-primary"
        @click.stop="emit('dismiss-status', coin.id)"
      >
        Dismiss
      </button>
      <div
        v-if="selectable"
        class="absolute top-2 left-2 z-[4] flex h-8 w-8 items-center justify-center rounded-full border-2 border-white/70 bg-black/40 transition-all [@media(display-mode:standalone)]:h-9 [@media(display-mode:standalone)]:w-9"
        :class="selected ? 'border-gold bg-gold text-black' : ''"
        @click.stop="emit('toggle-select', coin.id)"
      >
        <Check v-if="selected" :size="16" :stroke-width="3" />
      </div>
      <div class="pointer-events-none absolute inset-0 z-[1] border-b border-gold-dim shadow-[inset_0_0_40px_rgba(0,0,0,0.35)] transition-shadow duration-300 group-hover:shadow-[inset_0_0_25px_rgba(0,0,0,0.2),0_0_20px_var(--accent-gold-glow)]"></div>
    </div>
    <div class="flex flex-1 flex-col gap-1.5 p-4">
      <h3 class="overflow-hidden text-[1rem] leading-[1.3] [display:-webkit-box] [-webkit-box-orient:vertical] [-webkit-line-clamp:2]">
        <span
          v-if="wishlist && coin.listingStatus === 'available'"
          class="mr-1.5 inline-block h-2 w-2 shrink-0 align-middle rounded-full bg-gain"
          title="Available"
        ></span>
        <span
          v-if="wishlist && coin.listingStatus === 'unknown'"
          class="mr-1.5 inline-block h-2 w-2 shrink-0 align-middle rounded-full bg-warning"
          title="Unknown"
        ></span>
        {{ coin.name }}
      </h3>
      <template v-if="!wishlist && !sold">
        <p v-if="cardInscription" class="overflow-hidden text-chip italic text-text-secondary [display:-webkit-box] [-webkit-box-orient:vertical] [-webkit-line-clamp:1]">{{ cardInscription }}</p>
        <div class="flex flex-wrap gap-2">
          <span v-if="coin.category" class="badge" :class="`badge-${coin.category.toLowerCase()}`">{{ coin.category }}</span>
          <span v-if="coin.denomination" class="rounded-full border border-border-subtle bg-surface px-[0.65rem] py-[0.2rem] text-chip text-text-secondary [@media(display-mode:standalone)]:px-3 [@media(display-mode:standalone)]:py-1 [@media(display-mode:standalone)]:text-body">{{ coin.denomination }}</span>
          <span v-if="coin.material" class="rounded-full border border-border-subtle bg-surface px-[0.65rem] py-[0.2rem] text-chip text-text-secondary [@media(display-mode:standalone)]:px-3 [@media(display-mode:standalone)]:py-1 [@media(display-mode:standalone)]:text-body">{{ coin.material }}</span>
        </div>
        <div v-if="coin.tags?.length" class="mb-1 flex flex-wrap gap-1">
          <span
            v-for="tag in coin.tags"
            :key="tag.id"
            class="whitespace-nowrap rounded-full border px-2 py-[0.15rem] text-sm leading-[1.4]"
            :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }"
          >{{ tag.name }}</span>
        </div>
      </template>
      <template v-if="sold">
        <div v-if="coin.tags?.length" class="mb-1 flex flex-wrap gap-1">
          <span
            v-for="tag in coin.tags"
            :key="tag.id"
            class="whitespace-nowrap rounded-full border px-2 py-[0.15rem] text-sm leading-[1.4]"
            :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }"
          >{{ tag.name }}</span>
        </div>
        <div class="mt-1 flex flex-col gap-[0.15rem] text-[0.82rem]">
          <div v-if="coin.soldPrice" class="font-semibold text-gold">Sold: {{ formatCurrency(coin.soldPrice) }}</div>
          <div v-if="coin.purchasePrice" class="text-[0.78rem] text-text-muted">Paid: {{ formatCurrency(coin.purchasePrice) }}</div>
          <div v-if="coin.soldPrice && coin.purchasePrice" class="text-[0.82rem] font-semibold" :class="coin.soldPrice < coin.purchasePrice ? 'text-loss' : 'text-gain'">
            {{ coin.soldPrice >= coin.purchasePrice ? '+' : '' }}{{ formatCurrency(coin.soldPrice - coin.purchasePrice) }}
          </div>
          <div v-if="coin.soldTo" class="mt-[0.15rem] text-[0.78rem] text-text-secondary">To: {{ coin.soldTo }}</div>
        </div>
      </template>
      <div v-if="wishlist && (coin.currentValue || coin.purchasePrice)" class="mt-auto text-base font-semibold text-gold">
        {{ formatCurrency(coin.currentValue || coin.purchasePrice || 0) }}
      </div>
      <SafeExternalLink
        v-if="wishlist && coin.referenceUrl"
        :href="coin.referenceUrl"
        class="block truncate text-chip text-gold hover:underline"
        target="_blank"
        rel="noopener noreferrer"
        @click.stop
      >
        {{ coin.referenceText || coin.referenceUrl }}
      </SafeExternalLink>
      <button
        v-if="wishlist"
        class="btn btn-primary btn-sm mt-2 flex w-full justify-center gap-[0.35rem] text-chip"
        @click.stop="emit('purchase', coin)"
      >
        <ShoppingCart :size="14" /> Purchased
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Coin, ImageType } from '@/types'
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Coins, ShoppingCart, Check } from 'lucide-vue-next'
import { formatCurrency } from '@/utils/format'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const router = useRouter()

const props = withDefaults(defineProps<{
  coin: Coin
  imageSide?: ImageType | null
  wishlist?: boolean
  sold?: boolean
  selectable?: boolean
  selected?: boolean
}>(), {
  imageSide: null,
  wishlist: false,
  sold: false,
  selectable: false,
  selected: false,
})

const emit = defineEmits<{
  purchase: [coin: Coin]
  'dismiss-status': [coinId: number]
  'toggle-select': [coinId: number]
}>()

function handleClick() {
  if (props.selectable) {
    emit('toggle-select', props.coin.id)
  } else {
    router.push(`/coin/${props.coin.id}`)
  }
}

const primaryImage = computed(() => {
  if (props.imageSide) {
    const byType = props.coin.images?.find((img) => img.imageType === props.imageSide)
    if (byType) return byType.filePath
  }
  const primary = props.coin.images?.find((img) => img.isPrimary)
  const first = props.coin.images?.[0]
  const img = primary || first
  return img ? img.filePath : null
})

const cardInscription = computed(() => {
  if (props.imageSide === 'reverse' && props.coin.reverseInscription) {
    return props.coin.reverseInscription
  }
  if (props.imageSide === 'obverse' && props.coin.obverseInscription) {
    return props.coin.obverseInscription
  }
  return props.coin.obverseInscription || props.coin.reverseInscription || ''
})
</script>
