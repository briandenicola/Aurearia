<template>
  <div v-if="total > perPage && viewMode === 'grid'" class="pagination">
    <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="$emit('prev')">← Previous</button>
    <span class="page-info">Page {{ page }} of {{ Math.ceil(total / perPage) }}</span>
    <button class="btn btn-secondary btn-sm" :disabled="page * perPage >= total" @click="$emit('next')">Next →</button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  page: number
  total: number
  perPage: number
  viewMode: string
}>()

defineEmits<{
  prev: []
  next: []
}>()
</script>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-top: 2rem;
  padding-top: 1.5rem;
  border-top: 1px solid var(--border-subtle);
}

.page-info {
  color: var(--text-secondary);
  font-size: 0.85rem;
}
</style>
