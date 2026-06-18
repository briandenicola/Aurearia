<template>
  <section class="unattributed card" aria-labelledby="unattributed-title">
    <button
      class="unattributed-toggle"
      type="button"
      :aria-expanded="expanded"
      aria-controls="unattributed-content"
      @click="$emit('update:expanded', !expanded)"
    >
      <span>
        <span id="unattributed-title" class="section-label">Unattributed Mints</span>
        <strong>{{ totalCount }} {{ totalCount === 1 ? 'coin needs' : 'coins need' }} mint review</strong>
      </span>
      <ChevronDown class="toggle-icon" :class="{ expanded }" :size="18" />
    </button>

    <div v-if="expanded" id="unattributed-content" class="unattributed-content">
      <div v-if="unknown.length" class="bucket-section">
        <h3>Unknown mint</h3>
        <ul class="coin-list">
          <li v-for="coin in unknown" :key="coin.id">
            <router-link :to="`/coin/${coin.id}`">{{ coin.name }}</router-link>
          </li>
        </ul>
      </div>

      <div v-for="group in unmatched" :key="group.normalizedName" class="bucket-section">
        <h3>{{ group.originalNames.join(', ') }}</h3>
        <p class="bucket-hint">No static mint coordinate matched this name.</p>
        <ul class="coin-list">
          <li v-for="coin in group.coins" :key="coin.id">
            <router-link :to="`/coin/${coin.id}`">{{ coin.name }}</router-link>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ChevronDown } from 'lucide-vue-next'
import type { Coin } from '@/types'
import type { UnmatchedMintGroup } from '@/utils/mintMap'

const props = defineProps<{
  unknown: Coin[]
  unmatched: UnmatchedMintGroup[]
  expanded: boolean
}>()

defineEmits<{
  'update:expanded': [value: boolean]
}>()

const totalCount = computed(() =>
  props.unknown.length + props.unmatched.reduce((total, group) => total + group.coins.length, 0),
)
</script>

<style scoped>
.unattributed {
  padding: 1rem;
}

.unattributed-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border: 0;
  background: transparent;
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}

.unattributed-toggle strong {
  display: block;
  margin-top: 0.25rem;
  color: var(--text-primary);
  font-size: 0.9rem;
}

.toggle-icon {
  flex-shrink: 0;
  color: var(--accent-gold);
  transition: transform var(--transition-fast);
}

.toggle-icon.expanded {
  transform: rotate(180deg);
}

.unattributed-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-top: 1rem;
}

.bucket-section {
  border-top: 1px solid var(--border-subtle);
  padding-top: 0.75rem;
}

.bucket-section h3 {
  margin: 0 0 0.25rem;
}

.bucket-hint {
  margin: 0 0 0.5rem;
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.coin-list {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin: 0;
  padding-left: 1rem;
}

.coin-list a {
  color: var(--accent-gold);
}
</style>
