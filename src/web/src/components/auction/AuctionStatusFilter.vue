<template>
  <div class="status-filter-menu" @keydown.esc="menuOpen = false">
    <button
      type="button"
      class="btn btn-sm btn-ghost menu-button"
      :class="{ active: modelValue }"
      aria-label="Auction status filters"
      aria-haspopup="menu"
      :aria-expanded="menuOpen"
      title="Status filters"
      @click="menuOpen = !menuOpen"
    >
      <Menu :size="18" />
    </button>

    <div v-if="menuOpen" class="status-menu" role="menu" aria-label="Auction status filters">
      <button
        v-for="s in statuses"
        :key="s.value"
        type="button"
        class="chip status-option"
        :class="{ active: modelValue === s.value }"
        role="menuitemradio"
        :aria-checked="modelValue === s.value"
        @click="selectStatus(s.value)"
      >
        {{ s.label }}
        <span v-if="counts[s.value]" class="count-badge">{{ counts[s.value] }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Menu } from 'lucide-vue-next'

defineProps<{
  modelValue: string
  counts: Record<string, number>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const menuOpen = ref(false)

const statuses = [
  { value: '', label: 'All' },
  { value: 'watching', label: 'Watching' },
  { value: 'bidding', label: 'Bidding' },
  { value: 'won', label: 'Won' },
  { value: 'lost', label: 'Lost' },
  { value: 'passed', label: 'Passed' },
]

function selectStatus(value: string) {
  emit('update:modelValue', value)
  menuOpen.value = false
}
</script>

<style scoped>
.status-filter-menu {
  position: relative;
  display: flex;
  justify-content: flex-end;
}

.menu-button {
  justify-content: center;
  padding-inline: 0.75rem;
}

.menu-button.active {
  background: var(--accent-gold-glow);
  border-color: var(--accent-gold);
  color: var(--accent-gold);
}

.status-menu {
  position: absolute;
  top: calc(100% + 0.35rem);
  right: 0;
  z-index: 5;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0.35rem;
  min-width: 9rem;
  padding: 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  box-shadow: var(--shadow-card);
}

.status-option {
  justify-content: space-between;
  gap: 0.5rem;
  width: 100%;
}

.count-badge {
  background: var(--bg-primary);
  padding: 0.05rem 0.4rem;
  border-radius: var(--radius-full);
  font-size: 0.7rem;
  font-weight: 600;
}
</style>
