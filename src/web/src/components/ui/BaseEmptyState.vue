<template>
  <div
    class="flex flex-col items-center gap-4 py-16 px-8 text-center text-text-muted"
    role="status"
  >
    <div v-if="$slots.icon" class="opacity-60">
      <slot name="icon" />
    </div>
    <component
      :is="iconComponent"
      v-else-if="iconComponent"
      :size="48"
      class="opacity-60"
    />

    <h3 v-if="title" class="text-text-secondary text-lg font-medium m-0">
      {{ title }}
    </h3>

    <p v-if="message" class="text-body max-w-md leading-relaxed m-0">
      {{ message }}
    </p>

    <slot />
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

withDefaults(defineProps<{
  title?: string
  message?: string
  iconComponent?: Component
}>(), {
  title: undefined,
  message: undefined,
  iconComponent: undefined,
})
</script>
