<template>
  <div class="container">
    <div class="page-header">
      <h1>Sets</h1>
      <div v-if="isPwa" class="pwa-actions">
        <button class="pwa-icon-btn" @click="openCreateModal" title="Create Set">
          <CirclePlus :size="22" />
        </button>
      </div>
      <div v-else class="header-actions">
        <button class="btn btn-primary" @click="openCreateModal">
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
      <button class="btn btn-primary mt-4" @click="openCreateModal">
        <Plus :size="16" /> Create Your First Set
      </button>
    </div>

    <div v-else class="flex flex-col gap-4">
      <SetDashboardCard
        v-for="set in sets"
        :key="set.id"
        :set="set"
        @click="goToSet(set.id)"
      />
    </div>

    <Teleport to="body">
      <div v-if="showCreateModal" class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/60 px-4 py-8" @click.self="showCreateModal = false">
        <div class="card max-h-[85vh] w-full max-w-[520px] overflow-y-auto">
          <div class="mb-6 flex items-center justify-between gap-4">
            <h2 class="m-0 text-lg text-text-primary">Create New Set</h2>
            <button class="inline-flex rounded-sm p-1 text-text-secondary transition-colors hover:text-text-primary" @click="showCreateModal = false">
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
import { formatSetBuilderError, getApiErrorMessage, getSets, createSet as createSetApi, createSetFromCsv, createSetBuilderRun } from '@/api/client'
import type { CoinSetSummary, CreateCoinSetRequest } from '@/types'
import SetDashboardCard from '@/components/sets/SetDashboardCard.vue'
import SetCreationWizard from '@/components/sets/SetCreationWizard.vue'
import { usePwa } from '@/composables/usePwa'
import { randomSetColor } from '@/utils/setColors'
import { useDialog } from '@/composables/useDialog'

const router = useRouter()
const { isPwa } = usePwa()
const { showAlert } = useDialog()
const loading = ref(true)
const sets = ref<CoinSetSummary[]>([])
const showCreateModal = ref(false)
const newSet = ref({
  name: '',
  description: '',
  color: randomSetColor(),
  setType: 'standard' as const,
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

function resetNewSet() {
  newSet.value = {
    name: '',
    description: '',
    color: randomSetColor(),
    setType: 'standard',
  }
}

function openCreateModal() {
  resetNewSet()
  showCreateModal.value = true
}

async function createSet(value: CreateCoinSetRequest, csv?: string) {
  try {
    if (value.setType === 'agentic') {
      const res = await createSetBuilderRun({ prompt: value.agenticPrompt || value.name })
      showCreateModal.value = false
      resetNewSet()
      await showAlert(
        `Your set proposal request has been submitted as run #${res.data.run.id}. The workflow will draft roster slots, estimate matches from your collection where data exists, and notify you when review is ready.`,
        { title: 'Proposal Request Submitted' },
      )
      return
    }
    if (csv) {
      await createSetFromCsv({ ...value, csv })
    } else {
      await createSetApi(value)
    }
    showCreateModal.value = false
    resetNewSet()
    await loadSets()
  } catch (error) {
    console.error('Failed to create set:', error)
    const title = value.setType === 'agentic' ? 'Failed to Submit Proposal Request' : 'Failed to Create Set'
    const message = value.setType === 'agentic' ? formatSetBuilderError(error) : (getApiErrorMessage(error) || 'Request failed.')
    await showAlert(message, { title })
  }
}

function goToSet(id: number) {
  router.push({ name: 'set-detail', params: { id } })
}
</script>
