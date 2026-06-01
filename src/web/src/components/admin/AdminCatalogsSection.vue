<template>
  <section class="admin-section card">
    <h2>Catalog Registry</h2>
    <p class="section-description">
      Manage coin catalog codes used in structured references. Deleting a catalog is only allowed if no coins reference it.
    </p>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner"></div>
    </div>

    <div v-else-if="error" class="error-message">
      <AlertCircle :size="20" />
      <span>{{ error }}</span>
    </div>

    <template v-else>
      <div class="actions-row">
        <button class="btn btn-primary btn-sm" @click="openCreateForm">
          + Add Catalog
        </button>
      </div>

      <div v-if="catalogs.length === 0" class="logs-empty">
        No catalogs defined yet.
      </div>

      <table v-else class="catalog-table">
        <thead>
          <tr>
            <th>Code</th>
            <th>Display Name</th>
            <th class="hide-mobile">Volume Required</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="cat in catalogs" :key="cat.id">
            <td>
              <div class="catalog-code-cell">
                <div class="catalog-code">{{ cat.catalog }}</div>
                <span class="chip-sm era-badge" :class="`era-${cat.era}`">{{ cat.era }}</span>
              </div>
            </td>
            <td>{{ cat.displayName }}</td>
            <td class="hide-mobile">
              <label class="toggle">
                <input
                  type="checkbox"
                  :checked="cat.volumeRequired"
                  disabled
                />
                <span class="toggle-slider"></span>
              </label>
            </td>
            <td>
              <div class="action-buttons">
                <button class="btn btn-ghost btn-xs" @click="openEditForm(cat)">
                  Edit
                </button>
                <button class="btn btn-danger btn-xs" @click="handleDelete(cat.id)">
                  Delete
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Create/Edit Modal -->
      <div v-if="showForm" class="modal-overlay" @click.self="closeForm">
        <div class="modal-content">
          <div class="modal-header">
            <h3>{{ editingCatalog ? 'Edit Catalog' : 'Add Catalog' }}</h3>
            <button class="modal-close" @click="closeForm">×</button>
          </div>
          <form class="modal-body" @submit.prevent="saveForm">
            <div class="form-group">
              <label class="form-label">Catalog Code<span class="required">*</span></label>
              <input
                v-model.trim="formData.catalog"
                class="form-input"
                placeholder="e.g. RIC, BMC, SNG"
                required
                :disabled="!!editingCatalog"
              />
            </div>
            <div class="form-group">
              <label class="form-label">Display Name<span class="required">*</span></label>
              <input
                v-model.trim="formData.displayName"
                class="form-input"
                placeholder="e.g. Roman Imperial Coinage"
                required
              />
            </div>
            <div class="form-group">
              <label class="form-label">Era<span class="required">*</span></label>
              <select v-model="formData.era" class="form-input" required>
                <option value="" disabled>Select era</option>
                <option value="ancient">Ancient</option>
                <option value="medieval">Medieval</option>
                <option value="modern">Modern</option>
              </select>
            </div>
            <div class="form-group toggle-row">
              <label class="form-label">Volume Required</label>
              <label class="toggle">
                <input
                  v-model="formData.volumeRequired"
                  type="checkbox"
                />
                <span class="toggle-slider"></span>
              </label>
            </div>
            <div class="modal-footer">
              <button type="button" class="btn btn-secondary btn-sm" @click="closeForm">
                Cancel
              </button>
              <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">
                {{ saving ? 'Saving...' : 'Save' }}
              </button>
            </div>
            <div v-if="formError" class="form-error">
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

<style scoped>
.admin-section {
  padding: 1.5rem;
}

.section-description {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-bottom: 1.5rem;
  line-height: 1.5;
}

.loading-overlay {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 3rem;
}

.spinner {
  border: 3px solid var(--border-subtle);
  border-top-color: var(--accent-gold);
  border-radius: 50%;
  width: 40px;
  height: 40px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-message {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 1rem;
  background: rgba(231, 76, 60, 0.1);
  border: 1px solid rgba(231, 76, 60, 0.3);
  border-radius: var(--radius-sm);
  color: #e74c3c;
  font-size: 0.85rem;
}

.actions-row {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 1rem;
}

.logs-empty {
  text-align: center;
  padding: 2rem;
  color: var(--text-muted);
  font-size: 0.85rem;
}

.catalog-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.catalog-table th,
.catalog-table td {
  text-align: left;
  padding: 0.75rem 0.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.catalog-table th {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-muted);
  font-weight: 600;
}

.catalog-code-cell {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  align-items: flex-start;
}

.catalog-code {
  font-weight: 600;
  color: var(--accent-gold);
}

.era-badge {
  text-transform: capitalize;
}

.era-ancient {
  background: rgba(155, 89, 182, 0.2);
  border-color: rgba(155, 89, 182, 0.4);
  color: #bb8fce;
}

.era-medieval {
  background: rgba(107, 142, 35, 0.2);
  border-color: rgba(107, 142, 35, 0.4);
  color: #9ccc65;
}

.era-modern {
  background: rgba(70, 130, 180, 0.2);
  border-color: rgba(70, 130, 180, 0.4);
  color: #7eb9e0;
}

.toggle {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 28px;
  flex-shrink: 0;
}

.toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-slider {
  position: absolute;
  cursor: pointer;
  inset: 0;
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  border-radius: 28px;
  transition: background var(--transition-fast);
}

.toggle-slider::before {
  content: '';
  position: absolute;
  width: 22px;
  height: 22px;
  left: 2px;
  bottom: 2px;
  background: var(--text-secondary);
  border-radius: 50%;
  transition: transform var(--transition-fast);
}

.toggle input:checked + .toggle-slider {
  background: var(--accent-gold-dim);
  border-color: var(--accent-gold);
}

.toggle input:checked + .toggle-slider::before {
  transform: translateX(22px);
  background: var(--accent-gold);
}

.toggle input:disabled + .toggle-slider {
  cursor: not-allowed;
  opacity: 0.6;
}

.action-buttons {
  display: flex;
  gap: 0.35rem;
  flex-shrink: 0;
  justify-content: flex-end;
}

.hide-mobile {
  display: table-cell;
}

@media (max-width: 768px) {
  .hide-mobile {
    display: none;
  }
}

/* Modal styles */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  max-width: 500px;
  width: 100%;
  max-height: 90vh;
  overflow: auto;
  box-shadow: var(--shadow-card);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-header h3 {
  margin: 0;
  font-size: 1.2rem;
  color: var(--text-heading);
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: 1.5rem;
  cursor: pointer;
  padding: 0;
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color var(--transition-fast);
}

.modal-close:hover {
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-label {
  display: block;
  font-size: 0.85rem;
  font-weight: 500;
  color: var(--text-primary);
  margin-bottom: 0.35rem;
}

.required {
  color: #e74c3c;
  margin-left: 0.15rem;
}

.toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.modal-footer {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
  margin-top: 1.5rem;
}

.form-error {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 0.75rem;
  background: rgba(231, 76, 60, 0.1);
  border: 1px solid rgba(231, 76, 60, 0.3);
  border-radius: var(--radius-sm);
  color: #e74c3c;
  font-size: 0.8rem;
}
</style>
