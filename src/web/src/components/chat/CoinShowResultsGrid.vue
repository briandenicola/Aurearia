<template>
  <div class="suggestions-grid">
    <div v-for="(show, j) in shows" :key="j" class="show-card">
      <div class="show-body">
        <SafeExternalLink v-if="safeShowUrl(show.url)" :href="show.url" target="_blank" rel="noopener" class="show-name-link">
          <h4>{{ show.name }} <ExternalLink :size="12" /></h4>
        </SafeExternalLink>
        <h4 v-else>{{ show.name }}</h4>
        <div class="show-details">
          <span v-if="show.dates" class="show-detail"><Calendar :size="13" /> {{ show.dates }}</span>
          <span v-if="show.venue" class="show-detail"><MapPin :size="13" /> {{ show.venue }}</span>
          <span v-if="show.location" class="show-detail-sub">{{ show.location }}</span>
          <span v-if="show.entryFee" class="show-detail"><Ticket :size="13" /> {{ show.entryFee }}</span>
        </div>
        <p v-if="show.description" class="show-desc">{{ show.description }}</p>
        <div v-if="show.notableDealers?.length" class="show-dealers">
          <span v-for="(dealer, k) in show.notableDealers" :key="k" class="meta-tag">{{ dealer }}</span>
        </div>
        <button
          class="save-cal-btn"
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

<style scoped>
.suggestions-grid {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
  width: 100%;
}

.show-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: border-color var(--transition-fast);
}

.show-card:hover {
  border-color: var(--accent-gold);
}

.show-body {
  padding: 0.7rem 0.85rem;
}

.show-body h4 {
  font-size: 0.88rem;
  margin: 0 0 0.4rem;
  color: var(--text-primary);
  line-height: 1.3;
}

.show-name-link {
  text-decoration: none;
  color: inherit;
}

.show-name-link h4 {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--accent-gold);
  transition: color var(--transition-fast);
}

.show-name-link:hover h4 {
  color: var(--accent-bronze);
}

.show-details {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-bottom: 0.4rem;
}

.show-detail {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.show-detail-sub {
  font-size: 0.75rem;
  color: var(--text-muted);
  padding-left: 1.5rem;
}

.show-desc {
  font-size: 0.78rem;
  color: var(--text-secondary);
  margin: 0 0 0.4rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.show-dealers {
  display: flex;
  flex-wrap: wrap;
  gap: 0.25rem;
}

.meta-tag {
  font-size: 0.7rem;
  padding: 0.1rem 0.4rem;
  border-radius: var(--radius-full);
  background: var(--bg-body);
  color: var(--text-muted);
  border: 1px solid var(--border-subtle);
}

.save-cal-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.75rem;
  padding: 0.3rem 0.65rem;
  margin-top: 0.5rem;
  background: transparent;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: inherit;
}

.save-cal-btn:hover:not(:disabled) {
  border-color: var(--accent-gold-dim);
  color: var(--accent-gold);
  background: var(--accent-gold-glow);
}

.save-cal-btn:disabled {
  opacity: 0.6;
  cursor: default;
}
</style>
