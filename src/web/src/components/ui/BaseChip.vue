<template>
  <button
    type="button"
    :class="[
      'inline-flex items-center',
      'border cursor-pointer',
      'transition-all duration-200',
      'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold',
      size === 'sm'
        ? 'px-2 py-[0.15rem] rounded-full text-sm font-medium'
        : 'px-[0.85rem] py-[0.35rem] rounded-full text-chip font-medium',
      active
        ? 'bg-gold-dim border-gold text-gold'
        : 'bg-transparent border-border-subtle text-text-secondary hover:border-border-accent hover:text-text-primary',
    ]"
    :aria-pressed="active"
    v-bind="$attrs"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  active?: boolean
  size?: 'sm' | 'md'
}>(), {
  active: false,
  size: 'md',
})
</script>
