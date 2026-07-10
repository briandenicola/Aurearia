<template>
  <component
    :is="as"
    :type="as === 'button' ? type : undefined"
    :disabled="disabled || loading"
    :class="[
      'inline-flex items-center justify-center gap-2',
      'border font-medium cursor-pointer',
      'transition-all duration-200',
      'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-gold',
      'disabled:opacity-50 disabled:cursor-not-allowed',
      variantClasses,
      sizeClasses,
    ]"
    v-bind="$attrs"
  >
    <BaseSpinner v-if="loading" size="sm" aria-hidden="true" />
    <slot />
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BaseSpinner from './BaseSpinner.vue'

const props = withDefaults(defineProps<{
  as?: string
  type?: 'button' | 'submit' | 'reset'
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'xs' | 'sm' | 'md'
  loading?: boolean
  disabled?: boolean
}>(), {
  as: 'button',
  type: 'button',
  variant: 'secondary',
  size: 'md',
  loading: false,
  disabled: false,
})

const variantClasses = computed(() => {
  switch (props.variant) {
    case 'primary':
      return [
        'bg-gradient-to-br from-gold to-bronze text-surface',
        'border-transparent',
        'hover:brightness-110 hover:shadow-[0_0_20px_var(--accent-gold-dim)]',
      ]
    case 'ghost':
      return [
        'bg-transparent text-text-secondary',
        'border-border-subtle',
        'hover:text-gold hover:border-border-accent hover:bg-gold-glow',
      ]
    case 'danger':
      return [
        'bg-[rgba(192,57,43,0.2)] text-[#e74c3c]',
        'border-[rgba(192,57,43,0.3)]',
        'hover:bg-[rgba(192,57,43,0.35)]',
      ]
    default: // secondary
      return [
        'bg-card text-text-primary',
        'border-border-subtle',
        'hover:border-border-accent hover:bg-card-hover',
      ]
  }
})

const sizeClasses = computed(() => {
  switch (props.size) {
    case 'xs': return 'px-[0.6rem] py-[0.25rem] text-sm rounded-sm'
    case 'sm': return 'px-[0.8rem] py-[0.4rem] text-chip rounded-sm'
    default:   return 'px-[1.2rem] py-[0.6rem] text-base rounded-sm'
  }
})
</script>
