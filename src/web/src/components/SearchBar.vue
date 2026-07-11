<template>
  <div class="relative flex w-full max-w-[400px] items-center">
    <Search class="pointer-events-none absolute left-3 text-text-muted" :size="16" />
    <input
      type="text"
      class="form-input rounded-full py-2 pr-10 pl-10 text-body placeholder:text-text-muted"
      :value="modelValue"
      @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
      placeholder="Search coins by name, ruler, inscription..."
    />
    <button v-if="modelValue" class="absolute right-2 p-1 text-text-muted transition-colors hover:text-text-primary" @click="$emit('update:modelValue', '')"><X :size="14" /></button>
  </div>
</template>

<script setup lang="ts">
import { Search, X } from 'lucide-vue-next'
defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
