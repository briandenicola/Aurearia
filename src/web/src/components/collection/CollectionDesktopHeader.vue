<template>
  <div class="desktop-sticky-header">
    <div class="page-header collection-header">
      <div class="header-spacer"></div>
      <SearchBar :model-value="search" @update:model-value="$emit('update:search', $event)" />
      <div class="header-sort">
        <SortSelect :model-value="sortKey" @update:model-value="$emit('update:sortKey', $event)" />
      </div>
    </div>

    <div class="collection-toolbar">
      <div class="toolbar-filters">
        <CategoryFilter :model-value="selectedCategory" @update:model-value="$emit('update:selectedCategory', $event)" />
        <select v-if="userTags.length" :value="selectedTag" @change="$emit('update:selectedTag', ($event.target as HTMLSelectElement).value)" class="tag-filter-select">
          <option value="">All Tags</option>
          <option v-for="tag in userTags" :key="tag.id" :value="String(tag.id)">{{ tag.name }}</option>
        </select>
      </div>
      <div class="toolbar-right">
        <button class="btn" :class="selectMode ? 'btn-primary' : 'btn-secondary'" @click="$emit('toggle-select-mode')">
          <CheckSquare :size="16" /> {{ selectMode ? 'Cancel' : 'Select' }}
        </button>
        <div class="side-toggle">
          <button class="btn btn-primary toggle-btn" :class="{ active: gridSide === null }" @click="$emit('update:gridSide', null)">
            Primary
          </button>
          <button class="btn btn-primary toggle-btn" :class="{ active: gridSide === 'obverse' }" @click="$emit('update:gridSide', 'obverse')">
            Obverse
          </button>
          <button class="btn btn-primary toggle-btn" :class="{ active: gridSide === 'reverse' }" @click="$emit('update:gridSide', 'reverse')">
            Reverse
          </button>
        </div>
        <router-link to="/add" class="btn btn-primary"><CirclePlus :size="16" /> Add Coin</router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ImageType, Tag } from '@/types'
import SearchBar from '@/components/SearchBar.vue'
import CategoryFilter from '@/components/CategoryFilter.vue'
import SortSelect from '@/components/SortSelect.vue'
import { CheckSquare, CirclePlus } from 'lucide-vue-next'

defineProps<{
  search: string
  selectMode: boolean
  selectedCategory: string
  selectedTag: string
  sortKey: string
  gridSide: ImageType | null
  userTags: Tag[]
}>()

defineEmits<{
  'update:search': [value: string]
  'update:selectedCategory': [value: string]
  'update:selectedTag': [value: string]
  'update:sortKey': [value: string]
  'update:gridSide': [value: ImageType | null]
  'toggle-select-mode': []
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

.collection-header {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header-spacer {
  flex: 1;
}

.collection-header :deep(.search-bar) {
  flex: 0 1 600px;
}

.collection-header :deep(.search-input) {
  padding: 0.75rem 2.5rem;
  font-size: 0.95rem;
}

.header-sort {
  flex: 1;
  display: flex;
  justify-content: flex-end;
}

.collection-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
  margin-bottom: 1rem;
}

.toolbar-filters {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.tag-filter-select {
  padding: 0.35rem 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-primary);
  font-size: 0.85rem;
  cursor: pointer;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.side-toggle {
  display: flex;
  gap: 0;
}

.side-toggle .toggle-btn {
  border-radius: 0;
  border-right: 1px solid rgba(255, 255, 255, 0.15);
}

.side-toggle .toggle-btn:first-child {
  border-radius: var(--radius-sm) 0 0 var(--radius-sm);
}

.side-toggle .toggle-btn:last-child {
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  border-right: none;
}

.toggle-btn {
  opacity: 0.6;
}

.toggle-btn.active {
  opacity: 1;
  background: var(--accent-gold);
  color: #1a1a2e;
}

.toggle-btn:hover:not(.active) {
  opacity: 0.8;
}

@media (max-width: 768px) {
  .collection-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-spacer {
    display: none;
  }

  .header-sort {
    justify-content: flex-start;
  }

  .collection-header :deep(.search-bar) {
    max-width: 100%;
  }

  .collection-header :deep(.search-input) {
    padding: 0.6rem 2.5rem;
    font-size: 0.85rem;
  }

  .header-filters {
    justify-content: flex-start;
  }
}
</style>
