<template>
  <span
    :class="[
      'inline-flex items-center px-[0.7rem] py-[0.2rem] rounded-full',
      'text-sm font-semibold uppercase tracking-[0.05em]',
      categoryClass ?? 'bg-[rgba(136,136,136,0.2)] text-[#aaa] border border-[rgba(136,136,136,0.3)]',
    ]"
  >
    <slot />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  category?: 'roman' | 'greek' | 'byzantine' | 'modern' | 'other' | string
}>()

const categoryClass = computed(() => {
  switch (props.category?.toLowerCase()) {
    case 'roman':
      return 'bg-[rgba(155,89,182,0.2)] text-[#b57edc] border border-[rgba(155,89,182,0.3)]'
    case 'greek':
      return 'bg-[rgba(107,142,35,0.2)] text-[#9ab85a] border border-[rgba(107,142,35,0.3)]'
    case 'byzantine':
      return 'bg-[rgba(192,57,43,0.2)] text-[#e67e73] border border-[rgba(192,57,43,0.3)]'
    case 'modern':
      return 'bg-[rgba(70,130,180,0.2)] text-[#7ab3d4] border border-[rgba(70,130,180,0.3)]'
    case 'other':
    default:
      return null
  }
})
</script>
