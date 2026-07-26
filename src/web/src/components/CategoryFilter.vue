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
      v-for="cat in categoryOptions"
      :key="cat"
      class="chip"
      :style="modelValue === cat
        ? {
            borderColor: colorForLabel(cat),
            backgroundColor: colorForLabelBackground(cat),
            color: colorForLabel(cat),
          }
        : undefined"
      @click="$emit('update:modelValue', cat)"
    >
      {{ cat }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useCoinOptions } from '@/composables/useCoinOptions'
import { colorForLabel, colorForLabelBackground } from '@/utils/categoryColor'

defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()

const { categoryOptions, loadOptions } = useCoinOptions()
onMounted(loadOptions)
</script>
