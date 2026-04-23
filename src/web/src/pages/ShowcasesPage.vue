<template>
  <div class="container">
    <div class="page-header">
      <h1>Showcases</h1>
      <button class="btn btn-primary" @click="showCreate = true">
        <Plus :size="16" /> New Showcase
      </button>
    </div>

    <div v-if="loading" class="loading-state">Loading showcases...</div>

    <div v-else-if="!showcases.length" class="empty-state">
      <Presentation :size="48" />
      <h3>No showcases yet</h3>
      <p>Create a showcase to share a curated selection of your coins with the world.</p>
      <button class="btn btn-primary" style="margin-top: 1rem" @click="showCreate = true">
        <Plus :size="16" /> Create Your First Showcase
      </button>
    </div>

    <div v-else class="showcases-grid">
      <div v-for="sc in showcases" :key="sc.id" class="card showcase-card">
        <div class="showcase-header">
          <h3 class="showcase-title">{{ sc.title }}</h3>
          <span class="badge" :class="sc.isActive ? 'badge-active' : 'badge-inactive'">
            {{ sc.isActive ? 'Active' : 'Inactive' }}
          </span>
        </div>
        <p v-if="sc.description" class="showcase-desc">{{ sc.description }}</p>
        <div class="showcase-meta">
          <span class="coin-count"><Coins :size="14" /> {{ sc.coinCount ?? 0 }} coins</span>
          <span class="showcase-slug">/s/{{ sc.slug }}</span>
        </div>
        <div class="showcase-actions">
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

    <div v-if="copied" class="toast">Link copied to clipboard</div>

    <!-- Create Modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal card">
        <div class="modal-header">
          <h2>New Showcase</h2>
          <button class="btn-close" @click="showCreate = false"><X :size="18" /></button>
        </div>
        <form @submit.prevent="handleCreate">
          <div class="form-group">
            <label for="sc-title">Title</label>
            <input id="sc-title" v-model="newTitle" type="text" required placeholder="e.g. Roman Imperial Highlights" />
          </div>
          <div class="form-group">
            <label for="sc-desc">Description (optional)</label>
            <textarea id="sc-desc" v-model="newDesc" rows="3" placeholder="A brief description of this showcase"></textarea>
          </div>
          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="showCreate = false">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="creating">
              {{ creating ? 'Creating...' : 'Create' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="deleteTarget" class="modal-overlay" @click.self="deleteTarget = null">
      <div class="modal card">
        <h2>Delete Showcase</h2>
        <p>Are you sure you want to delete "{{ deleteTarget.title }}"? This cannot be undone.</p>
        <div class="modal-actions">
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
import { Plus, Pencil, Trash2, Link, Eye, EyeOff, X, Coins, Presentation } from 'lucide-vue-next'
import { listShowcases, createShowcase, updateShowcase, deleteShowcase } from '@/api/client'

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

<style scoped>
.container { max-width: 1200px; margin: 0 auto; padding: 1.5rem; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1.5rem; }
.page-header h1 { font-size: 1.75rem; color: var(--text-primary); }
.btn { display: inline-flex; align-items: center; gap: 0.35rem; padding: 0.5rem 1rem; border-radius: 8px; border: none; cursor: pointer; font-weight: 500; font-size: 0.875rem; }
.btn-primary { background: var(--accent-gold); color: #1e1e1e; }
.btn-secondary { background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border-subtle); }
.btn-danger { background: #dc3545; color: white; }
.btn-sm { padding: 0.35rem 0.65rem; font-size: 0.8rem; }
.loading-state { text-align: center; padding: 2rem; color: var(--text-secondary); }
.empty-state { text-align: center; padding: 3rem; color: var(--text-secondary); }
.empty-state h3 { color: var(--text-primary); margin: 1rem 0 0.5rem; }

.showcases-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 1rem; }
.showcase-card { background: var(--bg-card); border: 1px solid var(--border-subtle); border-radius: 12px; padding: 1.25rem; display: flex; flex-direction: column; gap: 0.75rem; }
.showcase-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 0.5rem; }
.showcase-title { font-size: 1.1rem; color: var(--text-primary); margin: 0; }
.badge { font-size: 0.7rem; padding: 0.2rem 0.5rem; border-radius: 4px; font-weight: 600; text-transform: uppercase; white-space: nowrap; }
.badge-active { background: rgba(40, 167, 69, 0.15); color: #28a745; }
.badge-inactive { background: rgba(108, 117, 125, 0.15); color: #6c757d; }
.showcase-desc { color: var(--text-secondary); font-size: 0.875rem; margin: 0; line-height: 1.4; }
.showcase-meta { display: flex; justify-content: space-between; align-items: center; font-size: 0.8rem; color: var(--text-secondary); }
.coin-count { display: inline-flex; align-items: center; gap: 0.25rem; }
.showcase-slug { font-family: monospace; opacity: 0.7; }
.showcase-actions { display: flex; flex-wrap: wrap; gap: 0.5rem; margin-top: auto; }

.toast { position: fixed; bottom: 2rem; left: 50%; transform: translateX(-50%); background: var(--accent-gold); color: #1e1e1e; padding: 0.5rem 1.25rem; border-radius: 8px; font-weight: 500; font-size: 0.875rem; z-index: 1000; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.6); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal { width: 90%; max-width: 480px; padding: 1.5rem; }
.modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; }
.modal-header h2 { margin: 0; color: var(--text-primary); }
.modal h2 { color: var(--text-primary); margin: 0 0 0.75rem; }
.modal p { color: var(--text-secondary); }
.btn-close { background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0.25rem; }
.form-group { margin-bottom: 1rem; }
.form-group label { display: block; margin-bottom: 0.35rem; color: var(--text-secondary); font-size: 0.875rem; }
.form-group input,
.form-group textarea { width: 100%; background: var(--bg-card); color: var(--text-primary); border: 1px solid var(--border-subtle); border-radius: 8px; padding: 0.5rem 0.75rem; font-size: 0.875rem; box-sizing: border-box; }
.modal-actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 1.25rem; }
</style>
