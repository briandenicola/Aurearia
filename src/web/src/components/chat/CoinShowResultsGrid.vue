<template>
  <div class="flex w-full flex-col gap-2.5">
    <div
      v-for="(show, j) in shows"
      :key="j"
      class="overflow-hidden rounded-md border border-border-subtle bg-card transition-colors hover:border-border-accent"
    >
      <div class="p-3">
        <SafeExternalLink v-if="safeShowUrl(show.url)" :href="show.url" target="_blank" rel="noopener" class="text-inherit no-underline">
          <h4 class="mb-1 flex items-center gap-1 text-base leading-snug text-gold transition-colors hover:text-bronze">
            {{ show.name }}
            <ExternalLink :size="12" />
          </h4>
        </SafeExternalLink>
        <h4 v-else class="mb-1 text-base leading-snug text-text-primary">{{ show.name }}</h4>
        <div class="mb-1.5 flex flex-col gap-1">
          <span v-if="show.dates" class="flex items-center gap-1.5 text-chip text-text-secondary">
            <Calendar :size="13" />
            {{ show.dates }}
          </span>
          <span v-if="show.venue" class="flex items-center gap-1.5 text-chip text-text-secondary">
            <MapPin :size="13" />
            {{ show.venue }}
          </span>
          <span v-if="show.location" class="pl-6 text-sm text-text-muted">{{ show.location }}</span>
          <span v-if="show.entryFee" class="flex items-center gap-1.5 text-chip text-text-secondary">
            <Ticket :size="13" />
            {{ show.entryFee }}
          </span>
        </div>
        <p v-if="show.description" class="mb-1.5 line-clamp-2 text-sm text-text-secondary">{{ show.description }}</p>
        <div v-if="show.notableDealers?.length" class="flex flex-wrap gap-1">
          <span v-for="(dealer, k) in show.notableDealers" :key="k" class="chip-sm">{{ dealer }}</span>
        </div>
        <button
          class="btn btn-ghost btn-xs mt-2 flex disabled:cursor-default disabled:opacity-60"
          :disabled="savedShows.has(showKey(show)) || savingShow === showKey(show)"
          @click="$emit('save-show', show)"
        >
          <CalendarPlus :size="13" />
          {{ savedShows.has(showKey(show)) ? 'Saved' : savingShow === showKey(show) ? 'Saving...' : 'Save to Calendar' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CoinShow } from '@/types'
import { Calendar, MapPin, Ticket, CalendarPlus, ExternalLink } from 'lucide-vue-next'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'

defineProps<{
  shows: CoinShow[]
  savedShows: Set<string>
  savingShow: string | null
}>()

defineEmits<{
  'save-show': [show: CoinShow]
}>()

function showKey(show: CoinShow): string {
  return `${show.name}|${show.dates}`
}

function safeShowUrl(url: string | null | undefined): string | null {
  return sanitizeExternalUrl(url)
}
</script>
