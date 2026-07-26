<template>
  <span
    class="inline-flex items-center px-[0.7rem] py-[0.2rem] rounded-full text-sm font-semibold uppercase tracking-[0.05em] border"
    :style="badgeStyle"
  >
    <slot />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { colorForLabel, colorForLabelBackground } from '@/utils/categoryColor'

const props = defineProps<{
  category?: 'roman' | 'greek' | 'byzantine' | 'modern' | 'other' | string
}>()

const badgeStyle = computed(() => {
  if (!props.category) {
    return { backgroundColor: 'rgba(136,136,136,0.2)', color: '#aaa', borderColor: 'rgba(136,136,136,0.3)' }
  }
  return {
    backgroundColor: colorForLabelBackground(props.category),
    color: colorForLabel(props.category),
    borderColor: colorForLabelBackground(props.category, 30),
  }
})
</script>
