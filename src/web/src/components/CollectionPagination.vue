<template>
  <div v-if="total > perPage && viewMode === 'grid'" class="pagination">
    <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="$emit('prev')">← Previous</button>
    <span class="page-info">
      <span class="page-range">Showing {{ rangeStart }}-{{ rangeEnd }} of {{ total }} coins</span>
      <span class="page-number">Page {{ page }} of {{ Math.ceil(total / perPage) }}</span>
    </span>
    <button class="btn btn-secondary btn-sm" :disabled="page * perPage >= total" @click="$emit('next')">Next →</button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  page: number
  total: number
  perPage: number
  viewMode: string
}>()

defineEmits<{
  prev: []
  next: []
}>()

const rangeStart = computed(() => (props.page - 1) * props.perPage + 1)
const rangeEnd = computed(() => Math.min(props.page * props.perPage, props.total))
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
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.page-range {
  font-weight: 500;
  color: var(--text-primary);
}

.page-number {
  font-size: 0.75rem;
  color: var(--text-muted);
}

@media (min-width: 769px) {
  .page-info {
    flex-direction: row;
    gap: 0.5rem;
  }

  .page-number::before {
    content: '-';
    margin-right: 0.5rem;
  }
}
</style>
