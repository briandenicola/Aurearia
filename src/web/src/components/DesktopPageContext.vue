<template>
  <Teleport v-if="!isPwa" defer to="#desktop-page-title">
    <span class="block truncate" :title="title">{{ title }}</span>
  </Teleport>
  <Teleport v-if="!isPwa && $slots.actions" defer to="#desktop-page-actions">
    <slot name="actions" />
  </Teleport>
  <div
    v-if="$slots.default"
    class="desktop-page-context-local"
    :class="{ 'is-pwa': isPwa }"
  >
    <slot />
  </div>
  <div
    v-if="$slots.desktop"
    class="desktop-page-context-desktop"
    :class="{ 'is-pwa': isPwa }"
  >
    <slot name="desktop" />
  </div>
</template>

<script setup lang="ts">
import { usePwa } from '@/composables/usePwa'

defineProps<{
  title: string
}>()

const { isPwa } = usePwa()
</script>

<style scoped>
.desktop-page-context-desktop {
  display: none;
}

@media (min-width: 769px) {
  .desktop-page-context-local:not(.is-pwa) {
    display: none;
  }

  .desktop-page-context-desktop:not(.is-pwa) {
    display: block;
  }
}
</style>
