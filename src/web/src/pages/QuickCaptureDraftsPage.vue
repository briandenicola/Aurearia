<template>
  <div class="container">
    <div class="form-wrapper grid gap-4">
      <div class="page-header">
        <h1>Quick Capture</h1>
        <div v-if="isPwa" class="pwa-actions">
          <RouterLink class="pwa-icon-btn" to="/quick-capture" title="New capture" aria-label="New capture">
            <CirclePlus :size="22" />
          </RouterLink>
        </div>
        <div v-else class="header-actions">
          <RouterLink class="btn btn-primary" to="/quick-capture">
            <Plus :size="16" /> New
          </RouterLink>
        </div>
      </div>
      <p v-if="loading" class="m-0 text-base text-text-secondary">Loading drafts...</p>
      <p v-else-if="error" class="m-0 text-chip text-warning">{{ error }}</p>
      <p v-else-if="drafts.length === 0" class="m-0 text-base text-text-secondary">No active drafts yet.</p>
      <div v-else class="grid gap-4">
        <QuickCaptureDraftCard v-for="draft in drafts" :key="draft.id" :draft="draft" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { CirclePlus, Plus } from 'lucide-vue-next'
import { getApiErrorMessage, listQuickCaptureDrafts } from '@/api/client'
import type { QuickCaptureDraft } from '@/types'
import QuickCaptureDraftCard from '@/components/quick-capture/QuickCaptureDraftCard.vue'
import { usePwa } from '@/composables/usePwa'

const drafts = ref<QuickCaptureDraft[]>([])
const loading = ref(true)
const error = ref('')
const { isPwa } = usePwa()

onMounted(async () => {
  try {
    const response = await listQuickCaptureDrafts({ status: 'active', limit: 50 })
    drafts.value = response.data.drafts
  } catch (err) {
    error.value = getApiErrorMessage(err) || 'Unable to load quick capture drafts.'
  } finally {
    loading.value = false
  }
})
</script>
