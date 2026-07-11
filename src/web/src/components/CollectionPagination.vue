<template>
  <div v-if="total > perPage && viewMode === 'grid'" class="mt-8 flex flex-col items-center justify-center gap-4 border-t border-border-subtle pt-6 md:flex-row">
    <button class="btn btn-secondary btn-sm" :disabled="page <= 1" @click="$emit('prev')">← Previous</button>
    <span class="flex flex-col items-center gap-1 text-body text-text-secondary md:flex-row md:gap-2">
      <span class="font-medium text-text-primary">Showing {{ rangeStart }}-{{ rangeEnd }} of {{ total }} coins</span>
      <span class="hidden text-text-muted md:inline">-</span>
      <span class="text-sm text-text-muted">Page {{ page }} of {{ Math.ceil(total / perPage) }}</span>
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
