<template>
  <button type="button" class="card" :aria-label="`Open ${coin.name}`" @click="emit('open', coin.id)">
    <span class="photo" :class="{ loading: thumb === undefined }">
      <img v-if="thumb" :src="thumb" :alt="`${coin.name} photo`">
      <span v-else-if="thumb === null" class="photo-note">No photo</span>
    </span>
    <span class="body">
      <span class="name">{{ coin.name }}</span>
      <span v-if="who" class="attribution">{{ who }}</span>
      <span v-if="where" class="context">{{ where }}</span>
      <span v-if="value || coin.grade || tone" class="footer">
        <span class="tags">
          <span v-if="tone" class="material" :title="coin.material"><span class="material-dot" :class="tone" />{{ coin.material }}</span>
          <span v-if="coin.grade" class="grade">{{ coin.grade }}</span>
        </span>
        <span v-if="value" class="value"><span class="value-label">{{ value.label }}</span> {{ value.text }}</span>
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { CoinSummary } from './types'
import { attribution, context, headlineValue, materialTone } from './viewer-data'

const props = defineProps<{
  coin: CoinSummary
  // undefined while loading, null when the coin has no photo.
  thumb: string | null | undefined
}>()

const emit = defineEmits<{ open: [coinId: number] }>()

const who = computed(() => attribution(props.coin))
const where = computed(() => context(props.coin))
const tone = computed(() => materialTone(props.coin))
const value = computed(() => headlineValue(props.coin))
</script>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  padding: 0;
  overflow: hidden;
  text-align: left;
  cursor: pointer;
  background: var(--viewer-surface);
  border: 1px solid var(--viewer-border);
  border-radius: var(--viewer-radius);
  transition: border-color 120ms ease, transform 120ms ease;
}

.card:hover {
  border-color: var(--viewer-border-strong);
  transform: translateY(-1px);
}

.photo {
  display: grid;
  place-items: center;
  aspect-ratio: 1;
  background: var(--viewer-photo);
}

.photo img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.photo.loading {
  background: linear-gradient(100deg, var(--viewer-photo) 40%, var(--viewer-surface-hover) 50%, var(--viewer-photo) 60%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

@keyframes shimmer {
  to { background-position: -200% 0; }
}

@media (prefers-reduced-motion: reduce) {
  .card,
  .photo.loading {
    transition: none;
    animation: none;
  }
}

.photo-note {
  font-size: 12px;
  color: var(--viewer-faint);
}

.body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px 12px;
}

.name {
  display: -webkit-box;
  overflow: hidden;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  font-family: var(--font-family-display);
  font-size: 15px;
  font-weight: 600;
  line-height: 1.25;
  color: var(--viewer-heading);
}

.attribution {
  font-size: 13px;
  color: var(--viewer-text);
}

.context {
  font-size: 12px;
  color: var(--viewer-muted);
}

.footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: auto;
  padding-top: 8px;
}

.tags {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.material {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--viewer-muted);
}

.value {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  color: var(--viewer-accent);
}

.value-label {
  font-size: 11px;
  font-weight: 400;
  color: var(--viewer-faint);
}
</style>
