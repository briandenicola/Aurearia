<template>
  <div class="detail-header">
    <button class="btn btn-ghost btn-xs back-action" @click="router.push('/')">
      <ArrowLeft :size="14" />
      Back to Gallery
    </button>
    <div class="detail-actions">
      <button v-if="!isWishlist && !isSold" class="btn btn-secondary btn-xs" @click="$emit('sell')">Sell</button>
      <router-link :to="`/edit/${coinId}`" class="btn btn-secondary btn-xs">Edit</router-link>
      <button class="btn btn-danger btn-xs" @click="$emit('delete')">Delete</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { ArrowLeft } from 'lucide-vue-next'

defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
}>()

defineEmits<{
  sell: []
  delete: []
}>()

const router = useRouter()
</script>

<style scoped>
.detail-header {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0;
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

@media (max-width: 768px) {
  .detail-header {
    grid-template-columns: auto 1fr;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .detail-actions {
    justify-content: flex-end;
    gap: 0.35rem;
  }
}
</style>
