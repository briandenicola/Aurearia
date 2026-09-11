<template>
  <main class="storage-trays-page">
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
      <StorageTrayGrid
        v-for="tray in trays"
        :key="tray.id"
        :tray="tray"
        @coin-clicked="openCoin"
      />
    </div>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getStorageTrays } from '@/api/client'
import type { TrayAggregate } from '@/types'
import StorageTrayGrid from '@/components/tray/StorageTrayGrid.vue'

const router = useRouter()
const trays = ref<TrayAggregate[]>([])
const loading = ref(true)
const error = ref('')

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

onMounted(load)
</script>

<style scoped>
.storage-trays-page {
  max-width: 100%;
}
.tray-list {
  display: grid;
  gap: 1.5rem;
}
</style>
