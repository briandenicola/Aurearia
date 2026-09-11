<template>
  <div class="museum-tray tray-surface" :class="`felt-${feltTheme}`">
    <slot />
  </div>
</template>

<script setup lang="ts">
import type { FeltColor } from '@/composables/useTrayPreference'

defineProps<{ feltTheme: FeltColor }>()
</script>

<style>
.tray-surface {
  position: relative;
  border-radius: var(--radius-md);
}

.tray-surface.felt-red {
  background:
    linear-gradient(135deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%, transparent 75%, rgba(0, 0, 0, 0.05) 75%),
    linear-gradient(45deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%, transparent 75%, rgba(0, 0, 0, 0.05) 75%),
    linear-gradient(to bottom, var(--felt-red-base), var(--felt-red-dark));
  background-size: 4px 4px, 4px 4px, 100% 100%;
  background-position: 0 0, 2px 2px, 0 0;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.3), var(--shadow-card);
}

.tray-surface.felt-green {
  background:
    linear-gradient(135deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%, transparent 75%, rgba(0, 0, 0, 0.05) 75%),
    linear-gradient(45deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%, transparent 75%, rgba(0, 0, 0, 0.05) 75%),
    linear-gradient(to bottom, var(--felt-green-base), var(--felt-green-dark));
  background-size: 4px 4px, 4px 4px, 100% 100%;
  background-position: 0 0, 2px 2px, 0 0;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.3), var(--shadow-card);
}

.tray-surface.felt-navy {
  background:
    linear-gradient(135deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%, transparent 75%, rgba(0, 0, 0, 0.05) 75%),
    linear-gradient(45deg, rgba(0, 0, 0, 0.05) 25%, transparent 25%, transparent 75%, rgba(0, 0, 0, 0.05) 75%),
    linear-gradient(to bottom, var(--felt-navy-base), var(--felt-navy-dark));
  background-size: 4px 4px, 4px 4px, 100% 100%;
  background-position: 0 0, 2px 2px, 0 0;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.3), var(--shadow-card);
}
</style>
