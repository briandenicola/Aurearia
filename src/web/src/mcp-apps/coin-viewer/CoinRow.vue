<template>
  <button type="button" class="row" :aria-label="`Open ${coin.name}`" @click="emit('open', coin.id)">
    <span class="photo" :class="{ loading: thumb === undefined }">
      <img v-if="thumb" :src="thumb" :alt="`${coin.name} photo`">
    </span>
    <span class="text">
      <span class="name">{{ coin.name }}</span>
      <span v-if="line" class="line">{{ line }}</span>
    </span>
    <span class="end">
      <span v-if="value" class="value">{{ value.text }}</span>
      <span v-if="tone || coin.grade" class="tags">
        <span v-if="tone" class="material-dot" :class="tone" :title="coin.material" />
        <span v-if="coin.grade" class="grade">{{ coin.grade }}</span>
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
  thumb: string | null | undefined
}>()

const emit = defineEmits<{ open: [coinId: number] }>()

const line = computed(() => [attribution(props.coin), context(props.coin)].filter(Boolean).join(' — '))
const tone = computed(() => materialTone(props.coin))
const value = computed(() => headlineValue(props.coin))
</script>

<style scoped>
.row {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  width: 100%;
  padding: 8px 10px;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 8px;
}

.row:hover {
  background: var(--viewer-surface-hover);
}

.photo {
  width: 52px;
  height: 52px;
  overflow: hidden;
  background: var(--viewer-photo);
  border-radius: 50%;
}

.photo img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.photo.loading {
  opacity: 0.6;
}

.text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-family-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--viewer-heading);
}

.line {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--viewer-muted);
}

.end {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.value {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  color: var(--viewer-accent);
}

.tags {
  display: flex;
  align-items: center;
  gap: 6px;
}
</style>
