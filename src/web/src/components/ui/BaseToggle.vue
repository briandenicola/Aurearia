<template>
  <label
    :class="[
      'relative inline-block shrink-0 rounded-full',
      size === 'sm' ? 'h-[22px] w-[42px]' : 'h-7 w-[50px]',
      $attrs.class,
    ]"
    :style="$attrs.style as StyleValue"
  >
    <input
      type="checkbox"
      class="peer sr-only"
      :checked="modelValue"
      :disabled="disabled"
      :aria-label="label"
      v-bind="inputAttrs"
      @change="onChange"
    />
    <span
      :class="[
        'absolute inset-0 rounded-full border border-border-subtle bg-input transition-colors',
        'peer-checked:border-gold peer-checked:bg-gold-dim',
        'peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-gold',
        'peer-disabled:cursor-not-allowed peer-disabled:opacity-60',
        disabled ? '' : 'cursor-pointer',
        'after:absolute after:left-[2px] after:rounded-full after:bg-text-secondary',
        `after:transition-transform after:content-['']`,
        'peer-checked:after:bg-gold',
        // 2px inset on all four edges of the span's padding box in both sizes.
        size === 'sm'
          ? 'after:bottom-[2px] after:h-4 after:w-4 peer-checked:after:translate-x-5'
          : 'after:bottom-[2px] after:h-[22px] after:w-[22px] peer-checked:after:translate-x-[22px]',
      ]"
    ></span>
  </label>
</template>

<script setup lang="ts">
import { computed, useAttrs, type StyleValue } from 'vue'

defineOptions({ inheritAttrs: false })

withDefaults(defineProps<{
  label: string
  modelValue?: boolean
  size?: 'sm' | 'md'
  disabled?: boolean
}>(), {
  modelValue: false,
  size: 'md',
  disabled: false,
})

const attrs = useAttrs()

// class and style belong on the wrapper; everything else on the hidden input.
const inputAttrs = computed(() => {
  const { class: _class, style: _style, ...rest } = attrs
  return rest
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

function onChange(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).checked)
}
</script>
