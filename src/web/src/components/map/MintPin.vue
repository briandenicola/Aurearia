<template>
  <g
    class="mint-pin"
    :class="{ active }"
    :transform="`translate(${x} ${y})`"
    role="button"
    tabindex="0"
    :aria-label="`${group.mint.displayName}: ${group.count} ${group.count === 1 ? 'coin' : 'coins'}`"
    @click="$emit('select', group)"
    @keydown.enter.prevent="$emit('select', group)"
    @keydown.space.prevent="$emit('select', group)"
  >
    <circle class="pin-halo" r="20" />
    <circle class="pin-marker" r="9" />
    <text class="pin-count" x="13" y="-12" text-anchor="middle" dominant-baseline="central">
      {{ group.count }}
    </text>
  </g>
</template>

<script setup lang="ts">
import type { MintGroup } from '@/utils/mintMap'

defineProps<{
  group: MintGroup
  x: number
  y: number
  active: boolean
}>()

defineEmits<{
  select: [group: MintGroup]
}>()
</script>

<style scoped>
.mint-pin {
  cursor: pointer;
  outline: none;
}

.pin-halo {
  fill: var(--accent-gold-glow);
  stroke: var(--border-accent);
  stroke-width: 1;
  opacity: 0.75;
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}

.pin-marker {
  fill: var(--accent-gold);
  stroke: var(--bg-card);
  stroke-width: 3;
  transition: fill var(--transition-fast), transform var(--transition-fast);
}

.pin-count {
  fill: var(--bg-primary);
  stroke: var(--accent-gold);
  stroke-width: 5;
  paint-order: stroke fill;
  font-size: 0.75rem;
  font-weight: 700;
  pointer-events: none;
}

.mint-pin:hover .pin-halo,
.mint-pin:focus-visible .pin-halo,
.mint-pin.active .pin-halo {
  opacity: 1;
  transform: scale(1.12);
}

.mint-pin:focus-visible .pin-marker,
.mint-pin.active .pin-marker {
  fill: var(--text-heading);
}
</style>
