<template>
  <div class="tray-item">
    <span v-if="displayName" class="tray-name">{{ displayName }}</span>
    <div
      class="tray-well"
      :class="{
        'is-interactive': interactive,
        'is-placeholder': coin.placeholder,
        'is-wishlist-placeholder': coin.wishlistPlaceholder,
      }"
      :style="{ width: `${renderSizePx}px`, height: `${renderSizePx}px` }"
      :aria-label="coin.name"
      :tabindex="interactive ? 0 : undefined"
      :role="interactive ? 'button' : undefined"
      @click="handleClick"
      @keydown.enter="handleClick"
    >
      <div class="well-container">
        <img
          v-if="resolvedImageSrc"
          :src="resolvedImageSrc"
          :alt="coin.name"
          class="well-coin"
          loading="eager"
          decoding="async"
        />
        <AuthenticatedImage
          v-else-if="primaryImage"
          :media-path="primaryImage"
          :alt="coin.name"
          class="well-coin"
          loading="eager"
          decoding="async"
        />
        <div v-else class="well-placeholder">
          <span
            v-if="coin.placeholderLabel"
            class="placeholder-label"
            :style="{ fontSize: `${placeholderFontSizePx}px` }"
          >{{ coin.placeholderLabel }}</span>
          <Coins v-else :size="Math.floor(renderSizePx * 0.4)" :stroke-width="1" />
        </div>
      </div>
    </div>
    <span v-if="displayCaption" class="tray-date">{{ displayCaption }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Coins } from 'lucide-vue-next'
import { selectTrayCoinImage, type TrayCoin, type TrayCoinFace } from '@/utils/trayLayout'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

interface Props {
  coin: TrayCoin
  renderSizePx: number
  imageSrcResolver?: (filePath: string) => string
  interactive?: boolean
  showCaptions?: boolean
  showNames?: boolean
  preferredFace?: TrayCoinFace
}

const props = withDefaults(defineProps<Props>(), {
  imageSrcResolver: undefined,
  interactive: true,
  showCaptions: true,
  showNames: false,
  preferredFace: 'obverse',
})
const emit = defineEmits<{
  'coin-clicked': [coinId: number]
}>()

const selectedImagePath = computed(() => selectTrayCoinImage(props.coin.images, props.preferredFace)?.filePath ?? null)

const primaryImage = computed(() => {
  const path = selectedImagePath.value
  if (!path) return null
  // Preserve absolute URLs; prefix relative paths with /uploads/
  if (path.startsWith('/') || path.startsWith('http://') || path.startsWith('https://')) {
    return path
  }
  return `/uploads/${path}`
})

const resolvedImageSrc = computed(() => {
  const path = selectedImagePath.value
  if (!path || !props.imageSrcResolver) return null
  return props.imageSrcResolver(path)
})

const placeholderFontSizePx = computed(() => Math.max(9, Math.min(13, Math.round(props.renderSizePx * 0.115))))

const displayCaption = computed(() => {
  if (!props.showCaptions) return null
  if (props.coin.placeholder) return null
  if (props.coin.wishlistPlaceholder) return 'TBD'
  return props.coin.purchaseDate?.slice(0, 10) || 'TBD'
})

const displayName = computed(() => {
  if (!props.showNames) return null
  if (props.coin.placeholder) return null
  return props.coin.name
})

function handleClick() {
  if (!props.interactive) return
  emit('coin-clicked', props.coin.id)
}
</script>

<style scoped>
.tray-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
}

.tray-name {
  min-height: 1rem;
  max-width: 100%;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1;
  color: var(--text-primary);
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tray-well {
  position: relative;
  transition: var(--transition-fast);
  border-radius: 50%;
  background: radial-gradient(
    circle at center,
    rgba(0, 0, 0, 0.3) 0%,
    rgba(0, 0, 0, 0.15) 40%,
    transparent 70%
  );
  padding: 8%;
  display: flex;
  align-items: center;
  justify-content: center;
  /* Caps how large zoom can render a well at each grid breakpoint below, so a
     high size-scale can't overflow the tray's fixed column count. */
  max-width: 120px;
  max-height: 120px;
}

@media (min-width: 576px) {
  .tray-well {
    max-width: 170px;
    max-height: 170px;
  }
}

@media (min-width: 768px) {
  .tray-well {
    max-width: 240px;
    max-height: 240px;
  }
}

.tray-well.is-interactive {
  cursor: pointer;
}

.tray-well.is-interactive:hover {
  transform: translateY(-2px);
  filter: brightness(1.1);
}

.tray-well.is-placeholder {
  border: 1px dashed var(--border-accent);
  background: radial-gradient(
    circle at center,
    var(--accent-gold-glow) 0%,
    rgba(0, 0, 0, 0.15) 48%,
    transparent 72%
  );
}

.tray-well.is-wishlist-placeholder {
  opacity: 0.24;
}

.tray-well.is-wishlist-placeholder .well-container {
  border: 1px dashed var(--border-accent);
  filter: grayscale(0.35) saturate(0.75);
}

.tray-well:focus-visible {
  outline: 2px solid var(--accent-gold);
  outline-offset: 4px;
}

.well-container {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 
    0 2px 4px rgba(0, 0, 0, 0.3),
    0 4px 8px rgba(0, 0, 0, 0.2),
    inset 0 1px 2px rgba(255, 255, 255, 0.1);
}

.well-coin {
  width: 100%;
  height: 100%;
  object-fit: cover;
  filter: drop-shadow(0 2px 3px rgba(0, 0, 0, 0.4));
}

.well-placeholder {
  color: var(--text-muted);
  opacity: 0.5;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 16%;
  box-sizing: border-box;
  overflow: hidden;
}

.is-placeholder .well-placeholder {
  color: var(--accent-gold);
  opacity: 1;
}

.placeholder-label {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 4;
  overflow: hidden;
  font-weight: 600;
  line-height: 1.2;
  text-align: center;
  word-break: break-word;
  overflow-wrap: break-word;
  hyphens: auto;
}

.tray-date {
  min-height: 1rem;
  font-size: 0.75rem;
  font-weight: 500;
  line-height: 1;
  color: var(--text-secondary);
}

@media (prefers-reduced-motion: reduce) {
  .tray-well {
    transition: none;
  }
  .tray-well.is-interactive:hover {
    transform: none;
  }
}
</style>
