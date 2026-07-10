<template>
  <div class="container">
    <div class="page-header">
      <h1>Showcases</h1>
      <div v-if="isPwa" class="pwa-actions">
        <button class="pwa-icon-btn" @click="showCreate = true" title="New Showcase">
          <CirclePlus :size="22" />
        </button>
      </div>
      <div v-else class="header-actions">
        <button class="btn btn-primary" @click="showCreate = true">
          <Plus :size="16" /> New Showcase
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
      <p>Loading showcases...</p>
    </div>

    <div v-else-if="!showcases.length" class="empty-state">
      <Presentation :size="48" />
      <h3>No showcases yet</h3>
      <p>Create a showcase to share a curated selection of your coins with the world.</p>
      <button class="btn btn-primary mt-4" @click="showCreate = true">
        <Plus :size="16" /> Create Your First Showcase
      </button>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
      <div v-for="sc in showcases" :key="sc.id" class="card flex flex-col gap-3 p-5">
        <div class="flex items-start justify-between gap-2">
          <h3 class="m-0 text-[1.1rem] text-text-primary">{{ sc.title }}</h3>
          <span class="badge whitespace-nowrap" :class="sc.isActive ? 'bg-[var(--accent-gold-glow)] text-gold' : 'bg-[var(--accent-gold-glow)] text-text-muted'">
            {{ sc.isActive ? 'Active' : 'Inactive' }}
          </span>
        </div>
        <p v-if="sc.description" class="m-0 text-body leading-[1.4] text-text-secondary">{{ sc.description }}</p>
        <div class="flex flex-col gap-2 text-chip text-text-secondary md:flex-row md:items-center md:justify-between">
          <span class="inline-flex items-center gap-1"><Coins :size="14" /> {{ sc.coinCount ?? 0 }} coins</span>
          <span class="font-mono opacity-70">/s/{{ sc.slug }}</span>
        </div>
        <div class="mt-auto flex flex-wrap gap-2">
          <router-link :to="`/showcases/${sc.id}/edit`" class="btn btn-secondary btn-sm">
            <Pencil :size="14" /> Edit
          </router-link>
          <button class="btn btn-secondary btn-sm" @click="copyLink(sc.slug)">
            <Link :size="14" /> Copy Link
          </button>
          <button
            class="btn btn-sm"
            :class="sc.isActive ? 'btn-secondary' : 'btn-primary'"
            @click="toggleActive(sc)"
          >
            <Eye v-if="sc.isActive" :size="14" />
            <EyeOff v-else :size="14" />
            {{ sc.isActive ? 'Deactivate' : 'Activate' }}
          </button>
          <button class="btn btn-danger btn-sm" @click="confirmDelete(sc)">
            <Trash2 :size="14" />
          </button>
        </div>
      </div>
    </div>

    <div v-if="copied" class="fixed bottom-8 left-1/2 z-[1000] -translate-x-1/2 rounded-sm bg-[var(--accent-gold)] px-5 py-2 text-body font-medium text-[var(--bg-primary)]">
      Link copied to clipboard
    </div>

    <div v-if="showCreate" class="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 px-4 py-6" @click.self="showCreate = false">
      <div class="card w-full max-w-[480px] p-6">
        <div class="mb-4 flex items-center justify-between gap-4">
          <h2 class="m-0 text-lg text-text-primary">New Showcase</h2>
          <button class="inline-flex rounded-sm p-1 text-text-secondary transition-colors hover:text-text-primary" @click="showCreate = false"><X :size="18" /></button>
        </div>
        <form @submit.prevent="handleCreate">
          <div class="form-group">
            <label for="sc-title" class="form-label">Title</label>
            <input id="sc-title" v-model="newTitle" type="text" class="form-input" required placeholder="e.g. Roman Imperial Highlights" />
          </div>
          <div class="form-group">
            <label for="sc-desc" class="form-label">Description (optional)</label>
            <textarea id="sc-desc" v-model="newDesc" rows="3" class="form-input" placeholder="A brief description of this showcase"></textarea>
          </div>
          <div class="mt-5 flex justify-end gap-2">
            <button type="button" class="btn btn-secondary" @click="showCreate = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="creating">
              {{ creating ? 'Creating...' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <div v-if="deleteTarget" class="fixed inset-0 z-[100] flex items-center justify-center bg-black/60 px-4 py-6" @click.self="deleteTarget = null">
      <div class="card w-full max-w-[480px] p-6">
        <h2 class="mb-3 mt-0 text-lg text-text-primary">Delete Showcase</h2>
        <p class="m-0 text-base text-text-secondary">Are you sure you want to delete "{{ deleteTarget.title }}"? This cannot be undone.</p>
        <div class="mt-5 flex justify-end gap-2">
          <button class="btn btn-secondary" @click="deleteTarget = null">Cancel</button>
          <button class="btn btn-danger" :disabled="deleting" @click="handleDelete">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { CirclePlus, Plus, Pencil, Trash2, Link, Eye, EyeOff, X, Coins, Presentation } from 'lucide-vue-next'
import { listShowcases, createShowcase, updateShowcase, deleteShowcase } from '@/api/client'
import { usePwa } from '@/composables/usePwa'

interface Showcase {
  id: number
  slug: string
  title: string
  description?: string
  isActive: boolean
  coinCount: number
  createdAt: string
  updatedAt: string
}

const loading = ref(true)
const { isPwa } = usePwa()
const showcases = ref<Showcase[]>([])
const showCreate = ref(false)
const newTitle = ref('')
const newDesc = ref('')
const creating = ref(false)
const deleteTarget = ref<Showcase | null>(null)
const deleting = ref(false)
const copied = ref(false)

async function loadShowcases() {
  loading.value = true
  try {
    const res = await listShowcases()
    showcases.value = res.data?.showcases ?? []
  } catch {
    showcases.value = []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!newTitle.value.trim()) return
  creating.value = true
  try {
    await createShowcase({ title: newTitle.value.trim(), description: newDesc.value.trim() || undefined })
    showCreate.value = false
    newTitle.value = ''
    newDesc.value = ''
    await loadShowcases()
  } finally {
    creating.value = false
  }
}

async function toggleActive(sc: Showcase) {
  try {
    await updateShowcase(sc.id, { isActive: !sc.isActive })
    sc.isActive = !sc.isActive
  } catch {
    // silently fail
  }
}

function confirmDelete(sc: Showcase) {
  deleteTarget.value = sc
}

async function handleDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await deleteShowcase(deleteTarget.value.id)
    deleteTarget.value = null
    await loadShowcases()
  } finally {
    deleting.value = false
  }
}

function copyLink(slug: string) {
  const url = `${window.location.origin}/s/${slug}`
  navigator.clipboard.writeText(url)
  copied.value = true
  setTimeout(() => { copied.value = false }, 2000)
}

onMounted(loadShowcases)
</script>
