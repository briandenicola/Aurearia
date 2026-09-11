<template>
  <button
    class="tray-face-toggle"
    :class="{ 'is-reverse': activeFace === 'reverse' }"
    type="button"
    :aria-pressed="activeFace === 'reverse'"
    :aria-label="`Show ${activeFace === 'obverse' ? 'reverse' : 'obverse'} side for all coins`"
    :title="`Show ${activeFace === 'obverse' ? 'reverse' : 'obverse'} side`"
    @click="emit('toggle')"
  >
    <RotateCcw :size="16" />
  </button>
</template>

<script setup lang="ts">
import { RotateCcw } from 'lucide-vue-next'
import type { TrayCoinFace } from '@/utils/trayLayout'

defineProps<{ activeFace: TrayCoinFace }>()
const emit = defineEmits<{ toggle: [] }>()
</script>

<style scoped>
.tray-face-toggle {
  width: 2.75rem;
  height: 2.75rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  background: color-mix(in srgb, var(--bg-card) 88%, transparent);
  color: var(--text-secondary);
  box-shadow: var(--shadow-card);
  transition: all var(--transition-fast);
}

.tray-face-toggle:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
  transform: translateY(-1px);
}

.tray-face-toggle:focus-visible {
  outline: 2px solid var(--accent-gold);
  outline-offset: 2px;
}

.tray-face-toggle svg {
  transition: transform var(--transition-fast);
}

.tray-face-toggle.is-reverse {
  border-color: var(--accent-gold);
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.tray-face-toggle.is-reverse svg {
  transform: rotate(-180deg);
}
</style>
