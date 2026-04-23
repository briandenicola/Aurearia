<template>
  <div class="lot-card card" :class="{ 'lot-card-selected': selectable && selected }" @click="handleClick">
    <div class="lot-image-container">
      <img v-if="lot.imageUrl" :src="proxiedImageUrl" :alt="lot.title" class="lot-image" loading="lazy" />
      <div v-else class="lot-image-placeholder"><Gavel :size="48" :stroke-width="1" /></div>
      <span class="lot-status-badge" :class="`status-${lot.status}`">{{ statusLabel }}</span>
      <div v-if="selectable" class="select-checkbox" :class="{ checked: selected }" @click.stop="emit('toggle-select', lot.id)">
        <Check v-if="selected" :size="14" :stroke-width="3" />
      </div>
    </div>
    <div class="lot-body">
      <h3 class="lot-title">{{ lot.title }}</h3>
      <div class="lot-meta">
        <span v-if="lot.auctionHouse" class="meta-item">{{ lot.auctionHouse }}</span>
        <span v-if="lot.saleName" class="meta-item">{{ lot.saleName }}</span>
        <span v-if="lot.lotNumber" class="meta-item lot-number">Lot {{ lot.lotNumber }}</span>
      </div>
      <div class="lot-details">
        <span v-if="lot.category" class="detail" :class="`category-${lot.category.toLowerCase()}`">{{ lot.category }}</span>
        <span v-if="lot.currency && lot.currency !== 'USD'" class="detail">{{ lot.currency }}</span>
      </div>
      <div class="lot-pricing">
        <div v-if="lot.estimate" class="lot-estimate">Est: {{ formatCurrency(lot.estimate, lot.currency) }}</div>
        <div v-if="lot.currentBid" class="lot-bid">Bid: {{ formatCurrency(lot.currentBid, lot.currency) }}</div>
        <div v-if="lot.maxBid" class="lot-max-bid">Max: {{ formatCurrency(lot.maxBid, lot.currency) }}</div>
      </div>
      <div v-if="saleCountdown" class="lot-countdown">{{ saleCountdown }}</div>
      <a
        :href="lot.numisBidsUrl"
        class="lot-link"
        target="_blank"
        rel="noopener noreferrer"
        @click.stop
      >
        View on NumisBids
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AuctionLot } from '@/types'
import { computed } from 'vue'
import { Gavel, Check } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  lot: AuctionLot
  selectable?: boolean
  selected?: boolean
}>(), {
  selectable: false,
  selected: false,
})
const emit = defineEmits<{
  select: [lot: AuctionLot]
  'toggle-select': [lotId: number]
}>()

function handleClick() {
  if (props.selectable) {
    emit('toggle-select', props.lot.id)
  } else {
    emit('select', props.lot)
  }
}

const API_BASE = import.meta.env.VITE_API_BASE_URL || ''

const proxiedImageUrl = computed(() => {
  if (!props.lot.imageUrl) return ''
  const token = localStorage.getItem('token') ?? ''
  return `${API_BASE}/api/proxy-image?url=${encodeURIComponent(props.lot.imageUrl)}&token=${encodeURIComponent(token)}`
})
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

function formatCurrency(value: number, currency?: string) {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency || 'USD' }).format(value)
}
</script>

<style scoped>
.lot-card {
  cursor: pointer;
  overflow: hidden;
  padding: 0;
  display: flex;
  flex-direction: column;
}

.lot-image-container {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  overflow: hidden;
  background: radial-gradient(ellipse at center, var(--bg-secondary) 0%, var(--bg-primary) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
}

.lot-image-container::after {
  content: '';
  position: absolute;
  inset: 0;
  box-shadow: inset 0 0 40px rgba(0, 0, 0, 0.35);
  border-bottom: 1px solid var(--accent-gold-dim);
  pointer-events: none;
  z-index: 1;
  transition: box-shadow var(--transition-med);
}

.lot-card:hover .lot-image-container::after {
  box-shadow: inset 0 0 25px rgba(0, 0, 0, 0.2), 0 0 20px var(--accent-gold-glow);
}

.lot-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform var(--transition-med), filter var(--transition-med);
}

.lot-card:hover .lot-image {
  transform: scale(1.05);
  filter: brightness(1.1);
}

.lot-image-placeholder {
  opacity: 0.3;
}

.lot-status-badge {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  z-index: 2;
}

.status-watching {
  background: rgba(100, 150, 255, 0.85);
  color: #fff;
}

.status-bidding {
  background: rgba(201, 168, 76, 0.9);
  color: #1a1a2e;
}

.status-won {
  background: rgba(74, 222, 128, 0.85);
  color: #1a1a2e;
}

.status-lost {
  background: rgba(248, 113, 113, 0.8);
  color: #fff;
}

.status-passed {
  background: rgba(120, 120, 120, 0.8);
  color: #fff;
}

.lot-body {
  padding: 1rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.lot-title {
  font-size: 0.95rem;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.lot-meta {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.meta-item {
  font-size: 0.78rem;
  color: var(--text-secondary);
}

.lot-number {
  font-weight: 600;
  color: var(--accent-gold);
}

.lot-details {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.detail {
  font-size: 0.72rem;
  padding: 0.12rem 0.45rem;
  background: var(--bg-primary);
  border-radius: var(--radius-full);
  color: var(--text-secondary);
}

.lot-pricing {
  display: flex;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-top: auto;
  font-size: 0.82rem;
}

.lot-estimate {
  color: var(--text-secondary);
}

.lot-bid {
  font-weight: 600;
  color: var(--accent-gold);
}

.lot-max-bid {
  color: var(--text-muted);
  font-style: italic;
}

.lot-countdown {
  font-size: 0.75rem;
  color: var(--accent-bronze);
  font-weight: 500;
}

.lot-link {
  font-size: 0.78rem;
  color: var(--accent-gold);
  text-decoration: none;
  margin-top: 0.25rem;
}

.lot-link:hover {
  text-decoration: underline;
}

.category-roman { color: #b57edc; }
.category-greek { color: #9ab85a; }
.category-byzantine { color: #e67e73; }
.category-modern { color: #7ab3d4; }
.category-other { color: #aaa; }

@media (display-mode: standalone) {
  .lot-image-container {
    aspect-ratio: 5 / 6;
  }
}

.select-checkbox {
  position: absolute;
  top: 0.5rem;
  left: 0.5rem;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, 0.7);
  background: rgba(0, 0, 0, 0.4);
  z-index: 4;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}

.select-checkbox.checked {
  background: var(--accent-gold);
  border-color: var(--accent-gold);
  color: #000;
}

.lot-card-selected {
  outline: 2px solid var(--accent-gold);
  outline-offset: -2px;
}
</style>
