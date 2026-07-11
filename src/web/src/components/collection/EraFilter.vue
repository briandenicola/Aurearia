<template>
  <select :value="modelValue" @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)" class="form-select h-[38px] min-w-0 flex-1 bg-card px-[0.6rem] py-[0.45rem] text-body transition-colors hover:border-border-accent">
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
