<template>
  <Teleport v-if="!isPwa" defer to="#desktop-page-title">
    <span class="block truncate" :title="title">{{ title }}</span>
  </Teleport>
  <Teleport v-if="!isPwa && $slots.actions" defer to="#desktop-page-actions">
    <slot name="actions" />
  </Teleport>
</template>

<script setup lang="ts">
import { usePwa } from '@/composables/usePwa'

defineProps<{
  title: string
}>()

const { isPwa } = usePwa()
</script>
