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
      <div v-if="draft.notes" class="draft-context markdown-rendered" v-html="renderedNotes"></div>
      <p v-else class="draft-context">{{ draft.acquisitionSource || 'Incomplete Quick Capture draft' }}</p>
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
import { renderSafeMarkdown } from '@/composables/useMarkdown'

const props = defineProps<{ draft: QuickCaptureDraft }>()
const previewImage = computed(() => props.draft.images.find(image => image.isPrimary) ?? props.draft.images[0])
const renderedNotes = computed(() => renderSafeMarkdown(props.draft.notes))

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
  grid-template-columns: 76px minmax(0, 1fr);
  gap: 1rem;
  text-decoration: none;
  color: inherit;
  box-sizing: border-box;
  max-width: 100%;
  overflow: hidden;
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

.draft-info {
  min-width: 0;
  overflow: hidden;
}

.draft-info h3, .draft-context {
  margin: 0 0 0.35rem;
}

.draft-info h3 {
  overflow-wrap: anywhere;
}

.draft-context {
  color: var(--text-secondary);
  font-size: 0.85rem;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.markdown-rendered {
  max-height: 8.5rem;
  overflow: hidden;
}

/*
 * :deep() audit — markdown-rendered content
 * Target: HTML elements emitted by markdown-it inside .markdown-rendered.
 * Draft body text is rendered by the markdown parser at runtime; the
 * elements carry no Vue-scope hash and are unreachable by scoped selectors
 * or Tailwind utilities.
 */
.markdown-rendered :deep(p),
.markdown-rendered :deep(ul),
.markdown-rendered :deep(ol) {
  margin: 0 0 0.4rem;
}

.markdown-rendered :deep(strong) {
  color: var(--text-primary);
  font-weight: 600;
}

.draft-meta {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
  min-width: 0;
  overflow: hidden;
}

.draft-meta .chip-sm {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.updated-at {
  font-size: 0.8rem;
  color: var(--text-muted);
}

@media (max-width: 600px) {
  .draft-card {
    grid-template-columns: 64px minmax(0, 1fr);
    gap: 0.75rem;
  }

  .draft-preview {
    width: 64px;
    height: 64px;
  }
}
</style>
