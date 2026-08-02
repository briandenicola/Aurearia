<template>
  <button
    type="button"
    class="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-transparent text-text-secondary transition-all hover:border-border-accent hover:bg-gold-glow hover:text-gold disabled:cursor-not-allowed disabled:opacity-55"
    :class="[{ 'border-border-accent bg-gold-glow text-gold': active }, extraClass]"
    :title="title"
    :aria-label="ariaLabel || title"
    :disabled="disabled"
    @click="$emit('click')"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  ariaLabel?: string
  disabled?: boolean
  active?: boolean
  extraClass?: string
}>()

defineEmits<{
  click: []
}>()
</script>
