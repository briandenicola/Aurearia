<template>
  <section class="mb-6">
    <div class="mb-3 flex items-center justify-between gap-3">
      <h3 class="m-0 text-base font-medium text-text-primary">Catalog References</h3>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="startCreate">
        + Add Reference
      </button>
    </div>

    <div v-if="!rows.length && !editing" class="section-content-card text-body text-text-secondary">
      No structured references added yet.
    </div>

    <div v-else class="flex flex-col gap-2">
      <article
        v-for="ref in rows"
        :key="ref.id"
        class="flex flex-wrap items-center justify-between gap-3 rounded-sm border border-border-subtle bg-card p-3"
      >
        <template v-if="editing?.mode === 'edit' && editing.id === ref.id">
          <form class="flex w-full flex-col gap-2" @submit.prevent="saveEdit">
            <div class="grid gap-2 md:grid-cols-3">
              <select v-model="draft.catalog" class="form-input">
                <option value="" disabled>Select catalog</option>
                <option v-for="opt in editingCatalogOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
              </select>
              <input v-model.trim="draft.volume" class="form-input" placeholder="Volume (optional)" />
              <input v-model.trim="draft.number" class="form-input" placeholder="Number" />
            </div>
            <div class="grid gap-2 md:grid-cols-2">
              <input v-model.trim="draft.invoiceNumber" class="form-input" placeholder="Invoice Number (optional)" />
              <input v-model.trim="draft.uri" class="form-input" placeholder="URI (optional)" />
            </div>
            <div class="inline-flex flex-wrap items-center gap-[0.35rem]">
              <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">Save</button>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="cancelEdit">Cancel</button>
            </div>
          </form>
        </template>

        <template v-else>
          <div class="inline-flex flex-wrap items-center gap-2">
            <span class="chip-sm border-border-accent text-gold">{{ ref.catalog }}</span>
            <span class="text-base text-text-primary">
              <template v-if="ref.volume">{{ ref.volume }} </template>{{ ref.number }}
            </span>
            <span v-if="ref.invoiceNumber" class="text-sm text-text-secondary">{{ ref.invoiceNumber }}</span>
          </div>
          <div class="inline-flex flex-wrap items-center gap-[0.35rem]">
            <SafeExternalLink
              v-if="ref.uri"
              :href="ref.uri"
              class="btn btn-ghost btn-xs"
            >
              Open
            </SafeExternalLink>
            <button type="button" class="btn btn-ghost btn-xs" :disabled="saving" @click="startEdit(ref)">
              Edit
            </button>
            <button type="button" class="btn btn-danger btn-xs" :disabled="saving" @click="removeReference(ref)">
              Delete
            </button>
          </div>
        </template>
      </article>

      <article v-if="editing?.mode === 'create'" class="rounded-sm border border-border-subtle bg-card p-3">
        <form class="flex w-full flex-col gap-2" @submit.prevent="saveCreate">
          <div class="grid gap-2 md:grid-cols-3">
            <select v-model="draft.catalog" class="form-input">
              <option value="" disabled>Select catalog</option>
              <option v-for="c in catalogs" :key="c.id" :value="c.catalog">{{ c.catalog }} — {{ c.displayName }}</option>
            </select>
            <input v-model.trim="draft.volume" class="form-input" placeholder="Volume (optional)" />
            <input v-model.trim="draft.number" class="form-input" placeholder="Number" />
          </div>
          <div class="grid gap-2 md:grid-cols-2">
            <input v-model.trim="draft.invoiceNumber" class="form-input" placeholder="Invoice Number (optional)" />
            <input v-model.trim="draft.uri" class="form-input" placeholder="URI (optional)" />
          </div>
          <div class="inline-flex flex-wrap items-center gap-[0.35rem]">
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
import SafeExternalLink from '@/components/SafeExternalLink.vue'

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
