<template>
  <div
    class="card group flex cursor-pointer flex-col overflow-hidden p-0"
    :class="selectable && selected ? 'outline-2 outline-gold outline-offset-[-2px]' : ''"
    @click="handleClick"
  >
    <div class="relative flex aspect-square w-full items-center justify-center overflow-hidden bg-[radial-gradient(ellipse_at_center,var(--bg-secondary)_0%,var(--bg-primary)_100%)] [@media(display-mode:standalone)]:aspect-[5/6]">
      <img v-if="proxiedImageUrl" :src="proxiedImageUrl" :alt="lot.title" class="h-full w-full object-cover transition duration-300 group-hover:scale-105 group-hover:brightness-110" loading="lazy" />
      <div v-else class="text-text-primary/30"><Gavel :size="48" :stroke-width="1" /></div>
      <div class="pointer-events-none absolute inset-0 z-[1] border-b border-gold-dim shadow-[inset_0_0_40px_rgba(0,0,0,0.35)] transition-shadow duration-300 group-hover:shadow-[inset_0_0_25px_rgba(0,0,0,0.2),0_0_20px_var(--accent-gold-glow)]"></div>
      <span
        class="absolute top-2 right-2 z-[2] rounded-full px-2.5 py-[0.2rem] text-label font-semibold uppercase tracking-[0.04em]"
        :class="{
          'bg-input text-text-primary': lot.status === 'watching',
          'bg-gold text-surface': lot.status === 'bidding',
          'bg-greek text-text-primary': lot.status === 'won',
          'bg-byzantine text-text-primary': lot.status === 'lost',
          'bg-text-muted text-surface': lot.status === 'passed',
        }"
      >
        {{ statusLabel }}
      </span>
      <div
        v-if="selectable"
        class="absolute top-2 left-2 z-[4] flex h-6 w-6 items-center justify-center rounded-full border-2 border-white/70 bg-black/40 transition-all"
        :class="selected ? 'border-gold bg-gold text-surface' : ''"
        @click.stop="emit('toggle-select', lot.id)"
      >
        <Check v-if="selected" :size="14" :stroke-width="3" />
      </div>
    </div>
    <div class="flex flex-1 flex-col gap-[0.35rem] p-4">
      <div v-if="needsAttention" class="flex items-center gap-1 text-[0.75rem] font-semibold text-[#f59e0b]" title="This lot's auction has closed but its status hasn't been confirmed yet">
        <AlertTriangle :size="13" /> Needs attention
      </div>
      <h3 class="overflow-hidden text-[0.95rem] leading-[1.3] [display:-webkit-box] [-webkit-box-orient:vertical] [-webkit-line-clamp:2]">{{ lot.title }}</h3>
      <div class="flex flex-wrap gap-2">
        <span v-if="lot.auctionHouse" class="text-[0.78rem] text-text-secondary">{{ lot.auctionHouse }}</span>
        <span v-if="lot.saleName" class="text-[0.78rem] text-text-secondary">{{ lot.saleName }}</span>
        <span v-if="lot.lotNumber" class="text-[0.78rem] font-semibold text-gold">Lot {{ lot.lotNumber }}</span>
      </div>
      <div class="flex flex-wrap gap-2">
        <span class="rounded-full bg-surface px-[0.45rem] py-[0.12rem] text-[0.72rem] text-text-secondary">{{ providerLabel }}</span>
        <span
          v-if="lot.category"
          class="rounded-full bg-surface px-[0.45rem] py-[0.12rem] text-[0.72rem]"
          :class="{
            'text-roman': lot.category.toLowerCase() === 'roman',
            'text-greek': lot.category.toLowerCase() === 'greek',
            'text-byzantine': lot.category.toLowerCase() === 'byzantine',
            'text-modern-cat': lot.category.toLowerCase() === 'modern',
            'text-other': lot.category.toLowerCase() === 'other',
          }"
        >
          {{ lot.category }}
        </span>
        <span v-if="lot.currency && lot.currency !== 'USD'" class="rounded-full bg-surface px-[0.45rem] py-[0.12rem] text-[0.72rem] text-text-secondary">{{ lot.currency }}</span>
      </div>
      <div class="mt-auto flex flex-wrap gap-2 text-[0.82rem]">
        <div v-if="lot.estimate" class="text-text-secondary">Est: {{ formatCurrency(lot.estimate, lot.currency) }}</div>
        <div v-if="lot.initialBid && !lot.winningBid" class="text-text-muted">Start: {{ formatCurrency(lot.initialBid, lot.currency) }}</div>
        <div v-if="lot.currentBid" class="font-semibold text-gold">Bid: {{ formatCurrency(lot.currentBid, lot.currency) }}</div>
        <div v-if="lot.maxBid && lot.status !== 'won'" class="italic text-text-muted">Max: {{ formatCurrency(lot.maxBid, lot.currency) }}</div>
        <div v-if="lot.winningBid" class="font-semibold text-[#4ade80]">Won: {{ formatCurrency(lot.winningBid, lot.currency) }}</div>
        <div v-if="biddingIndicator" :class="biddingIndicator.cls" class="text-[0.78rem] font-semibold">{{ biddingIndicator.label }}</div>
        <div v-if="statusSourceLabel" class="text-[0.78rem] text-text-muted" :title="statusSourceLabel.title">{{ statusSourceLabel.text }}</div>
      </div>
      <div v-if="priceAlerts.length || bidReminders.length" class="flex flex-wrap gap-[0.35rem]" aria-label="Auction alerts">
        <span v-if="priceAlerts.length" class="chip-sm">{{ priceAlerts.length }} price {{ priceAlerts.length === 1 ? 'alert' : 'alerts' }}</span>
        <span v-if="bidReminders.length" class="chip-sm">{{ bidReminders.length }} {{ bidReminders.length === 1 ? 'reminder' : 'reminders' }}</span>
      </div>
      <div v-if="saleCountdown" class="text-sm font-medium text-bronze">{{ saleCountdown }}</div>
      <SafeExternalLink
        v-if="externalUrl"
        :href="externalUrl"
        class="mt-1 text-[0.78rem] text-gold hover:underline"
        target="_blank"
        rel="noopener noreferrer"
        @click.stop
      >
        View on {{ providerLabel }}
      </SafeExternalLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AuctionLot, BidReminder, PriceAlert } from '@/types'
import { computed } from 'vue'
import { Gavel, Check, AlertTriangle } from 'lucide-vue-next'
import { formatCurrency } from '@/utils/format'
import { auctionLotNeedsAttention, auctionLotStatusSourceLabel } from '@/utils/auctionLot'
import { useProxiedImage } from '@/composables/useProxiedImage'
import SafeExternalLink from '@/components/SafeExternalLink.vue'

const props = withDefaults(defineProps<{
  lot: AuctionLot
  selectable?: boolean
  selected?: boolean
  priceAlerts?: PriceAlert[]
  bidReminders?: BidReminder[]
}>(), {
  selectable: false,
  selected: false,
  priceAlerts: () => [],
  bidReminders: () => [],
})
const emit = defineEmits<{
  select: [lot: AuctionLot]
  'toggle-select': [lotId: number]
}>()
const providerLabel = computed(() => props.lot.source === 'cng' ? 'CNG' : 'NumisBids')
const externalUrl = computed(() => props.lot.sourceUrl || props.lot.numisBidsUrl)

function handleClick() {
  if (props.selectable) {
    emit('toggle-select', props.lot.id)
  } else {
    emit('select', props.lot)
  }
}

const lotImageSource = computed(() => props.lot.imageUrl ?? '')
const { proxiedImageUrl } = useProxiedImage(lotImageSource)
const statusLabel = computed(() => {
  const labels: Record<string, string> = {
    watching: 'Watching',
    bidding: 'Bidding',
    won: 'Won',
    lost: 'Lost',
    passed: 'Passed',
  }
  return labels[props.lot.status] ?? props.lot.status
})

const saleCountdown = computed(() => {
  if (!props.lot.saleDate) return null
  const sale = new Date(props.lot.saleDate)
  const now = new Date()
  const diff = sale.getTime() - now.getTime()
  if (diff <= 0) return null
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  if (days > 30) return `${Math.floor(days / 30)}mo away`
  if (days > 0) return `${days}d away`
  const hours = Math.floor(diff / (1000 * 60 * 60))
  return `${hours}h away`
})

const biddingIndicator = computed(() => {
  if (props.lot.status !== 'bidding' || !props.lot.currentBid || !props.lot.maxBid) return null
  if (props.lot.maxBid >= props.lot.currentBid) {
    return { label: 'Winning', cls: 'text-[#4ade80]' }
  }
  return { label: 'Outbid', cls: 'text-[#f87171]' }
})

const needsAttention = computed(() => auctionLotNeedsAttention(props.lot))
const statusSourceLabel = computed(() => auctionLotStatusSourceLabel(props.lot))


</script>
