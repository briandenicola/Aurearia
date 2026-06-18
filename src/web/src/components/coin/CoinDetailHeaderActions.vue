<template>
  <div class="detail-header">
    <div class="detail-navigation">
      <button class="btn btn-ghost btn-xs back-action" @click="router.push('/')">
        <ArrowLeft :size="14" />
        Back to Gallery
      </button>
      <button class="btn btn-secondary btn-xs share-action" :disabled="sharing" @click="$emit('share')">
        <Share2 :size="14" />
        {{ sharing ? 'Sharing...' : 'Share' }}
      </button>
    </div>
    <div class="detail-actions">
      <button v-if="!isWishlist && !isSold" class="btn btn-secondary btn-xs" @click="$emit('sell')">Sell</button>
      <router-link :to="`/edit/${coinId}`" class="btn btn-secondary btn-xs">Edit</router-link>
      <button class="btn btn-danger btn-xs" @click="$emit('delete')">Delete</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowLeft, Share2 } from 'lucide-vue-next'

withDefaults(defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
  sharing?: boolean
}>(), {
  sharing: false,
})

defineEmits<{
  share: []
  sell: []
  delete: []
}>()

const router = useRouter()
</script>

<style scoped>
.detail-header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0;
}

.detail-navigation {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  min-width: 0;
}

.detail-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 0.5rem;
  min-width: 0;
}

.back-action {
  justify-self: start;
  white-space: nowrap;
}

.share-action {
  white-space: nowrap;
}

@media (max-width: 768px) {
  .detail-header {
    grid-template-columns: 1fr;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .detail-actions {
    justify-content: flex-start;
    gap: 0.35rem;
  }
}
</style>
