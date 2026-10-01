<template>
  <component
    :is="to ? RouterLink : 'button'"
    :to="to"
    :type="to ? undefined : 'button'"
    class="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-transparent text-text-secondary transition-all hover:border-border-accent hover:bg-gold-glow hover:text-gold disabled:cursor-not-allowed disabled:opacity-55"
    :class="[{ 'border-border-accent bg-gold-glow text-gold': active }, extraClass]"
    :title="title"
    :aria-label="ariaLabel || title"
    :disabled="to ? undefined : disabled"
    :aria-disabled="to && disabled ? 'true' : undefined"
    @click="$emit('click')"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'

defineProps<{
  title: string
  ariaLabel?: string
  disabled?: boolean
  active?: boolean
  extraClass?: string
  to?: RouteLocationRaw
}>()

defineEmits<{
  click: []
}>()
</script>
