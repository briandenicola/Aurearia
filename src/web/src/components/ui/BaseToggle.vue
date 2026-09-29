<template>
  <label
    :class="[
      'relative inline-block shrink-0 rounded-full',
      'focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-gold',
      size === 'sm' ? 'h-[22px] w-[42px]' : 'h-7 w-[50px]',
    ]"
  >
    <input
      type="checkbox"
      class="peer sr-only"
      :checked="modelValue"
      :disabled="disabled"
      :aria-label="label"
      v-bind="$attrs"
      @change="onChange"
    />
    <span
      :class="[
        'absolute inset-0 rounded-full border border-border-subtle bg-input transition-colors',
        'peer-checked:border-gold peer-checked:bg-gold-dim',
        'peer-disabled:cursor-not-allowed peer-disabled:opacity-60',
        disabled ? '' : 'cursor-pointer',
        'after:absolute after:left-[2px] after:rounded-full after:bg-text-secondary',
        `after:transition-transform after:content-['']`,
        'peer-checked:after:bg-gold',
        size === 'sm'
          ? 'after:bottom-[3px] after:h-4 after:w-4 peer-checked:after:translate-x-5'
          : 'after:bottom-[2px] after:h-[22px] after:w-[22px] peer-checked:after:translate-x-[22px]',
      ]"
    ></span>
  </label>
</template>

<script setup lang="ts">
defineOptions({ inheritAttrs: false })

withDefaults(defineProps<{
  modelValue?: boolean
  size?: 'sm' | 'md'
  disabled?: boolean
  label?: string
}>(), {
  modelValue: false,
  size: 'md',
  disabled: false,
  label: undefined,
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

function onChange(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).checked)
}
</script>
