<template>
  <button type="button" class="card" :aria-label="`Open ${coin.name}`" @click="emit('open', coin.id)">
    <span class="thumb">
      <img v-if="thumb" :src="thumb" :alt="`${coin.name} photo`">
      <span v-else class="thumb-note">{{ thumb === null ? 'No photo' : '…' }}</span>
    </span>
    <span class="body">
      <span class="name">{{ coin.name }}</span>
      <span v-if="caption" class="sub">{{ caption }}</span>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CoinSummary } from './types'
import { subtitle } from './viewer-data'

const props = defineProps<{
  coin: CoinSummary
  // undefined while loading, null when the coin has no photo.
  thumb: string | null | undefined
}>()

const emit = defineEmits<{ open: [coinId: number] }>()

const caption = computed(() => subtitle(props.coin))
</script>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
  text-align: left;
  font: inherit;
  color: inherit;
  cursor: pointer;
  background: var(--viewer-surface);
  border: 1px solid var(--viewer-border);
  border-radius: var(--viewer-radius);
}

.card:hover,
.card:focus-visible {
  border-color: var(--viewer-accent);
  outline: none;
}

.thumb {
  display: grid;
  place-items: center;
  aspect-ratio: 1;
  background: var(--viewer-placeholder);
}

.thumb img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumb-note {
  font-size: 12px;
  color: var(--viewer-muted);
}

.body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px 10px;
}

.name {
  font-weight: 600;
}

.sub {
  font-size: 12px;
  color: var(--viewer-muted);
}
</style>
