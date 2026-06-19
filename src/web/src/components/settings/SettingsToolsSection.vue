<template>
  <section class="settings-section card">
    <h2>Image Processor</h2>
    <p class="setting-desc" style="margin-bottom: 1rem">
      Remove backgrounds and crop coin images for your collection.
    </p>
    <ImageProcessor @saved="(coinId: number) => $emit('saved', coinId)" />

    <h3 style="margin-top: 2rem">Blocked Users</h3>
    <p class="setting-desc" style="margin-bottom: 0.75rem">
      Blocked users cannot send you follow requests or view your collection.
    </p>
    <div v-if="blockedUsers.length" class="apikey-list">
      <div v-for="user in blockedUsers" :key="user.id" class="apikey-item">
        <div class="apikey-item-info" style="display: flex; align-items: center; gap: 0.5rem;">
          <AuthenticatedImage
            :media-path="user.avatarPath ? user.avatarPath : '/coin-logo.jpg'"
            :alt="user.username"
            style="width: 28px; height: 28px; border-radius: 50%; object-fit: cover; border: 1px solid var(--border-subtle);"
          />
          <span class="apikey-item-name">{{ user.username }}</span>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="blockedLoading" @click="$emit('unblock', user)">Unblock</button>
      </div>
    </div>
    <p v-else class="setting-desc" style="margin-top: 0.5rem">No blocked users.</p>
  </section>
</template>

<script setup lang="ts">
import ImageProcessor from '@/components/ImageProcessor.vue'
import AuthenticatedImage from '@/components/AuthenticatedImage.vue'

defineProps<{
  blockedUsers: { id: number; username: string; avatarPath: string }[]
  blockedLoading: boolean
}>()

defineEmits<{
  saved: [coinId: number]
  unblock: [user: { id: number; username: string; avatarPath: string }]
}>()
</script>

<style scoped>
.settings-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.settings-section h3 {
  font-size: 0.95rem;
  margin-top: 1.25rem;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.apikey-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.apikey-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.6rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 0.75rem;
}

.apikey-item:last-child {
  border-bottom: none;
}

.apikey-item-info {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.apikey-item-name {
  font-size: 0.9rem;
  font-weight: 500;
}
</style>
