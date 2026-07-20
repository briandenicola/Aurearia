<template>
  <div class="relative" ref="wrapperRef">
    <div class="mb-2 flex flex-wrap gap-1">
      <button
        v-for="tab in roleTabs"
        :key="tab.value"
        type="button"
        class="rounded-full border px-2 py-0.5 text-xs transition-colors"
        :class="activeRole === tab.value
          ? 'border-gold bg-gold-dim text-heading'
          : 'border-border-subtle text-text-muted hover:border-gold'"
        @click="selectRole(tab.value)"
      >
        {{ tab.label }}
      </button>
    </div>
    <div class="flex gap-2">
      <input
        v-model="query"
        class="form-input"
        placeholder="Search Roman imperial figures..."
        autocomplete="off"
        @input="onInput"
        @focus="onFocus"
      />
      <button
        v-if="modelValue"
        type="button"
        class="btn btn-sm"
        @click="clearSelection"
      >
        Clear
      </button>
    </div>
    <ul
      v-if="showDropdown && results.length"
      class="absolute inset-x-0 top-full z-50 mt-1 max-h-[240px] overflow-y-auto rounded-sm border border-border-accent bg-card shadow-card"
    >
      <li
        v-for="(figure, i) in results"
        :key="figure.id"
        :class="[
          'cursor-pointer px-3 py-2 text-base text-text-primary transition-colors',
          i === highlightIndex ? 'bg-gold-dim text-heading' : 'hover:bg-gold-dim hover:text-heading',
        ]"
        @mousedown.prevent="select(figure)"
      >
        {{ figure.name }}
        <span class="text-xs text-text-muted">({{ figure.role }})</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { searchRomanImperialFigures, getRomanImperialFigure } from '@/api/client'
import type { RomanImperialFigure, ImperialFigureRole } from '@/types'

const props = defineProps<{
  modelValue: number | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const roleTabs: { value: ImperialFigureRole | ''; label: string }[] = [
  { value: '', label: 'All' },
  { value: 'emperor', label: 'Emperor' },
  { value: 'empress', label: 'Empress' },
  { value: 'caesar', label: 'Caesar' },
  { value: 'usurper', label: 'Usurper' },
  { value: 'other', label: 'Other' },
]

const query = ref('')
const results = ref<RomanImperialFigure[]>([])
const showDropdown = ref(false)
const highlightIndex = ref(-1)
const activeRole = ref<ImperialFigureRole | ''>('')
const wrapperRef = ref<HTMLElement | null>(null)
let debounceTimer: ReturnType<typeof setTimeout>

async function fetchResults(q: string) {
  try {
    const res = await searchRomanImperialFigures({
      q: q || undefined,
      role: activeRole.value || undefined,
      limit: 50,
    })
    results.value = res.data?.figures ?? []
  } catch {
    results.value = []
  }
}

function onInput() {
  // Typing invalidates any previously confirmed selection.
  if (props.modelValue) {
    emit('update:modelValue', null)
  }
  highlightIndex.value = -1
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => fetchResults(query.value), 200)
  showDropdown.value = true
}

function onFocus() {
  clearTimeout(debounceTimer)
  fetchResults(query.value)
  showDropdown.value = true
}

function selectRole(role: ImperialFigureRole | '') {
  activeRole.value = role
  fetchResults(query.value)
  showDropdown.value = true
}

function select(figure: RomanImperialFigure) {
  query.value = figure.name
  emit('update:modelValue', figure.id)
  showDropdown.value = false
  results.value = []
}

function clearSelection() {
  query.value = ''
  emit('update:modelValue', null)
  results.value = []
}

function onClickOutside(e: MouseEvent) {
  if (wrapperRef.value && !wrapperRef.value.contains(e.target as Node)) {
    showDropdown.value = false
  }
}

onMounted(async () => {
  document.addEventListener('click', onClickOutside)
  if (props.modelValue) {
    try {
      const res = await getRomanImperialFigure(props.modelValue)
      query.value = res.data.name
    } catch {
      // Figure lookup failed (e.g. deleted from the seed) — leave blank
      // rather than showing a stale/incorrect name.
    }
  }
})
onUnmounted(() => {
  document.removeEventListener('click', onClickOutside)
  clearTimeout(debounceTimer)
})
</script>
