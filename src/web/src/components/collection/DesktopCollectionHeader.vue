<template>
  <div class="desktop-sticky-header">
    <div class="command-bar">
      <div class="command-row search-row">
        <SearchBar :model-value="search" @update:model-value="$emit('update:search', $event)" />
        <div class="sort-zone">
          <SortSelect :model-value="sortKey" @update:model-value="$emit('update:sortKey', $event)" />
        </div>
      </div>

      <div class="command-row action-row">
        <div class="filter-zone">
          <CategoryFilter :model-value="selectedCategory" @update:model-value="$emit('update:selectedCategory', $event)" />
        </div>

        <div class="toolbar-divider"></div>

        <div class="dropdown-zone">
          <EraFilter :model-value="selectedEra" :eras="eraOptions" @update:model-value="$emit('update:selectedEra', $event)" />
          <select v-if="userTags.length" :value="selectedTag" @change="$emit('update:selectedTag', ($event.target as HTMLSelectElement).value)" class="tag-filter-select">
            <option value="">All Sets</option>
            <option v-for="tag in userTags" :key="tag.filterValue" :value="tag.filterValue">{{ tag.name }}</option>
          </select>
        </div>

        <div class="toolbar-divider action-divider"></div>

        <div class="action-zone">
          <div class="face-toggle">
            <button class="face-btn" :class="{ active: gridSide === 'obverse' }" @click="$emit('update:gridSide', gridSide === 'obverse' ? null : 'obverse')">
              Obverse
            </button>
            <button class="face-btn" :class="{ active: gridSide === 'reverse' }" @click="$emit('update:gridSide', gridSide === 'reverse' ? null : 'reverse')">
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

<style scoped>
.desktop-sticky-header {
  position: sticky;
  top: 60px;
  z-index: 50;
  background: var(--bg-primary);
  padding-bottom: 0.5rem;
  margin: 0 -2rem;
  padding-left: 2rem;
  padding-right: 2rem;
}

.command-bar {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-card);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.search-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

/*
 * :deep() audit — child-component slot piercing
 * Targets .search-bar, .search-input, and .era-filter-select rendered inside
 * child components (CoinSearch, EraFilterSelect). The header owns the layout
 * constraints (flex sizing, height), while the child owns its internal CSS
 * classes. These cannot be done inside the child because the sizes are driven
 * by the header's grid context, not the child's own concerns.
 */
.search-row :deep(.search-bar) {
  flex: 1;
  max-width: none;
}

.search-row :deep(.search-input) {
  padding: 0.5rem 2.5rem;
  font-size: 0.9rem;
  height: 38px;
}

.sort-zone {
  flex-shrink: 0;
  min-width: 12rem;
}

.action-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: nowrap;
}

.filter-zone {
  display: flex;
  gap: 0.35rem;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.toolbar-divider {
  width: 1px;
  height: 24px;
  background: var(--border-subtle);
  flex-shrink: 0;
}

.dropdown-zone {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex: 1 1 auto;
  min-width: 0;
}

.dropdown-zone :deep(.era-filter-select) {
  height: 38px;
  padding: 0.45rem 0.6rem;
  transition: border-color var(--transition-fast);
  flex: 1 1 10rem;
  min-width: 0;
}

.dropdown-zone :deep(.era-filter-select:hover) {
  border-color: var(--border-accent);
}

.tag-filter-select {
  padding: 0.45rem 0.6rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 0.85rem;
  cursor: pointer;
  height: 38px;
  transition: border-color var(--transition-fast);
  flex: 1 1 10rem;
  min-width: 0;
}

.tag-filter-select:hover {
  border-color: var(--border-accent);
}

.action-zone {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.face-toggle {
  display: inline-flex;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  padding: 2px;
  gap: 2px;
  white-space: nowrap;
}

.face-btn {
  padding: 0.4rem 0.8rem;
  border: none;
  border-radius: calc(var(--radius-sm) - 2px);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.face-btn:hover {
  color: var(--text-primary);
  background: var(--bg-card-hover);
}

.face-btn.active {
  background: var(--accent-gold);
  color: var(--bg-primary);
  font-weight: 600;
}

@media (max-width: 768px) {
  .command-bar {
    padding: 0.75rem;
  }

  .action-row {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-divider {
    display: none;
  }

  .filter-zone,
  .dropdown-zone,
  .action-zone {
    width: 100%;
    margin-left: 0;
  }

  .action-zone {
    justify-content: space-between;
  }
}
</style>
