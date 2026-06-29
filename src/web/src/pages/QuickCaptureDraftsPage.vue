<template>
  <div class="container">
    <div class="form-wrapper">
      <div class="page-header">
        <h1>Quick Capture Drafts</h1>
        <RouterLink class="btn btn-primary" to="/quick-capture">New Draft</RouterLink>
      </div>
      <p v-if="loading">Loading drafts...</p>
      <p v-else-if="error" class="status-text status-warning">{{ error }}</p>
      <p v-else-if="drafts.length === 0">No active drafts yet.</p>
      <div v-else class="draft-list">
        <QuickCaptureDraftCard v-for="draft in drafts" :key="draft.id" :draft="draft" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { getApiErrorMessage, listQuickCaptureDrafts } from '@/api/client'
import type { QuickCaptureDraft } from '@/types'
import QuickCaptureDraftCard from '@/components/quick-capture/QuickCaptureDraftCard.vue'

const drafts = ref<QuickCaptureDraft[]>([])
const loading = ref(true)
const error = ref('')

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

<style scoped>
.draft-list {
  display: grid;
  gap: 1rem;
}
</style>
