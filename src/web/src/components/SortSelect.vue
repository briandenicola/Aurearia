<template>
  <div class="relative flex items-center">
    <ArrowUpDown class="pointer-events-none absolute left-2.5 text-text-muted" :size="14" />
    <select
      class="form-select rounded-full py-2 pl-8 text-chip"
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
