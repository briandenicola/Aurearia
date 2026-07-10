<template>
  <g
    class="mint-pin group cursor-pointer outline-none"
    :transform="`translate(${x} ${y})`"
    role="button"
    tabindex="0"
    :aria-label="`${group.mint.displayName}: ${group.count} ${group.count === 1 ? 'coin' : 'coins'}`"
    @click="$emit('select', group)"
    @keydown.enter.prevent="$emit('select', group)"
    @keydown.space.prevent="$emit('select', group)"
  >
    <circle
      r="20"
      class="pointer-events-none origin-center fill-gold-glow stroke-border-accent opacity-75 transition-all duration-150"
      :class="active ? 'scale-110 opacity-100' : 'group-hover:scale-110 group-hover:opacity-100 group-focus-visible:scale-110 group-focus-visible:opacity-100'"
    />
    <circle
      r="9"
      class="origin-center fill-gold stroke-card [stroke-width:3] transition-all duration-150"
      :class="active ? 'fill-heading' : 'group-focus-visible:fill-heading'"
    />
    <text
      x="13"
      y="-12"
      text-anchor="middle"
      dominant-baseline="central"
      class="pointer-events-none fill-surface text-sm font-bold [paint-order:stroke] stroke-gold [stroke-width:5]"
    >
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
