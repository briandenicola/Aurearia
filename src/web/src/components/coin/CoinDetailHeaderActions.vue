<template>
  <div class="detail-header">
    <button class="btn btn-secondary btn-sm" @click="router.back()">← Back</button>
    <div class="detail-actions">
      <button v-if="isWishlist" class="btn btn-primary btn-sm" @click="$emit('purchase')">Mark as Purchased</button>
      <button v-if="!isWishlist && !isSold" class="btn btn-secondary btn-sm" @click="$emit('sell')">Sell</button>
      <router-link :to="`/edit/${coinId}`" class="btn btn-secondary btn-sm">Edit</router-link>
      <button class="btn btn-danger btn-sm" @click="$emit('delete')">Delete</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

defineProps<{
  isWishlist: boolean
  isSold: boolean
  coinId: number
}>()

defineEmits<{
  purchase: []
  sell: []
  delete: []
}>()

const router = useRouter()
</script>

<style scoped>
.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.detail-actions {
  display: flex;
  gap: 0.5rem;
}
</style>
