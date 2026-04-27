<template>
  <div class="image-gallery">
    <div v-if="images.length" class="gallery-main">
      <img :src="activeImageSrc" :alt="activeImage?.imageType" class="gallery-active-img" />
      <span class="gallery-type-badge">{{ activeImage?.imageType }}</span>
      <div v-if="activeImage && !processing" class="gallery-action-btns">
        <button
          class="gallery-action-btn"
          title="Remove background"
          @click="$emit('removeBg', activeImage!)"
        >
          ✨ Remove BG
        </button>
        <button
          class="gallery-action-btn gallery-delete-btn"
          title="Delete image"
          @click="$emit('deleteImage', activeImage!)"
        >
          🗑 Delete
        </button>
      </div>
      <div v-if="processing" class="gallery-processing-overlay">
        <div class="spinner"></div>
        <p>Removing background...</p>
        <p class="processing-hint">First run downloads the ML model (~40MB)</p>
      </div>
    </div>
    <div v-else class="gallery-empty">
      <span class="empty-icon"><Coins :size="48" :stroke-width="1" /></span>
      <p>No images uploaded</p>
    </div>
    <div v-if="images.length > 1" class="gallery-thumbs">
      <button
        v-for="img in images"
        :key="img.id"
        class="thumb-btn"
        :class="{ active: activeImage?.id === img.id }"
        @click="activeImage = img"
      >
        <img :src="`/uploads/${img.filePath}`" :alt="img.imageType" class="thumb-img" />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { CoinImage } from '@/types'
import { Coins } from 'lucide-vue-next'

const props = defineProps<{
  images: CoinImage[]
  processing?: boolean
}>()

defineEmits<{
  removeBg: [image: CoinImage]
  deleteImage: [image: CoinImage]
}>()

const activeImage = ref<CoinImage | null>(null)

watch(
  () => props.images,
  (imgs) => {
    if (imgs.length) {
      // Keep current selection if still present, otherwise pick primary/first
      if (activeImage.value && imgs.find((i) => i.id === activeImage.value!.id)) return
      activeImage.value = imgs.find((i) => i.isPrimary) || imgs[0] || null
    }
  },
  { immediate: true },
)

const activeImageSrc = computed(() => {
  return activeImage.value ? `/uploads/${activeImage.value.filePath}` : ''
})
</script>

<style scoped>
.gallery-main {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  border-radius: var(--radius-md);
  overflow: hidden;
  background: var(--bg-primary);
}

.gallery-active-img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.gallery-type-badge {
  position: absolute;
  bottom: 0.5rem;
  left: 0.5rem;
  padding: 0.15rem 0.5rem;
  background: rgba(0, 0, 0, 0.7);
  color: var(--accent-gold);
  font-size: 0.75rem;
  border-radius: var(--radius-full);
  border: 1px solid rgba(255, 255, 255, 0.15);
  text-transform: capitalize;
  backdrop-filter: blur(4px);
}

.gallery-action-btns {
  position: absolute;
  bottom: 0.5rem;
  right: 0.5rem;
  display: flex;
  gap: 0.35rem;
}

.gallery-action-btn {
  padding: 0.3rem 0.7rem;
  background: rgba(0, 0, 0, 0.75);
  color: var(--accent-gold);
  font-size: 0.75rem;
  font-weight: 500;
  border: 1px solid var(--accent-gold-dim);
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--transition-fast);
  backdrop-filter: blur(4px);
}

.gallery-action-btn:hover {
  background: var(--accent-gold);
  color: #000;
  border-color: var(--accent-gold);
}

.gallery-delete-btn:hover {
  background: #e74c3c;
  color: #fff;
  border-color: #e74c3c;
}

.gallery-processing-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  background: rgba(0, 0, 0, 0.7);
  color: var(--accent-gold);
  border-radius: var(--radius-md);
  backdrop-filter: blur(4px);
}

.gallery-processing-overlay p {
  font-size: 0.9rem;
}

.gallery-processing-overlay .processing-hint {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.gallery-empty {
  width: 100%;
  aspect-ratio: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--bg-primary);
  border-radius: var(--radius-md);
  border: 2px dashed var(--border-subtle);
  color: var(--text-muted);
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 0.5rem;
}

.gallery-thumbs {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.75rem;
  overflow-x: auto;
  padding-bottom: 0.25rem;
}

.thumb-btn {
  flex-shrink: 0;
  width: 60px;
  height: 60px;
  border: 2px solid transparent;
  border-radius: var(--radius-sm);
  overflow: hidden;
  cursor: pointer;
  background: var(--bg-primary);
  padding: 0;
  transition: border-color var(--transition-fast);
}

.thumb-btn.active,
.thumb-btn:hover {
  border-color: var(--accent-gold);
}

.thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
