<template>
  <div class="container">
    <div class="page-header">
      <h1>Sets</h1>
      <div v-if="isPwa" class="pwa-actions">
        <button class="pwa-icon-btn" @click="showCreateModal = true" title="Create Set">
          <CirclePlus :size="22" />
        </button>
      </div>
      <div v-else class="header-actions">
        <button class="btn btn-primary" @click="showCreateModal = true">
          <Plus :size="16" /> Create Set
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading sets...</p>
    </div>

    <div v-else-if="sets.length === 0" class="empty-state">
      <Layers3 :size="48" />
      <h3>No sets yet</h3>
      <p>Create a set to organize your collection by theme, era, or completion goals</p>
      <button class="btn btn-primary empty-action" @click="showCreateModal = true">
        <Plus :size="16" /> Create Your First Set
      </button>
    </div>

    <div v-else class="sets-grid">
      <SetDashboardCard
        v-for="set in sets"
        :key="set.id"
        :set="set"
        @click="goToSet(set.id)"
      />
    </div>

    <!-- Create Set Modal -->
    <Teleport to="body">
      <div v-if="showCreateModal" class="modal-overlay" @click.self="showCreateModal = false">
        <div class="modal-content card">
          <div class="modal-header">
            <h2>Create New Set</h2>
            <button class="modal-close" @click="showCreateModal = false">
              <X :size="20" />
            </button>
          </div>
          <SetCreationWizard
            :initial-value="newSet"
            submit-label="Create"
            @submit="createSet"
            @cancel="showCreateModal = false"
          />
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { CirclePlus, Layers3, Plus, X } from 'lucide-vue-next'
import { getSets, createSet as createSetApi, createSetFromCsv } from '@/api/client'
import type { CoinSetSummary, CreateCoinSetRequest } from '@/types'
import SetDashboardCard from '@/components/sets/SetDashboardCard.vue'
import SetCreationWizard from '@/components/sets/SetCreationWizard.vue'
import { usePwa } from '@/composables/usePwa'

const router = useRouter()
const { isPwa } = usePwa()
const loading = ref(true)
const sets = ref<CoinSetSummary[]>([])
const showCreateModal = ref(false)
const newSet = ref({
  name: '',
  description: '',
  color: '#6b7280',
  setType: 'open' as const,
})

onMounted(async () => {
  await loadSets()
})

async function loadSets() {
  loading.value = true
  try {
    const res = await getSets()
    sets.value = res.data.sets
  } catch (error) {
    console.error('Failed to load sets:', error)
  } finally {
    loading.value = false
  }
}

async function createSet(value: CreateCoinSetRequest, csv?: string) {
  try {
    if (csv) {
      await createSetFromCsv({ ...value, csv })
    } else {
      await createSetApi(value)
    }
    showCreateModal.value = false
    newSet.value = {
      name: '',
      description: '',
      color: '#6b7280',
      setType: 'open',
    }
    await loadSets()
  } catch (error) {
    console.error('Failed to create set:', error)
    alert('Failed to create set')
  }
}

function goToSet(id: number) {
  router.push({ name: 'set-detail', params: { id } })
}
</script>

<style scoped>
.loading-overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
  padding: 3rem 1rem;
  color: var(--text-secondary);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: var(--radius-full);
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 4rem 1rem;
  color: var(--text-secondary);
  text-align: center;
}

.empty-state h3 {
  margin: 0.75rem 0 0;
  font-size: 1.1rem;
  color: var(--text-primary);
}

.empty-state p {
  margin: 0;
  max-width: 32rem;
  font-size: 0.9rem;
  color: var(--text-muted);
}

.empty-action {
  margin-top: 1rem;
}

.sets-grid {
  display: flex;
  flex-direction: column;
  gap: 0.9rem;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 2rem 1rem;
}

.modal-content {
  max-width: 520px;
  width: 100%;
  max-height: 85vh;
  overflow-y: auto;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--text-primary);
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast);
  display: flex;
}

.modal-close:hover {
  color: var(--text-primary);
}
</style>
