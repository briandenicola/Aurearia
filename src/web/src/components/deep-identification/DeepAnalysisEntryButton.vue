<template>
  <BaseButton
    type="button"
    variant="ghost"
    size="sm"
    class="min-h-[44px]"
    :disabled="disabled"
    aria-haspopup="dialog"
    :aria-label="label"
    @click="$emit('click')"
  >
    <Microscope :size="16" aria-hidden="true" />
    {{ label }}
  </BaseButton>
</template>

<script setup lang="ts">
import { Microscope } from 'lucide-vue-next'
import BaseButton from '@/components/ui/BaseButton.vue'

withDefaults(defineProps<{
  label?: string
  disabled?: boolean
}>(), {
  label: 'Deep Analysis',
  disabled: false,
})

defineEmits<{ click: [] }>()
</script>
