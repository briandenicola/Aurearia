<template>
  <RouterLink class="draft-card card" :to="`/quick-capture/drafts/${draft.id}`">
    <AuthenticatedImage
      v-if="previewImage"
      :media-path="previewImage.filePath"
      :alt="draft.workingTitle || 'Quick Capture draft preview'"
      class="draft-preview"
    />
    <div v-else class="draft-preview empty">No image</div>
    <div class="draft-info">
      <h3>{{ draft.workingTitle || 'Untitled draft' }}</h3>
      <p>{{ draft.notes || draft.acquisitionSource || 'Incomplete Quick Capture draft' }}</p>
      <div class="draft-meta">
        <span class="chip-sm">{{ draft.status }}</span>
        <span v-if="draft.source === 'find_coin_ai'" class="chip-sm">AI draft</span>
        <span v-if="draft.ngcCertNumber" class="chip-sm">NGC {{ draft.ngcCertNumber }}</span>
        <span class="updated-at">{{ relativeTime }}</span>
      </div>
    </div>
  </RouterLink>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import type { QuickCaptureDraft } from '@/types'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

const props = defineProps<{ draft: QuickCaptureDraft }>()
const previewImage = computed(() => props.draft.images.find(image => image.isPrimary) ?? props.draft.images[0])

const relativeTime = computed(() => {
  const date = new Date(props.draft.updatedAt)
  const diffMs = Date.now() - date.getTime()
  const diffMins = Math.floor(diffMs / 60_000)
  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 30) return `${diffDays}d ago`
  return date.toLocaleDateString()
})
</script>

<style scoped>
.draft-card {
  display: grid;
  grid-template-columns: 76px 1fr;
  gap: 1rem;
  text-decoration: none;
  color: inherit;
}
.draft-preview {
  width: 76px;
  height: 76px;
  border-radius: var(--radius-sm);
  object-fit: cover;
  background: var(--bg-input);
}

.empty {
  display: grid;
  place-items: center;
  color: var(--text-muted);
  font-size: 0.8rem;
}

.draft-info h3, .draft-info p {
  margin: 0 0 0.35rem;
}

.draft-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.updated-at {
  font-size: 0.8rem;
  color: var(--text-muted);
}
</style>
