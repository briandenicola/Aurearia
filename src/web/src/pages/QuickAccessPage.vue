<template>
  <PullToRefresh :on-refresh="refresh">
    <div class="container mx-auto max-w-[900px]">
      <header class="page-header">
        <h1>Quick Access</h1>
        <button
          class="btn btn-ghost btn-sm"
          :disabled="loading"
          :aria-busy="loading"
          :aria-label="loading ? 'Refreshing Quick Access' : 'Refresh Quick Access'"
          @click="refresh"
        >
          <RefreshCw :size="16" :class="{ 'animate-spin': loading }" />
          {{ loading ? 'Refreshing...' : 'Refresh' }}
        </button>
      </header>

      <div v-if="loading && !items.length" class="loading-overlay" role="status" aria-live="polite" aria-label="Loading Quick Access">
        <div class="spinner"></div>
      </div>

      <div v-else>
        <div v-if="error" class="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-sm border border-[var(--color-negative)] bg-card p-3" role="alert">
          <span class="text-body text-[var(--color-negative)]">{{ error }}</span>
          <button class="btn btn-secondary btn-sm" :disabled="loading" @click="refresh">Retry</button>
        </div>

        <BaseEmptyState
          v-if="!items.length && !error"
          title="Nothing pinned yet"
          message="Pin coins, sets, active auction lots, or manual calendar events to keep them close at hand."
          :icon-component="Pin"
        />

        <div v-else-if="items.length" class="grid gap-3" aria-label="Pinned items">
          <article
            v-for="item in items"
            :key="`${item.type}-${item.id}`"
            class="card group grid cursor-pointer grid-cols-[4rem_minmax(0,1fr)_auto] items-center gap-3 p-3 transition-colors hover:border-border-accent hover:bg-card-hover max-sm:grid-cols-[3.25rem_minmax(0,1fr)_auto]"
            tabindex="0"
            @click="openItem(item)"
            @keydown.enter="openItem(item)"
            @keydown.space.prevent="openItem(item)"
          >
            <div
              class="flex h-16 w-16 items-center justify-center overflow-hidden rounded-sm border border-border-subtle bg-input text-text-muted max-sm:h-[3.25rem] max-sm:w-[3.25rem]"
              :style="setAccentStyle(item)"
            >
              <AuthenticatedImage
                v-if="imageFor(item)"
                :media-path="imageFor(item)"
                :alt="titleFor(item)"
                class="h-full w-full object-cover"
              >
                <template #fallback><component :is="iconFor(item)" :size="24" /></template>
              </AuthenticatedImage>
              <component :is="iconFor(item)" v-else :size="24" />
            </div>

            <div class="min-w-0">
              <div class="mb-1 flex flex-wrap items-center gap-2">
                <h2 class="m-0 truncate text-base text-text-primary">{{ titleFor(item) }}</h2>
                <span class="chip-sm">{{ typeLabel(item.type) }}</span>
              </div>
              <p class="m-0 truncate text-body text-text-secondary">{{ metadataFor(item) }}</p>
              <p class="m-0 mt-1 text-sm text-text-muted">Pinned {{ formatDate(item.pinnedAt) }}</p>
            </div>

            <button
              class="inline-flex h-11 w-11 items-center justify-center rounded-sm text-gold transition-colors hover:bg-gold-glow disabled:opacity-55"
              :disabled="isBusy(item.type, item.id).value"
              :aria-busy="isBusy(item.type, item.id).value"
              :aria-label="isBusy(item.type, item.id).value ? `Removing ${titleFor(item)} from Quick Access` : `Unpin ${titleFor(item)} from Quick Access`"
              aria-pressed="true"
              @click.stop="removeItem(item)"
              @keydown.enter.stop
              @keydown.space.stop
            >
              <PinOff :size="20" />
            </button>
          </article>
        </div>
      </div>
    </div>
  </PullToRefresh>
</template>

<script setup lang="ts">
import type { Component, StyleValue } from 'vue'
import { useRouter } from 'vue-router'
import { CalendarDays, CircleDot, Coins, Crown, Gavel, Landmark, Layers3, Pin, PinOff, RefreshCw, Star, Target, Trophy } from 'lucide-vue-next'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'
import PullToRefresh from '@/components/PullToRefresh.vue'
import BaseEmptyState from '@/components/ui/BaseEmptyState.vue'
import { useQuickAccess } from '@/composables/useQuickAccess'
import { useToast } from '@/composables/useToast'
import type { QuickAccessItem, QuickAccessTargetType } from '@/types'

const router = useRouter()
const { showToast } = useToast()
const { items, loading, error, refresh, unpin, isBusy } = useQuickAccess()

function titleFor(item: QuickAccessItem) {
  switch (item.type) {
    case 'coin': return item.coin.name
    case 'coin_set': return item.coinSet.name
    case 'auction_lot': return item.auctionLot.title
    case 'calendar_event': return item.calendarEvent.title
  }
}

function imageFor(item: QuickAccessItem) {
  if (item.type === 'coin') return item.coin.primaryImageUrl
  if (item.type === 'auction_lot') return item.auctionLot.imageUrl
  return null
}

function iconFor(item: QuickAccessItem): Component {
  switch (item.type) {
    case 'coin': return Coins
    case 'coin_set': return setIconFor(item.coinSet.icon)
    case 'auction_lot': return Gavel
    case 'calendar_event': return CalendarDays
  }
}

const setIcons: Record<string, Component> = {
  circledot: CircleDot,
  crown: Crown,
  landmark: Landmark,
  layers3: Layers3,
  star: Star,
  target: Target,
  trophy: Trophy,
}

function setIconFor(icon: string): Component {
  return setIcons[icon.replace(/[^a-z0-9]/gi, '').toLowerCase()] ?? Layers3
}

function setAccentStyle(item: QuickAccessItem): StyleValue | undefined {
  if (item.type !== 'coin_set' || !/^#[0-9a-f]{6}$/i.test(item.coinSet.color)) return undefined
  return { borderColor: item.coinSet.color, color: item.coinSet.color }
}

function typeLabel(type: QuickAccessTargetType) {
  return {
    coin: 'Coin',
    coin_set: 'Set',
    auction_lot: 'Auction Lot',
    calendar_event: 'Calendar Event',
  }[type]
}

function formatOptionalDate(value: string | null) {
  return value ? new Date(value).toLocaleDateString() : 'Date not set'
}

function metadataFor(item: QuickAccessItem) {
  switch (item.type) {
    case 'coin': return item.coin.classification === 'wishlist' ? 'Wishlist coin' : 'Collection coin'
    case 'coin_set': return item.coinSet.setType
    case 'auction_lot': return [item.auctionLot.status, item.auctionLot.auctionHouse, formatOptionalDate(item.auctionLot.auctionEndTime ?? item.auctionLot.saleDate)].filter(Boolean).join(' · ')
    case 'calendar_event': return [item.calendarEvent.auctionHouse, formatOptionalDate(item.calendarEvent.startDate)].filter(Boolean).join(' · ')
  }
}

function formatDate(value: string) {
  return new Date(value).toLocaleDateString()
}

function openItem(item: QuickAccessItem) {
  switch (item.type) {
    case 'coin': void router.push(`/coin/${item.id}`); break
    case 'coin_set': void router.push(`/sets/${item.id}`); break
    case 'auction_lot': void router.push({ path: '/auctions', query: { lot: String(item.id) } }); break
    case 'calendar_event': void router.push({ path: '/calendar', query: { event: String(item.id) } }); break
  }
}

async function removeItem(item: QuickAccessItem) {
  try {
    await unpin(item.type, item.id)
    showToast(`${titleFor(item)} removed from Quick Access`, 'success')
  } catch {
    showToast(error.value || 'Unable to unpin this item.', 'error')
  }
}

</script>
