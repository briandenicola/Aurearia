<template>
  <section class="card">
    <h2 class="text-lg font-medium mb-5 pb-3 border-b border-border-subtle">Appearance</h2>

    <!-- Theme -->
    <div class="flex justify-between items-center py-3 border-b border-border-subtle gap-4 last:border-0 md:flex-row flex-col md:items-center items-stretch">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Theme</span>
        <span class="text-sm text-text-muted">Choose your preferred color scheme</span>
      </div>
      <div class="flex flex-wrap gap-1 bg-surface rounded-[var(--radius-md)] p-[0.2rem]">
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="theme === 'dark' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-theme', 'dark')"
        >
          Dark
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="theme === 'light' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-theme', 'light')"
        >
          Light
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="theme === 'british-museum' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-theme', 'british-museum')"
        >
          British Museum
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="theme === 'louvre' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-theme', 'louvre')"
        >
          Louvre
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="theme === 'capitoline' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-theme', 'capitoline')"
        >
          Capitoline
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="theme === 'byzantine' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-theme', 'byzantine')"
        >
          Byzantine
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="theme === 'modern-greek' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-theme', 'modern-greek')"
        >
          Modern Greek
        </button>
      </div>
    </div>

    <!-- Timezone -->
    <div class="flex justify-between items-center py-3 border-b border-border-subtle gap-4 md:flex-row flex-col md:items-center items-stretch">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Timezone</span>
        <span class="text-sm text-text-muted">Used for date display</span>
      </div>
      <select
        :value="timezone"
        class="form-select max-w-[250px]"
        @change="$emit('save-timezone', ($event.target as HTMLSelectElement).value)"
      >
        <option v-for="tz in timezones" :key="tz" :value="tz">{{ tz }}</option>
      </select>
    </div>

    <!-- Default View -->
    <div class="flex justify-between items-center py-3 border-b border-border-subtle gap-4 md:flex-row flex-col md:items-center items-stretch">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Default View</span>
        <span class="text-sm text-text-muted">Preferred collection view on mobile / PWA</span>
      </div>
      <div class="flex gap-1 bg-surface rounded-full p-[0.2rem]">
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="defaultView === 'swipe' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-default-view', 'swipe')"
        >
          Swipe
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="defaultView === 'grid' ? 'bg-gold-dim text-gold' : 'bg-transparent text-text-secondary'"
          @click="$emit('set-default-view', 'grid')"
        >
          Grid
        </button>
      </div>
    </div>

    <!-- Tray Felt Color -->
    <div class="flex justify-between items-center py-3 border-b border-border-subtle gap-4 md:flex-row flex-col md:items-center items-stretch">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Tray Felt Color</span>
        <span class="text-sm text-text-muted">Choose the museum tray background color</span>
      </div>
      <div class="flex flex-wrap gap-1 bg-surface rounded-full p-[0.2rem]">
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="trayFeltColor === 'red'
            ? 'bg-[var(--felt-red-dim)] text-[var(--felt-red-bright)]'
            : 'bg-transparent text-text-secondary'"
          @click="$emit('set-tray-felt-color', 'red')"
        >
          Red
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="trayFeltColor === 'green'
            ? 'bg-[var(--felt-green-dim)] text-[var(--felt-green-bright)]'
            : 'bg-transparent text-text-secondary'"
          @click="$emit('set-tray-felt-color', 'green')"
        >
          Green
        </button>
        <button
          class="px-3 py-[0.35rem] rounded-full text-chip cursor-pointer transition-all duration-200 border-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
          :class="trayFeltColor === 'navy'
            ? 'bg-[var(--felt-navy-dim)] text-[var(--felt-navy-bright)]'
            : 'bg-transparent text-text-secondary'"
          @click="$emit('set-tray-felt-color', 'navy')"
        >
          Navy
        </button>
      </div>
    </div>

    <!-- Default Sort -->
    <div class="flex justify-between items-center py-3 gap-4 md:flex-row flex-col md:items-center items-stretch">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Default Sort</span>
        <span class="text-sm text-text-muted">How coins are sorted by default</span>
      </div>
      <select
        :value="defaultSort"
        class="form-select max-w-[250px]"
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
    <!-- Swipe Navigation -->
    <div class="flex justify-between items-center py-3 gap-4 md:flex-row flex-col md:items-center items-stretch">
      <div class="flex flex-col gap-[0.15rem]">
        <span class="text-base font-medium">Swipe Navigation</span>
        <span class="text-sm text-text-muted">Swipe left or right on a coin's pages to move between its sections. Applies to the installed app only; it has no effect in a web browser.</span>
        <span v-if="pwaSwipeNavError" class="mt-1 text-chip text-[var(--color-negative)]">{{ pwaSwipeNavError }}</span>
      </div>
      <label class="relative inline-block h-7 w-[50px] shrink-0">
        <input
          type="checkbox"
          class="peer sr-only"
          :checked="pwaSwipeNavLocal"
          :disabled="pwaSwipeNavSaving"
          @change="handlePwaSwipeNavChange(($event.target as HTMLInputElement).checked)"
        />
        <span
          class="absolute inset-0 cursor-pointer rounded-full border border-border-subtle bg-[var(--bg-primary)] transition-colors peer-checked:border-gold peer-checked:bg-gold-dim peer-focus-visible:outline-2 peer-focus-visible:outline-gold peer-focus-visible:outline-offset-2 after:absolute after:bottom-[3px] after:left-[3px] after:h-5 after:w-5 after:rounded-full after:bg-text-secondary after:transition-transform after:content-[''] peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"
        ></span>
      </label>
    </div>
  </section></template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Theme } from '@/types'
import type { FeltColor } from '@/composables/useTrayPreference'
import { useSettingsProfile } from '@/composables/useSettingsProfile'

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

// savePwaSwipeNav handles auth-store sync, localStorage write, optimistic update, and rollback.
const { pwaSwipeNavEnabled: pwaSwipeNavLocal, savePwaSwipeNav } = useSettingsProfile()
const pwaSwipeNavError = ref('')
const pwaSwipeNavSaving = ref(false)

async function handlePwaSwipeNavChange(value: boolean) {
  pwaSwipeNavError.value = ''
  pwaSwipeNavSaving.value = true
  try {
    await savePwaSwipeNav(value)
  } catch {
    pwaSwipeNavError.value = 'Failed to save swipe navigation setting'
  } finally {
    pwaSwipeNavSaving.value = false
  }
}
</script>