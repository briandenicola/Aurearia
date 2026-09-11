<template>
  <main class="container storage-trays-page">
    <header class="page-header">
      <h1>Storage Trays</h1>
      <router-link class="btn btn-secondary btn-sm" to="/">Back to Collection</router-link>
    </header>

    <p v-if="loading" role="status" class="text-text-secondary">Loading storage trays...</p>
    <div v-else-if="error" class="rounded-md border border-border-subtle bg-card p-5" role="alert">
      <p>{{ error }}</p>
      <button type="button" class="btn btn-primary btn-sm" @click="load">Retry</button>
    </div>
    <div v-else-if="trays.length === 0" class="rounded-md border border-border-subtle bg-card p-5">
      <p>No coin trays are configured yet.</p>
      <router-link to="/settings" class="btn btn-primary btn-sm">Open Settings</router-link>
    </div>
    <div v-else class="tray-list">
      <div class="tray-controls-row">
        <label class="tray-size-control" for="storage-tray-size-slider">
          <span>Coin size</span>
          <input
            id="storage-tray-size-slider"
            v-model.number="traySizeScale"
            class="tray-size-slider"
            type="range"
            min="0.75"
            max="2.5"
            step="0.05"
          />
          <span class="tray-size-value">{{ traySizeScale.toFixed(2) }}x</span>
        </label>
      </div>
      <StorageTrayGrid
        v-for="tray in trays"
        :key="tray.id"
        :tray="tray"
        :felt-theme="feltColor"
        :size-scale="traySizeScale"
        @coin-clicked="openCoin"
      />
    </div>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getStorageTrays } from '@/api/client'
import { useTrayPreference } from '@/composables/useTrayPreference'
import type { TrayAggregate } from '@/types'
import StorageTrayGrid from '@/components/tray/StorageTrayGrid.vue'

const router = useRouter()
const { feltColor } = useTrayPreference()
const trays = ref<TrayAggregate[]>([])
const loading = ref(true)
const error = ref('')
const traySizeScale = ref(1)

watch(traySizeScale, (value) => {
  const normalizedValue = Math.min(2.5, Math.max(0.75, Number(value) || 1))
  traySizeScale.value = normalizedValue
  localStorage.setItem('tray:sizeScale', normalizedValue.toString())
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const response = await getStorageTrays()
    trays.value = response.data?.trays ?? []
  } catch {
    trays.value = []
    error.value = 'Storage trays could not be loaded.'
  } finally {
    loading.value = false
  }
}

function openCoin(coinId: number) {
  void router.push({ name: 'coin-detail', params: { id: coinId } })
}

onMounted(() => {
  const storedScale = Number(localStorage.getItem('tray:sizeScale'))
  if (Number.isFinite(storedScale) && storedScale > 0) {
    traySizeScale.value = Math.min(2.5, Math.max(0.75, storedScale))
  }
  void load()
})
</script>

<style scoped>
.storage-trays-page {
  padding-bottom: 1.5rem;
}
.tray-list {
  display: grid;
  gap: 1.5rem;
}
.tray-controls-row {
  display: flex;
  justify-content: flex-end;
}
.tray-size-control {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.55rem 0.9rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.04);
  color: var(--text-secondary);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}
.tray-size-slider {
  width: 140px;
  accent-color: var(--accent-gold);
}
.tray-size-value {
  min-width: 3.2rem;
  text-align: right;
  color: var(--text-primary);
}
@media (min-width: 769px) {
  .storage-trays-page {
    max-width: none;
  }
}
@media (max-width: 575px) {
  .tray-controls-row {
    justify-content: center;
  }
  .tray-size-control {
    width: 100%;
    justify-content: space-between;
  }
}
</style>
