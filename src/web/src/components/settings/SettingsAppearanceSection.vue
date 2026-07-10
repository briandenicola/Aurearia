<template>
  <section class="settings-section card">
    <h2>Appearance</h2>
    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Theme</span>
        <span class="setting-desc">Choose your preferred color scheme</span>
      </div>
      <div class="theme-toggle theme-toggle-grid">
        <button
          class="theme-btn"
          :class="{ active: theme === 'dark' }"
          @click="$emit('set-theme', 'dark')"
        >
          Dark
        </button>
        <button
          class="theme-btn"
          :class="{ active: theme === 'light' }"
          @click="$emit('set-theme', 'light')"
        >
          Light
        </button>
        <button
          class="theme-btn"
          :class="{ active: theme === 'british-museum' }"
          @click="$emit('set-theme', 'british-museum')"
        >
          British Museum
        </button>
        <button
          class="theme-btn"
          :class="{ active: theme === 'louvre' }"
          @click="$emit('set-theme', 'louvre')"
        >
          Louvre
        </button>
        <button
          class="theme-btn"
          :class="{ active: theme === 'capitoline' }"
          @click="$emit('set-theme', 'capitoline')"
        >
          Capitoline
        </button>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Timezone</span>
        <span class="setting-desc">Used for date display</span>
      </div>
      <select
        :value="timezone"
        class="form-select tz-select"
        @change="$emit('save-timezone', ($event.target as HTMLSelectElement).value)"
      >
        <option v-for="tz in timezones" :key="tz" :value="tz">{{ tz }}</option>
      </select>
    </div>

    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Default View</span>
        <span class="setting-desc">Preferred collection view on mobile / PWA</span>
      </div>
      <div class="theme-toggle">
        <button
          class="theme-btn"
          :class="{ active: defaultView === 'swipe' }"
          @click="$emit('set-default-view', 'swipe')"
        >
          Swipe
        </button>
        <button
          class="theme-btn"
          :class="{ active: defaultView === 'grid' }"
          @click="$emit('set-default-view', 'grid')"
        >
          Grid
        </button>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Tray Felt Color</span>
        <span class="setting-desc">Choose the museum tray background color</span>
      </div>
      <div class="theme-toggle felt-toggle">
        <button
          class="theme-btn felt-btn felt-btn-red"
          :class="{ active: trayFeltColor === 'red' }"
          @click="$emit('set-tray-felt-color', 'red')"
        >
          Red
        </button>
        <button
          class="theme-btn felt-btn felt-btn-green"
          :class="{ active: trayFeltColor === 'green' }"
          @click="$emit('set-tray-felt-color', 'green')"
        >
          Green
        </button>
        <button
          class="theme-btn felt-btn felt-btn-navy"
          :class="{ active: trayFeltColor === 'navy' }"
          @click="$emit('set-tray-felt-color', 'navy')"
        >
          Navy
        </button>
      </div>
    </div>

    <div class="setting-item">
      <div class="setting-info">
        <span class="setting-label">Default Sort</span>
        <span class="setting-desc">How coins are sorted by default</span>
      </div>
      <select
        :value="defaultSort"
        class="form-select sort-select"
        @change="$emit('save-default-sort', ($event.target as HTMLSelectElement).value)"
      >
        <option value="updated_at_desc">Last Updated</option>
        <option value="created_at_desc">Newest First</option>
        <option value="created_at_asc">Oldest First</option>
        <option value="current_value_desc">Price: High → Low</option>
        <option value="current_value_asc">Price: Low → High</option>
        <option value="random_desc">Random</option>
      </select>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Theme } from '@/types'
import type { FeltColor } from '@/composables/useTrayPreference'

defineProps<{
  theme: Theme
  timezone: string
  timezones: string[]
  defaultView: 'grid' | 'swipe'
  defaultSort: string
  trayFeltColor: FeltColor
}>()

defineEmits<{
  'set-theme': [theme: Theme]
  'save-timezone': [tz: string]
  'set-default-view': [view: 'grid' | 'swipe']
  'save-default-sort': [sort: string]
  'set-tray-felt-color': [color: FeltColor]
}>()
</script>

<style scoped>
.theme-toggle {
  display: flex;
  gap: 0.25rem;
  background: var(--bg-primary);
  border-radius: var(--radius-full);
  padding: 0.2rem;
}

.theme-toggle-grid {
  border-radius: var(--radius-sm);
  flex-wrap: wrap;
}

.theme-btn {
  padding: 0.35rem 0.75rem;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.8rem;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.theme-btn.active {
  background: var(--accent-gold-dim);
  color: var(--accent-gold);
}

.felt-toggle {
  flex-wrap: wrap;
}

.felt-btn-red.active {
  background: var(--felt-red-dim);
  color: var(--felt-red-bright);
}

.felt-btn-green.active {
  background: var(--felt-green-dim);
  color: var(--felt-green-bright);
}

.felt-btn-navy.active {
  background: var(--felt-navy-dim);
  color: var(--felt-navy-bright);
}

.tz-select {
  max-width: 250px;
}

.sort-select {
  max-width: 250px;
}

.settings-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 0;
  border-bottom: 1px solid var(--border-subtle);
  gap: 1rem;
}

.setting-item:last-child {
  border-bottom: none;
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.setting-label {
  font-size: 0.9rem;
  font-weight: 500;
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

@media (max-width: 640px) {
  .setting-item {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
