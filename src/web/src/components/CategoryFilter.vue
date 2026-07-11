<template>
  <div class="flex flex-wrap gap-[0.4rem]">
    <button
      class="chip"
      :class="{ active: !modelValue }"
      @click="$emit('update:modelValue', '')"
    >
      All
    </button>
    <button
      v-for="cat in CATEGORIES"
      :key="cat"
      class="chip"
      :class="modelValue === cat
        ? cat === 'Roman'
          ? 'border-roman bg-roman/20 text-roman'
          : cat === 'Greek'
            ? 'border-greek bg-greek/20 text-greek'
            : cat === 'Byzantine'
              ? 'border-byzantine bg-byzantine/20 text-byzantine'
              : cat === 'Modern'
                ? 'border-modern-cat bg-modern-cat/20 text-modern-cat'
                : 'border-other bg-other/20 text-other'
        : ''"
      @click="$emit('update:modelValue', cat)"
    >
      {{ cat }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { CATEGORIES } from '@/types'

defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
