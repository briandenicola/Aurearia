<template>
  <section class="references-section">
    <div class="references-header">
      <h3>Catalog References</h3>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="startCreate">
        + Add Reference
      </button>
    </div>

    <div v-if="!rows.length && !editing" class="section-content-card references-empty">
      No structured references added yet.
    </div>

    <div v-else class="references-list">
      <article
        v-for="ref in rows"
        :key="ref.id"
        class="reference-card"
      >
        <template v-if="editing?.mode === 'edit' && editing.id === ref.id">
          <form class="reference-form" @submit.prevent="saveEdit">
            <div class="reference-grid">
              <select v-model="draft.catalog" class="form-input">
                <option value="" disabled>Select catalog</option>
                <option v-for="opt in editingCatalogOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
              <input v-model.trim="draft.volume" class="form-input" placeholder="Volume (optional)" />
              <input v-model.trim="draft.number" class="form-input" placeholder="Number" />
            </div>
            <div class="reference-grid reference-grid-two">
              <input v-model.trim="draft.invoiceNumber" class="form-input" placeholder="Invoice Number (optional)" />
              <input v-model.trim="draft.uri" class="form-input" placeholder="URI (optional)" />
            </div>
            <div class="reference-actions">
              <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">Save</button>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="cancelEdit">Cancel</button>
            </div>
          </form>
        </template>

        <template v-else>
          <div class="reference-main">
            <span class="chip-sm reference-catalog">{{ ref.catalog }}</span>
            <span class="reference-value">
              <template v-if="ref.volume">{{ ref.volume }} </template>{{ ref.number }}
            </span>
            <span v-if="ref.invoiceNumber" class="reference-invoice">{{ ref.invoiceNumber }}</span>
          </div>
          <div class="reference-actions">
            <a
              v-if="ref.uri"
              :href="ref.uri"
              target="_blank"
              rel="noopener noreferrer"
              class="btn btn-ghost btn-xs"
            >
              Open
            </a>
            <button type="button" class="btn btn-ghost btn-xs" :disabled="saving" @click="startEdit(ref)">
              Edit
            </button>
            <button type="button" class="btn btn-danger btn-xs" :disabled="saving" @click="removeReference(ref)">
              Delete
            </button>
          </div>
        </template>
      </article>

      <article v-if="editing?.mode === 'create'" class="reference-card">
        <form class="reference-form" @submit.prevent="saveCreate">
          <div class="reference-grid">
            <select v-model="draft.catalog" class="form-input">
              <option value="" disabled>Select catalog</option>
              <option v-for="c in catalogs" :key="c.id" :value="c.catalog">{{ c.catalog }} — {{ c.displayName }}</option>
            </select>
            <input v-model.trim="draft.volume" class="form-input" placeholder="Volume (optional)" />
            <input v-model.trim="draft.number" class="form-input" placeholder="Number" />
          </div>
          <div class="reference-grid reference-grid-two">
            <input v-model.trim="draft.invoiceNumber" class="form-input" placeholder="Invoice Number (optional)" />
            <input v-model.trim="draft.uri" class="form-input" placeholder="URI (optional)" />
          </div>
          <div class="reference-actions">
            <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">Save</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="cancelEdit">Cancel</button>
          </div>
        </form>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { createCoinReference, deleteCoinReference, updateCoinReference, listCatalogs } from '@/api/client'
import type { CoinReference, CatalogRegistry } from '@/types'
import { useDialog } from '@/composables/useDialog'

type ReferenceDraft = {
  catalog: string
  volume: string
  number: string
  invoiceNumber: string
  uri: string
}

const props = defineProps<{
  coinId: number
  references: CoinReference[]
}>()

const emit = defineEmits<{
  changed: []
}>()

const { showAlert, showConfirm } = useDialog()

const localReferences = ref<CoinReference[]>([])
const catalogs = ref<CatalogRegistry[]>([])
const saving = ref(false)
const editing = ref<{ mode: 'create' } | { mode: 'edit'; id: number } | null>(null)
const draft = ref<ReferenceDraft>({
  catalog: '',
  volume: '',
  number: '',
  invoiceNumber: '',
  uri: '',
})

const rows = computed(() =>
  [...localReferences.value].sort((a, b) => {
    if (a.catalog !== b.catalog) return a.catalog.localeCompare(b.catalog)
    return a.number.localeCompare(b.number)
  }),
)

const editingCatalogOptions = computed(() => {
  const opts = catalogs.value.map(c => ({ value: c.catalog, label: `${c.catalog} — ${c.displayName}` }))
  if (draft.value.catalog && !catalogs.value.some(c => c.catalog === draft.value.catalog)) {
    opts.push({ value: draft.value.catalog, label: `${draft.value.catalog} (legacy)` })
  }
  return opts
})

onMounted(async () => {
  try {
    catalogs.value = await listCatalogs()
  } catch {
    // ignore catalog load failure
  }
})

watch(
  () => props.references,
  (next) => {
    localReferences.value = (next ?? []).map((item) => ({ ...item }))
    if (editing.value?.mode === 'edit') {
      const editID = editing.value.id
      const stillExists = localReferences.value.some((item) => item.id === editID)
      if (!stillExists) editing.value = null
    }
  },
  { immediate: true, deep: true },
)

function resetDraft(value?: Partial<CoinReference>) {
  draft.value = {
    catalog: value?.catalog ?? '',
    volume: value?.volume ?? '',
    number: value?.number ?? '',
    invoiceNumber: value?.invoiceNumber ?? '',
    uri: value?.uri ?? '',
  }
}

function startCreate() {
  editing.value = { mode: 'create' }
  resetDraft()
}

function startEdit(reference: CoinReference) {
  editing.value = { mode: 'edit', id: reference.id }
  resetDraft(reference)
}

function cancelEdit() {
  editing.value = null
  resetDraft()
}

async function saveCreate() {
  if (!draft.value.catalog || !draft.value.number) {
    await showAlert('Catalog and number are required.', { title: 'Missing Data' })
    return
  }

  saving.value = true
  try {
    const created = await createCoinReference(props.coinId, draft.value)
    localReferences.value = [...localReferences.value, created.data]
    editing.value = null
    emit('changed')
  } catch (error) {
    await showAlert(getErrorMessage(error), { title: 'Failed to add reference' })
  } finally {
    saving.value = false
  }
}

async function saveEdit() {
  if (editing.value?.mode !== 'edit') return
  if (!draft.value.catalog || !draft.value.number) {
    await showAlert('Catalog and number are required.', { title: 'Missing Data' })
    return
  }

  saving.value = true
  try {
    const editID = editing.value.id
    const updated = await updateCoinReference(props.coinId, editID, draft.value)
    localReferences.value = localReferences.value.map((item) =>
      item.id === editID ? updated.data : item,
    )
    editing.value = null
    emit('changed')
  } catch (error) {
    await showAlert(getErrorMessage(error), { title: 'Failed to update reference' })
  } finally {
    saving.value = false
  }
}

async function removeReference(reference: CoinReference) {
  const confirmed = await showConfirm(
    `Delete ${reference.catalog} ${reference.volume ? `${reference.volume} ` : ''}${reference.number}?`,
    { title: 'Delete Reference', variant: 'danger' },
  )
  if (!confirmed) return

  saving.value = true
  try {
    await deleteCoinReference(props.coinId, reference.id)
    localReferences.value = localReferences.value.filter((item) => item.id !== reference.id)
    emit('changed')
  } catch (error) {
    await showAlert(getErrorMessage(error), { title: 'Failed to delete reference' })
  } finally {
    saving.value = false
  }
}

function getErrorMessage(error: unknown): string {
  const err = error as { response?: { data?: { error?: string } } }
  return err.response?.data?.error || 'Request failed.'
}
</script>

<style scoped>
.references-section {
  margin-bottom: 1.5rem;
}

.references-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.references-header h3 {
  margin: 0;
  font-size: 1rem;
}

.references-empty {
  color: var(--text-secondary);
  font-size: 0.85rem;
}

.references-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.reference-card {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-card);
  padding: 0.75rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.reference-main {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.reference-catalog {
  color: var(--accent-gold);
  border: 1px solid var(--border-accent);
}

.reference-value {
  font-size: 0.9rem;
  color: var(--text-primary);
}

.reference-invoice {
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.reference-actions {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  flex-wrap: wrap;
}

.reference-form {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.reference-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.5rem;
}

.reference-grid.reference-grid-two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

@media (max-width: 768px) {
  .reference-grid,
  .reference-grid.reference-grid-two {
    grid-template-columns: 1fr;
  }
}
</style>
