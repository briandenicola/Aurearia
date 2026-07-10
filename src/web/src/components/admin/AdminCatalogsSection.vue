<template>
  <section class="card p-6">
    <h2 class="text-xl font-medium text-heading">Catalog Registry</h2>
    <p class="mb-6 text-body leading-6 text-text-secondary">
      Manage coin catalog codes used in structured references. Deleting a catalog is only allowed if no coins reference it.
    </p>

    <div v-if="loading" class="flex items-center justify-center p-12">
      <div class="spinner"></div>
    </div>

    <div
      v-else-if="error"
      class="flex items-center gap-2 rounded-sm border border-[color-mix(in_srgb,var(--color-negative)_30%,transparent)] bg-[color-mix(in_srgb,var(--color-negative)_10%,transparent)] p-4 text-body text-[var(--color-negative)]"
      role="alert"
    >
      <AlertCircle :size="20" />
      <span>{{ error }}</span>
    </div>

    <template v-else>
      <div class="mb-4 flex justify-end">
        <button class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="openCreateForm">
          + Add Catalog
        </button>
      </div>

      <div v-if="catalogs.length === 0" class="py-8 text-center text-body text-text-muted">
        No catalogs defined yet.
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full border-collapse text-body">
          <thead>
            <tr>
              <th class="border-b border-border-subtle px-2 py-3 text-left text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Code</th>
              <th class="border-b border-border-subtle px-2 py-3 text-left text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Display Name</th>
              <th class="hidden border-b border-border-subtle px-2 py-3 text-left text-label font-semibold uppercase tracking-[0.08em] text-text-muted md:table-cell">Volume Required</th>
              <th class="border-b border-border-subtle px-2 py-3 text-left text-label font-semibold uppercase tracking-[0.08em] text-text-muted">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="cat in catalogs" :key="cat.id" class="last:[&>td]:border-b-0">
              <td class="border-b border-border-subtle px-2 py-3 align-top">
                <div class="flex flex-col items-start gap-[0.35rem]">
                  <div class="font-semibold text-gold">{{ cat.catalog }}</div>
                  <span
                    class="chip-sm capitalize"
                    :class="cat.era === 'ancient'
                      ? 'border-[rgba(155,89,182,0.4)] bg-[rgba(155,89,182,0.2)] text-[#bb8fce]'
                      : cat.era === 'medieval'
                        ? 'border-[rgba(107,142,35,0.4)] bg-[rgba(107,142,35,0.2)] text-[#9ccc65]'
                        : 'border-[rgba(70,130,180,0.4)] bg-[rgba(70,130,180,0.2)] text-[#7eb9e0]'"
                  >
                    {{ cat.era }}
                  </span>
                </div>
              </td>
              <td class="border-b border-border-subtle px-2 py-3 align-top text-text-primary">{{ cat.displayName }}</td>
              <td class="hidden border-b border-border-subtle px-2 py-3 align-top md:table-cell">
                <label class="relative inline-block h-7 w-[50px] shrink-0" aria-label="Volume required">
                  <input
                    type="checkbox"
                    :checked="cat.volumeRequired"
                    disabled
                    class="peer sr-only"
                  />
                  <span class="absolute inset-0 rounded-full border border-border-subtle bg-input transition-colors peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] peer-disabled:cursor-not-allowed peer-disabled:opacity-60 after:absolute after:bottom-[2px] after:left-[2px] after:h-[22px] after:w-[22px] after:rounded-full after:bg-text-secondary after:content-[''] after:transition-transform peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"></span>
                </label>
              </td>
              <td class="border-b border-border-subtle px-2 py-3 align-top">
                <div class="flex flex-wrap justify-end gap-[0.35rem]">
                  <button class="btn btn-ghost btn-xs focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="openEditForm(cat)">
                    Edit
                  </button>
                  <button class="btn btn-danger btn-xs focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="handleDelete(cat.id)">
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="showForm" class="fixed inset-0 z-[200] flex items-center justify-center bg-[rgba(0,0,0,0.6)] p-4" @click.self="closeForm">
        <div class="max-h-[90vh] w-full max-w-[500px] overflow-auto rounded-md border border-border-subtle bg-card shadow-[var(--shadow-card)]">
          <div class="flex items-center justify-between gap-4 border-b border-border-subtle px-6 py-4">
            <h3 class="m-0 text-lg font-medium text-heading">{{ editingCatalog ? 'Edit Catalog' : 'Add Catalog' }}</h3>
            <button
              class="inline-flex h-[30px] w-[30px] items-center justify-center p-0 text-text-muted transition-colors hover:text-text-primary focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
              @click="closeForm"
            >
              ×
            </button>
          </div>
          <form class="p-6" @submit.prevent="saveForm">
            <div class="form-group">
              <label class="form-label">Catalog Code<span class="ml-[0.15rem] text-[var(--color-negative)]">*</span></label>
              <input
                v-model.trim="formData.catalog"
                class="form-input"
                placeholder="e.g. RIC, BMC, SNG"
                required
                :disabled="!!editingCatalog"
              />
            </div>
            <div class="form-group">
              <label class="form-label">Display Name<span class="ml-[0.15rem] text-[var(--color-negative)]">*</span></label>
              <input
                v-model.trim="formData.displayName"
                class="form-input"
                placeholder="e.g. Roman Imperial Coinage"
                required
              />
            </div>
            <div class="form-group">
              <label class="form-label">Era<span class="ml-[0.15rem] text-[var(--color-negative)]">*</span></label>
              <select v-model="formData.era" class="form-input" required>
                <option value="" disabled>Select era</option>
                <option value="ancient">Ancient</option>
                <option value="medieval">Medieval</option>
                <option value="modern">Modern</option>
              </select>
            </div>
            <div class="form-group flex items-center justify-between gap-4">
              <label class="form-label mb-0">Volume Required</label>
              <label class="relative inline-block h-7 w-[50px] shrink-0 rounded-full focus-within:outline-2 focus-within:outline-gold focus-within:outline-offset-2">
                <input
                  v-model="formData.volumeRequired"
                  type="checkbox"
                  class="peer sr-only"
                />
                <span class="absolute inset-0 rounded-full border border-border-subtle bg-input transition-colors peer-checked:border-gold peer-checked:bg-[var(--accent-gold-dim)] after:absolute after:bottom-[2px] after:left-[2px] after:h-[22px] after:w-[22px] after:rounded-full after:bg-text-secondary after:content-[''] after:transition-transform peer-checked:after:translate-x-[22px] peer-checked:after:bg-gold"></span>
              </label>
            </div>
            <div class="mt-6 flex flex-col gap-2 md:flex-row md:justify-end">
              <button type="button" class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" @click="closeForm">
                Cancel
              </button>
              <button type="submit" class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2" :disabled="saving">
                {{ saving ? 'Saving...' : 'Save' }}
              </button>
            </div>
            <div
              v-if="formError"
              class="mt-4 flex items-center gap-2 rounded-sm border border-[color-mix(in_srgb,var(--color-negative)_30%,transparent)] bg-[color-mix(in_srgb,var(--color-negative)_10%,transparent)] p-3 text-sm text-[var(--color-negative)]"
              role="alert"
            >
              <AlertCircle :size="16" />
              <span>{{ formError }}</span>
            </div>
          </form>
        </div>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { AlertCircle } from 'lucide-vue-next'
import { listCatalogs, adminCreateCatalog, adminUpdateCatalog, adminDeleteCatalog } from '@/api/client'
import type { CatalogRegistry } from '@/types'
import { useDialog } from '@/composables/useDialog'

const { showAlert, showConfirm } = useDialog()

const catalogs = ref<CatalogRegistry[]>([])
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const showForm = ref(false)
const editingCatalog = ref<CatalogRegistry | null>(null)
const formError = ref('')
const formData = ref({
  catalog: '',
  displayName: '',
  era: '',
  volumeRequired: false,
})

async function loadCatalogs() {
  loading.value = true
  error.value = ''
  try {
    catalogs.value = await listCatalogs()
  } catch (_e) {
    error.value = 'Failed to load catalogs'
  } finally {
    loading.value = false
  }
}

function openCreateForm() {
  editingCatalog.value = null
  formData.value = {
    catalog: '',
    displayName: '',
    era: '',
    volumeRequired: false,
  }
  formError.value = ''
  showForm.value = true
}

function openEditForm(catalog: CatalogRegistry) {
  editingCatalog.value = catalog
  formData.value = {
    catalog: catalog.catalog,
    displayName: catalog.displayName,
    era: catalog.era,
    volumeRequired: catalog.volumeRequired,
  }
  formError.value = ''
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editingCatalog.value = null
  formData.value = {
    catalog: '',
    displayName: '',
    era: '',
    volumeRequired: false,
  }
  formError.value = ''
}

async function saveForm() {
  if (!formData.value.catalog || !formData.value.displayName || !formData.value.era) {
    formError.value = 'All required fields must be filled.'
    return
  }

  saving.value = true
  formError.value = ''
  try {
    const payload = {
      catalog: formData.value.catalog,
      displayName: formData.value.displayName,
      era: formData.value.era,
      volumeRequired: formData.value.volumeRequired,
    }

    if (editingCatalog.value) {
      await adminUpdateCatalog(editingCatalog.value.id, payload)
    } else {
      await adminCreateCatalog(payload)
    }

    await loadCatalogs()
    closeForm()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { error?: string } } }
    formError.value = err.response?.data?.error || 'Failed to save catalog'
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  const confirmed = await showConfirm(
    'Are you sure you want to delete this catalog? This will fail if any coins reference it.',
    { title: 'Delete Catalog', variant: 'danger' },
  )
  if (!confirmed) return

  try {
    await adminDeleteCatalog(id)
    await loadCatalogs()
  } catch (e: unknown) {
    const err = e as { response?: { status?: number; data?: { error?: string } } }
    if (err.response?.status === 409) {
      await showAlert(
        'This catalog is in use by one or more coins and cannot be deleted.',
        { title: 'Cannot Delete' },
      )
    } else {
      await showAlert(
        err.response?.data?.error || 'Failed to delete catalog',
        { title: 'Delete Failed' },
      )
    }
  }
}

onMounted(() => {
  loadCatalogs()
})
</script>
