<template>
  <div class="pwa-header">
    <SearchBar :modelValue="search" @update:modelValue="$emit('update:search', $event)" />
    <div class="hamburger-wrapper">
      <button class="hamburger-btn" @click="$emit('update:menuOpen', !menuOpen)" :class="{ active: menuOpen }">
        <SlidersHorizontal :size="22" />
      </button>
      <Transition name="menu-slide">
        <div v-if="menuOpen" class="pwa-menu">
          <div class="pwa-menu-section">
            <span class="pwa-menu-label">Selection</span>
            <button class="menu-toggle-btn" :class="{ active: selectMode }" @click="$emit('toggle-select-mode')">
              {{ selectMode ? 'Exit Selection Mode' : 'Enable Selection Mode' }}
            </button>
          </div>
          <div class="pwa-menu-section">
            <span class="pwa-menu-label">Category</span>
            <CategoryFilter :modelValue="selectedCategory" @update:modelValue="$emit('update:selectedCategory', $event)" />
          </div>
          <div v-if="userTags.length" class="pwa-menu-section">
            <span class="pwa-menu-label">Tag</span>
            <select :value="selectedTag" @change="$emit('update:selectedTag', ($event.target as HTMLSelectElement).value)" class="tag-filter-select pwa-tag-select">
              <option value="">All Tags</option>
              <option v-for="tag in userTags" :key="tag.id" :value="String(tag.id)">{{ tag.name }}</option>
            </select>
          </div>
          <div class="pwa-menu-section">
            <span class="pwa-menu-label">Sort</span>
            <SortSelect :modelValue="sortKey" @update:modelValue="$emit('update:sortKey', $event)" />
          </div>
          <div class="pwa-menu-section">
            <span class="pwa-menu-label">View</span>
            <div class="pwa-menu-row">
              <div class="view-toggle">
                <button class="view-btn" :class="{ active: viewMode === 'swipe' }" @click="$emit('update:viewMode', 'swipe')" title="Swipe view">
                  <Layers :size="18" />
                </button>
                <button class="view-btn" :class="{ active: viewMode === 'grid' }" @click="$emit('update:viewMode', 'grid')" title="Grid view">
                  <LayoutGrid :size="18" />
                </button>
              </div>
            </div>
          </div>
          <div v-if="viewMode === 'grid'" class="pwa-menu-section">
            <span class="pwa-menu-label">Face</span>
            <div class="face-filter">
              <button class="chip" :class="{ active: gridSide === 'obverse' }" @click="$emit('update:gridSide', gridSide === 'obverse' ? null : 'obverse')">Obverse</button>
              <button class="chip" :class="{ active: gridSide === 'reverse' }" @click="$emit('update:gridSide', gridSide === 'reverse' ? null : 'reverse')">Reverse</button>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </div>
  <div v-if="menuOpen" class="pwa-menu-backdrop" @click="$emit('update:menuOpen', false)"></div>
</template>

<script setup lang="ts">
import type { ImageType, Tag } from '@/types'
import CategoryFilter from '@/components/CategoryFilter.vue'
import SearchBar from '@/components/SearchBar.vue'
import SortSelect from '@/components/SortSelect.vue'
import { Layers, LayoutGrid, SlidersHorizontal } from 'lucide-vue-next'

defineProps<{
  search: string
  selectMode: boolean
  menuOpen: boolean
  selectedCategory: string
  selectedTag: string
  userTags: Tag[]
  sortKey: string
  viewMode: 'grid' | 'swipe'
  gridSide: ImageType | null
}>()

defineEmits<{
  'update:search': [value: string]
  'update:menuOpen': [value: boolean]
  'update:selectedCategory': [value: string]
  'update:selectedTag': [value: string]
  'update:sortKey': [value: string]
  'update:viewMode': [value: 'grid' | 'swipe']
  'update:gridSide': [value: ImageType | null]
  'toggle-select-mode': []
}>()
</script>

<style scoped>
.pwa-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.75rem;
  position: sticky;
  top: 60px;
  z-index: 150;
  background: var(--bg-primary);
  padding: 0.5rem 0;
}

.pwa-header :deep(.search-bar) {
  flex: 1;
  max-width: none;
}

.hamburger-wrapper {
  position: relative;
  flex-shrink: 0;
}

.hamburger-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.hamburger-btn.active,
.hamburger-btn:hover {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
  background: var(--accent-gold-dim);
}

.pwa-menu {
  position: absolute;
  top: calc(100% + 0.5rem);
  right: 0;
  z-index: 100;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1rem;
  min-width: 260px;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.4);
}

.pwa-menu-backdrop {
  position: fixed;
  inset: 0;
  z-index: 90;
}

.pwa-menu-section {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.pwa-menu-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  font-weight: 600;
}

.pwa-menu-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  flex-wrap: wrap;
}

.menu-slide-enter-active,
.menu-slide-leave-active {
  transition: all 0.2s ease;
}
.menu-slide-enter-from,
.menu-slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
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

.pwa-tag-select {
  width: 100%;
}

.menu-toggle-btn {
  width: 100%;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  color: var(--text-secondary);
  padding: 0.5rem 0.7rem;
  font-size: 0.85rem;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.menu-toggle-btn.active {
  border-color: var(--accent-gold);
  color: var(--accent-gold);
  background: var(--accent-gold-dim);
}

.menu-toggle-btn:hover {
  border-color: var(--accent-gold);
}

.view-toggle {
  display: flex;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.view-btn {
  padding: 0.4rem 0.6rem;
  border: none;
  background: var(--bg-card);
  color: var(--text-secondary);
  font-size: 1rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  line-height: 1;
}

.view-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.view-btn:hover:not(.active) {
  background: var(--bg-card-hover);
}

.face-filter {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}
</style>
