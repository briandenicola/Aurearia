<template>
  <div class="sticky top-[60px] z-50 mx-[-2rem] bg-surface px-8 pb-2">
    <div class="mb-4 flex flex-col gap-3 rounded-md border border-border-subtle bg-card p-4 shadow-card">
      <div class="flex items-center gap-3">
        <SearchBar
          class="!max-w-none flex-1"
          :model-value="search"
          @update:model-value="$emit('update:search', $event)"
        />
        <div class="min-w-48 shrink-0">
          <SortSelect :model-value="sortKey" @update:model-value="$emit('update:sortKey', $event)" />
        </div>
      </div>

      <div class="flex flex-nowrap items-center gap-3 max-md:flex-col max-md:items-stretch">
        <div class="flex shrink-0 flex-wrap gap-[0.35rem] max-md:w-full">
          <CategoryFilter :model-value="selectedCategory" @update:model-value="$emit('update:selectedCategory', $event)" />
        </div>

        <div class="h-6 w-px shrink-0 bg-border-subtle max-md:hidden"></div>

        <div class="flex min-w-0 flex-1 items-center gap-2 max-md:w-full">
          <EraFilter :model-value="selectedEra" :eras="eraOptions" @update:model-value="$emit('update:selectedEra', $event)" />
          <select v-if="userTags.length" :value="selectedTag" @change="$emit('update:selectedTag', ($event.target as HTMLSelectElement).value)" class="form-select h-[38px] min-w-0 flex-1 bg-card px-[0.6rem] py-[0.45rem] text-body transition-colors hover:border-border-accent">
            <option value="">All Sets</option>
            <option v-for="tag in userTags" :key="tag.filterValue" :value="tag.filterValue">{{ tag.name }}</option>
          </select>
        </div>

        <div class="h-6 w-px shrink-0 bg-border-subtle max-md:hidden"></div>

        <div class="flex shrink-0 items-center gap-2 max-md:w-full max-md:justify-between">
          <div class="inline-flex whitespace-nowrap rounded-sm border border-border-subtle bg-input p-[2px]">
            <button
              class="rounded-[6px] bg-transparent px-3 py-1.5 text-chip font-medium text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
              :class="gridSide === 'obverse' ? 'bg-gold text-surface hover:bg-gold hover:text-surface font-semibold' : ''"
              @click="$emit('update:gridSide', gridSide === 'obverse' ? null : 'obverse')"
            >
              Obverse
            </button>
            <button
              class="rounded-[6px] bg-transparent px-3 py-1.5 text-chip font-medium text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary"
              :class="gridSide === 'reverse' ? 'bg-gold text-surface hover:bg-gold hover:text-surface font-semibold' : ''"
              @click="$emit('update:gridSide', gridSide === 'reverse' ? null : 'reverse')"
            >
              Reverse
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CollectionSetOption, ImageType } from '@/types'
import CategoryFilter from '@/components/CategoryFilter.vue'
import EraFilter from '@/components/collection/EraFilter.vue'
import SearchBar from '@/components/SearchBar.vue'
import SortSelect from '@/components/SortSelect.vue'

defineProps<{
  search: string
  selectedCategory: string
  selectedEra: string
  selectedTag: string
  userTags: CollectionSetOption[]
  eraOptions: string[]
  sortKey: string
  gridSide: ImageType | null
}>()

defineEmits<{
  'update:search': [value: string]
  'update:selectedCategory': [value: string]
  'update:selectedEra': [value: string]
  'update:selectedTag': [value: string]
  'update:sortKey': [value: string]
  'update:gridSide': [value: ImageType | null]
}>()
</script>
