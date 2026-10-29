<template>
  <article class="detail">
    <div class="media">
      <div class="stage">
        <img v-if="selected" :src="selected.src" :alt="`${coin.name} ${selected.imageType}`">
        <p v-else class="stage-note">{{ stageNote }}</p>
      </div>
      <div v-if="images.length > 1" class="strip" role="group" aria-label="Photos">
        <button
          v-for="(image, index) in images"
          :key="image.imageId"
          type="button"
          class="thumb"
          :aria-pressed="index === selectedIndex"
          :aria-label="`Show ${image.imageType}`"
          @click="selectedIndex = index"
        >
          <img :src="image.src" alt="">
          <span>{{ image.imageType }}</span>
        </button>
      </div>
    </div>

    <div class="info">
      <header>
        <h1>{{ coin.name }}</h1>
        <p v-if="who" class="attribution">{{ who }}</p>
        <p v-if="where" class="context">{{ where }}</p>
      </header>

      <div v-if="values.length" class="values">
        <div v-for="figure in values" :key="figure.label" class="figure">
          <span class="figure-label">{{ figure.label }}</span>
          <span class="figure-text" :class="figure.tone">{{ figure.text }}</span>
        </div>
      </div>

      <section v-for="group in groups" :key="group.title" class="facts">
        <h2>{{ group.title }}</h2>
        <dl>
          <template v-for="[label, text] in group.rows" :key="label">
            <dt>{{ label }}</dt>
            <dd>{{ text }}</dd>
          </template>
        </dl>
      </section>

      <p v-if="missing.length" class="missing">Not recorded: {{ missing.join(', ') }}</p>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { CoinImage, CoinSummary } from './types'
import { attribution, context, factGroups, missingLabels, valueFigures } from './viewer-data'

const props = defineProps<{
  coin: CoinSummary
  images: CoinImage[]
  galleryState: 'loading' | 'ready' | 'error'
}>()

const selectedIndex = ref(0)
watch(() => props.images, () => {
  selectedIndex.value = 0
})

const selected = computed(() => props.images[selectedIndex.value] ?? null)
const stageNote = computed(() =>
  props.galleryState === 'loading' ? 'Loading photos…' : props.galleryState === 'error' ? 'Photos unavailable.' : 'No photos.',
)
const who = computed(() => attribution(props.coin))
const where = computed(() => context(props.coin))
const values = computed(() => valueFigures(props.coin))
const groups = computed(() => factGroups(props.coin))
const missing = computed(() => missingLabels(props.coin))
</script>

<style scoped>
.detail {
  display: grid;
  grid-template-columns: minmax(0, 5fr) minmax(0, 6fr);
  gap: 20px;
}

@media (max-width: 560px) {
  .detail {
    grid-template-columns: 1fr;
  }
}

.media {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stage {
  display: grid;
  place-items: center;
  aspect-ratio: 1;
  overflow: hidden;
  background: var(--viewer-photo);
  border-radius: var(--viewer-radius);
}

.stage img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.stage-note {
  margin: 0;
  font-size: 13px;
  color: var(--viewer-faint);
}

.strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.thumb {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 0;
  cursor: pointer;
  background: none;
  border: 0;
}

.thumb img {
  width: 56px;
  height: 56px;
  object-fit: contain;
  background: var(--viewer-photo);
  border: 2px solid transparent;
  border-radius: 8px;
}

.thumb[aria-pressed='true'] img {
  border-color: var(--viewer-accent);
}

.thumb span {
  font-size: 11px;
  color: var(--viewer-muted);
  text-transform: capitalize;
}

.info {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

header p {
  margin: 0;
}

h1 {
  margin: 0 0 4px;
  font-family: var(--font-family-display);
  font-size: 22px;
  font-weight: 600;
  line-height: 1.2;
  color: var(--viewer-heading);
}

.attribution {
  font-size: 14px;
}

.context {
  font-size: 13px;
  color: var(--viewer-muted);
}

.values {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 20px;
  padding: 10px 12px;
  background: var(--viewer-surface);
  border: 1px solid var(--viewer-border);
  border-radius: var(--viewer-radius);
}

.figure {
  display: flex;
  flex-direction: column;
}

.figure-label {
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--viewer-faint);
}

.figure-text {
  font-size: 16px;
  font-weight: 600;
  color: var(--viewer-accent);
}

.figure-text.up {
  color: var(--viewer-up);
}

.figure-text.down {
  color: var(--viewer-down);
}

h2 {
  margin: 0 0 6px;
  padding-bottom: 4px;
  font-family: var(--font-family-display);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--viewer-muted);
  border-bottom: 1px solid var(--viewer-border);
}

dl {
  display: grid;
  /* Fixed label column keeps every group's values aligned. */
  grid-template-columns: 7.5em 1fr;
  gap: 4px 16px;
  margin: 0;
}

dt {
  color: var(--viewer-muted);
}

dd {
  margin: 0;
}

.missing {
  margin: 0;
  font-size: 12px;
  color: var(--viewer-faint);
}
</style>
