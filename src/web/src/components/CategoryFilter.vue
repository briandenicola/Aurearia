<template>
  <div class="category-filter">
    <button
      class="filter-btn"
      :class="{ active: !modelValue }"
      @click="$emit('update:modelValue', '')"
    >
      All
    </button>
    <button
      v-for="cat in CATEGORIES"
      :key="cat"
      class="filter-btn"
      :class="[`filter-${cat.toLowerCase()}`, { active: modelValue === cat }]"
      @click="$emit('update:modelValue', cat)"
    >
      {{ cat }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { CATEGORIES } from '@/types'

defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<style scoped>
.category-filter {
  display: flex;
  gap: 0.4rem;
  flex-wrap: wrap;
}

.filter-btn {
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.8rem;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.filter-btn:hover {
  border-color: var(--border-accent);
  color: var(--text-primary);
}

.filter-btn.active {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.filter-roman.active { background: color-mix(in srgb, var(--cat-roman) 20%, transparent); border-color: var(--cat-roman); color: var(--cat-roman); }
.filter-greek.active { background: color-mix(in srgb, var(--cat-greek) 20%, transparent); border-color: var(--cat-greek); color: var(--cat-greek); }
.filter-byzantine.active { background: color-mix(in srgb, var(--cat-byzantine) 20%, transparent); border-color: var(--cat-byzantine); color: var(--cat-byzantine); }
.filter-modern.active { background: color-mix(in srgb, var(--cat-modern) 20%, transparent); border-color: var(--cat-modern); color: var(--cat-modern); }
.filter-other.active { background: color-mix(in srgb, var(--cat-other) 20%, transparent); border-color: var(--cat-other); color: var(--cat-other); }
</style>
