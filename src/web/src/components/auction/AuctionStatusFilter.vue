<template>
  <div class="filter-bar">
    <div class="status-filters">
      <button
        v-for="s in statuses"
        :key="s.value"
        class="filter-btn"
        :class="{ active: modelValue === s.value }"
        @click="$emit('update:modelValue', s.value)"
      >
        {{ s.label }}
        <span v-if="counts[s.value]" class="count-badge">{{ counts[s.value] }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  modelValue: string
  counts: Record<string, number>
}>()

defineEmits<{
  'update:modelValue': [value: string]
}>()

const statuses = [
  { value: '', label: 'All' },
  { value: 'watching', label: 'Watching' },
  { value: 'bidding', label: 'Bidding' },
  { value: 'won', label: 'Won' },
  { value: 'lost', label: 'Lost' },
  { value: 'passed', label: 'Passed' },
]
</script>

<style scoped>
.filter-bar {
  margin-bottom: 1.25rem;
}

.status-filters {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.filter-btn {
  padding: 0.4rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.82rem;
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  gap: 0.4rem;
}

.filter-btn:hover {
  border-color: var(--accent-gold-dim);
  color: var(--text-primary);
}

.filter-btn.active {
  background: var(--accent-gold-glow);
  border-color: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.count-badge {
  background: var(--bg-primary);
  padding: 0.05rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}
</style>
