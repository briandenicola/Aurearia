<template>
  <RouterLink
    class="card grid max-w-full grid-cols-[64px_minmax(0,1fr)] gap-3 overflow-hidden no-underline text-text-primary min-[601px]:grid-cols-[76px_minmax(0,1fr)] min-[601px]:gap-4"
    :to="`/quick-capture/drafts/${draft.id}`"
  >
    <AuthenticatedImage
      v-if="previewImage"
      :media-path="previewImage.filePath"
      :alt="draft.workingTitle || 'Quick Capture draft preview'"
      class="h-16 w-16 rounded-sm bg-input object-cover min-[601px]:h-[76px] min-[601px]:w-[76px]"
    />
    <div v-else class="grid h-16 w-16 place-items-center rounded-sm bg-input text-sm text-text-muted min-[601px]:h-[76px] min-[601px]:w-[76px]">No image</div>
    <div class="min-w-0 overflow-hidden">
      <h3 class="mb-[0.35rem] break-words">{{ draft.workingTitle || 'Untitled draft' }}</h3>
      <div
        v-if="draft.notes"
        class="markdown-rendered mb-[0.35rem] max-h-[8.5rem] overflow-hidden break-words text-body leading-[1.4] text-text-secondary [&_ol]:mb-[0.4rem] [&_p]:mb-[0.4rem] [&_strong]:font-semibold [&_strong]:text-text-primary [&_ul]:mb-[0.4rem]"
        v-html="renderedNotes"
      ></div>
      <p v-else class="mb-[0.35rem] break-words text-body leading-[1.4] text-text-secondary">{{ draft.acquisitionSource || 'Incomplete Quick Capture draft' }}</p>
      <div class="flex min-w-0 flex-wrap items-center gap-2 overflow-hidden">
        <span class="chip-sm inline-block max-w-full truncate align-middle">{{ draft.status }}</span>
        <span v-if="draft.source === 'find_coin_ai'" class="chip-sm inline-block max-w-full truncate align-middle">AI draft</span>
        <span v-if="draft.ngcCertNumber" class="chip-sm inline-block max-w-full truncate align-middle">NGC {{ draft.ngcCertNumber }}</span>
        <span class="text-sm text-text-muted">{{ relativeTime }}</span>
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
/*
 * :deep() audit — markdown-rendered content
 * Target: HTML elements emitted by markdown-it inside .markdown-rendered.
 * Draft body text is rendered by the markdown parser at runtime; the
 * resulting nodes do not carry Vue scope attributes and cannot be styled
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
</style>
