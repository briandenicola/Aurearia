<template>
  <select :value="modelValue" @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)" class="form-select h-[38px] min-w-0 flex-1 bg-card px-[0.6rem] py-[0.45rem] text-body transition-colors hover:border-border-accent">
    <option value="">All Categories</option>
    <option v-for="cat in categoryOptions" :key="cat" :value="cat">{{ cat }}</option>
  </select>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useCoinOptions } from '@/composables/useCoinOptions'

defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()

const { categoryOptions, loadOptions } = useCoinOptions()
onMounted(loadOptions)
</script>
