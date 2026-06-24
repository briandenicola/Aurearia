<template>
  <select :value="modelValue" @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)" class="era-filter-select">
    <option value="">All Eras</option>
    <option v-for="era in eras" :key="era" :value="era">{{ formatEra(era) }}</option>
  </select>
</template>

<script setup lang="ts">
defineProps<{ modelValue: string; eras: string[] }>()
defineEmits<{ 'update:modelValue': [value: string] }>()

function formatEra(era: string): string {
  return era.charAt(0).toUpperCase() + era.slice(1)
}
</script>

<style scoped>
.era-filter-select {
  padding: 0.35rem 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 0.85rem;
  cursor: pointer;
}
</style>
