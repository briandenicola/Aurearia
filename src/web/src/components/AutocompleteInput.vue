<template>
  <div class="relative" ref="wrapperRef">
    <input
      :value="modelValue"
      @input="onInput"
      @focus="onFocus"
      @keydown="onKeydown"
      class="form-input"
      :placeholder="placeholder"
      :required="required"
      autocomplete="off"
    />
    <ul v-if="showDropdown && suggestions.length" class="absolute inset-x-0 top-full z-50 mt-1 max-h-[200px] overflow-y-auto rounded-sm border border-border-accent bg-card shadow-card">
      <li
        v-for="(item, i) in suggestions"
        :key="item"
        :class="[
          'cursor-pointer px-3 py-2 text-base text-text-primary transition-colors',
          i === highlightIndex ? 'bg-gold-dim text-heading' : 'hover:bg-gold-dim hover:text-heading',
        ]"
        @mousedown.prevent="select(item)"
      >
        {{ item }}
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { getSuggestions } from '@/api/client'

const props = defineProps<{
  modelValue: string
  field: string
  placeholder?: string
  required?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const suggestions = ref<string[]>([])
const showDropdown = ref(false)
const highlightIndex = ref(-1)
const wrapperRef = ref<HTMLElement | null>(null)
let debounceTimer: ReturnType<typeof setTimeout>

async function fetchSuggestions(q: string) {
  if (!q || q.length < 1) {
    suggestions.value = []
    return
  }
  try {
    const res = await getSuggestions(props.field, q)
    suggestions.value = res.data || []
  } catch {
    suggestions.value = []
  }
}

function onInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  emit('update:modelValue', val)
  highlightIndex.value = -1
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => fetchSuggestions(val), 200)
  showDropdown.value = true
}

function onFocus() {
  if (props.modelValue) {
    fetchSuggestions(props.modelValue)
  }
  showDropdown.value = true
}

function onKeydown(e: KeyboardEvent) {
  if (!showDropdown.value || !suggestions.value.length) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    highlightIndex.value = Math.min(highlightIndex.value + 1, suggestions.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    highlightIndex.value = Math.max(highlightIndex.value - 1, 0)
  } else if (e.key === 'Enter' && highlightIndex.value >= 0) {
    e.preventDefault()
    const val = suggestions.value[highlightIndex.value]
    if (val) select(val)
  } else if (e.key === 'Escape') {
    showDropdown.value = false
  }
}

function select(val: string) {
  emit('update:modelValue', val)
  showDropdown.value = false
  suggestions.value = []
}

function onClickOutside(e: MouseEvent) {
  if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
    showDropdown.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside))
onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
  clearTimeout(debounceTimer)
})
</script>
