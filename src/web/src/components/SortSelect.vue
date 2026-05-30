<template>
  <div class="sort-select">
    <ArrowUpDown class="sort-icon" :size="14" />
    <select
      :value="modelValue"
      @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
    >
      <option value="updated_at_desc">Last Updated</option>
      <option value="created_at_desc">Newest First</option>
      <option value="created_at_asc">Oldest First</option>
      <option value="current_value_desc">Price: High → Low</option>
      <option value="current_value_asc">Price: Low → High</option>
      <option value="random_desc">Random</option>
      <option value="needs_attention">Needs Attention</option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { ArrowUpDown } from 'lucide-vue-next'
defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<style scoped>
.sort-select {
  position: relative;
  display: flex;
  align-items: center;
}

.sort-icon {
  position: absolute;
  left: 0.6rem;
  color: var(--text-muted);
  pointer-events: none;
}

select {
  appearance: none;
  padding: 0.5rem 2rem 0.5rem 2rem;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  color: var(--text-primary);
  font-size: 0.8rem;
  cursor: pointer;
  transition: border-color var(--transition-fast);
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2'%3E%3Cpath d='m6 9 6 6 6-6'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.6rem center;
}

select:focus {
  outline: none;
  border-color: var(--accent-gold);
  box-shadow: 0 0 0 2px var(--accent-gold-glow);
}
</style>
