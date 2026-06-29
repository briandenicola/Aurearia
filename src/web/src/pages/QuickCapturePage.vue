<template>
  <div class="container">
    <div class="form-wrapper">
      <div class="page-header">
        <h1>Quick Capture</h1>
        <div v-if="isPwa" class="pwa-actions">
          <RouterLink class="pwa-icon-btn" to="/quick-capture/drafts" title="All captures" aria-label="All captures">
            <List :size="22" />
          </RouterLink>
        </div>
        <div v-else class="header-actions">
          <RouterLink class="btn btn-secondary" to="/quick-capture/drafts">
            <List :size="16" /> All
          </RouterLink>
        </div>
      </div>
      <p class="intro">Capture sparse coin details quickly. Drafts remain active and incomplete until you finish them later.</p>
      <QuickCaptureForm @saved="onSaved" />
      <div v-if="lastDraft" class="card saved-card">
        <strong>Draft saved.</strong>
        <span>Draft #{{ lastDraft.id }} is active and excluded from collection, wishlist, sold, stats, and health counts.</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import { List } from 'lucide-vue-next'
import QuickCaptureForm from '@/components/quick-capture/QuickCaptureForm.vue'
import type { QuickCaptureDraft } from '@/types'
import { usePwa } from '@/composables/usePwa'

const lastDraft = ref<QuickCaptureDraft | null>(null)
const { isPwa } = usePwa()

function onSaved(draft: QuickCaptureDraft) {
  lastDraft.value = draft
}
</script>

<style scoped>
.intro, .saved-card {
  margin-bottom: 1rem;
}
.saved-card {
  display: grid;
  gap: 0.25rem;
}
</style>
