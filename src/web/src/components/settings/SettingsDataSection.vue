<template>
  <section class="card">
    <h2 class="mb-5 border-b border-border-subtle pb-3 text-lg">Data Management</h2>
    <div class="mt-8 grid grid-cols-1 gap-6 md:grid-cols-2">
      <section class="min-w-0" aria-labelledby="tags-heading">
        <h3 id="tags-heading" class="mb-3 text-base font-medium text-text-secondary">Tags and Open Sets</h3>
        <p class="text-sm text-text-muted">Legacy tags remain supported. New open sets can be managed from the Sets page.</p>
        <router-link
          to="/sets"
          class="btn btn-secondary btn-sm mt-3 inline-flex focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
        >
          Open Sets
        </router-link>

        <div class="my-4 flex flex-wrap items-center gap-2">
          <input
            v-model="newTagName"
            type="text"
            class="form-input min-w-[150px] flex-1 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
            placeholder="New tag name..."
            maxlength="50"
            @keydown.enter="handleCreateTag"
          />
          <div class="flex items-center gap-[0.3rem]">
            <button
              v-for="c in TAG_COLORS"
              :key="c"
              class="h-[22px] w-[22px] cursor-pointer rounded-full border-2 border-transparent p-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
              :class="newTagColor === c ? 'border-text-primary ring-2 ring-[var(--bg-card)]' : ''"
              :style="{ backgroundColor: c }"
              @click="newTagColor = c"
            ></button>
          </div>
          <button
            class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
            @click="handleCreateTag"
            :disabled="!newTagName.trim()"
          >
            Create Tag
          </button>
        </div>
        <p v-if="tagError" class="mt-1 text-body text-[var(--cat-byzantine)]">{{ tagError }}</p>

        <div v-if="tagList.length" class="mt-4 flex flex-col gap-2">
          <div v-for="tag in tagList" :key="tag.id" class="flex flex-wrap items-center gap-2 rounded-sm border border-border-subtle p-2">
            <template v-if="editingTag?.id === tag.id">
              <input
                v-model="editTagName"
                class="form-input min-w-[120px] flex-1 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                maxlength="50"
                @keydown.enter="handleSaveTag"
              />
              <div class="flex items-center gap-[0.3rem]">
                <button
                  v-for="c in TAG_COLORS"
                  :key="c"
                  class="h-[18px] w-[18px] cursor-pointer rounded-full border-2 border-transparent p-0 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                  :class="editTagColor === c ? 'border-text-primary ring-2 ring-[var(--bg-card)]' : ''"
                  :style="{ backgroundColor: c }"
                  @click="editTagColor = c"
                ></button>
              </div>
              <button
                class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                @click="handleSaveTag"
              >
                Save
              </button>
              <button
                class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                @click="editingTag = null"
              >
                Cancel
              </button>
            </template>
            <template v-else>
              <span
                class="shrink-0 rounded-full border px-[0.6rem] py-[0.2rem] text-chip"
                :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }"
              >
                {{ tag.name }}
              </span>
              <div class="ml-auto flex gap-1">
                <button
                  class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                  @click="startEditTag(tag)"
                >
                  Edit
                </button>
                <button
                  class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                  @click="handleDeleteTag(tag)"
                >
                  Delete
                </button>
              </div>
            </template>
          </div>
        </div>
        <p v-else class="mt-4 text-body text-text-secondary">No tags created yet. Create your first tag above.</p>
      </section>

      <section class="min-w-0" aria-labelledby="storage-locations-heading">
        <h3 id="storage-locations-heading" class="mb-3 text-base font-medium text-text-secondary">Storage Locations</h3>
        <p class="text-sm text-text-muted">Create shelf, tray, safe, or box locations for the coin form dropdown.</p>

        <div class="my-4 flex flex-wrap items-center gap-2">
          <input
            v-model="newStorageLocationName"
            type="text"
            class="form-input min-w-[150px] flex-1 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
            placeholder="New storage location..."
            maxlength="100"
            :disabled="storageLocationSaving"
            @keydown.enter="handleCreateStorageLocation"
          />
          <button
            class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
            @click="handleCreateStorageLocation"
            :disabled="!newStorageLocationName.trim() || storageLocationSaving"
          >
            {{ storageLocationSaving ? 'Saving...' : 'Create Location' }}
          </button>
        </div>
        <p v-if="storageLocationError" class="mt-1 text-body text-[var(--cat-byzantine)]">{{ storageLocationError }}</p>
        <p v-if="storageLocationsLoading" class="mt-4 text-body text-text-secondary">Loading storage locations...</p>

        <div v-else-if="storageLocationList.length" class="mt-4 flex flex-col gap-2">
          <div v-for="location in storageLocationList" :key="location.id" class="flex flex-wrap items-center gap-2 rounded-sm border border-border-subtle p-2">
            <template v-if="editingStorageLocation?.id === location.id">
              <input
                v-model="editStorageLocationName"
                class="form-input min-w-[120px] flex-1 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                maxlength="100"
                @keydown.enter="handleSaveStorageLocation"
              />
              <button
                class="btn btn-primary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                @click="handleSaveStorageLocation"
                :disabled="storageLocationSaving"
              >
                Save
              </button>
              <button
                class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                @click="editingStorageLocation = null"
                :disabled="storageLocationSaving"
              >
                Cancel
              </button>
            </template>
            <template v-else>
              <span class="chip-sm shrink-0 bg-input text-text-primary">{{ location.name }}</span>
              <div class="ml-auto flex gap-1">
                <button
                  class="btn btn-secondary btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                  @click="startEditStorageLocation(location)"
                >
                  Edit
                </button>
                <button
                  class="btn btn-danger btn-sm focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
                  :disabled="deletingStorageLocationId === location.id"
                  @click="handleDeleteStorageLocation(location)"
                >
                  {{ deletingStorageLocationId === location.id ? 'Deleting...' : 'Delete' }}
                </button>
              </div>
            </template>
          </div>
        </div>
        <p v-else class="mt-4 text-body text-text-secondary">No storage locations created yet. Create your first location above.</p>
      </section>
    </div>

    <section class="mt-8 border-t border-border-subtle pt-8" aria-labelledby="migration-heading">
      <div class="mb-2 flex items-center gap-2">
        <Database :size="20" />
        <h3 id="migration-heading" class="m-0 text-base font-medium text-text-secondary">Catalog Reference Migration</h3>
      </div>
      <p class="text-sm text-text-muted">
        Convert legacy free-text Rarity/RIC values into structured Catalog References.
        This is non-destructive (originals are kept) and records outcomes in each coin's journal.
      </p>
      
      <button
        class="btn btn-primary mt-4 inline-flex items-center gap-2 focus-visible:outline-2 focus-visible:outline-gold focus-visible:outline-offset-2"
        :disabled="migrationRunning"
        @click="handleMigrate"
      >
        <RefreshCw :size="16" :class="migrationRunning ? 'animate-spin' : ''" />
        {{ migrationRunning ? 'Migrating...' : 'Run Migration' }}
      </button>

      <div v-if="migrationResult" class="mt-6 rounded-sm border border-border-subtle bg-input p-4">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-3 md:gap-4">
          <div class="flex flex-col gap-[0.35rem] text-center">
            <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">SUCCEEDED</span>
            <span class="text-xl font-semibold text-gold">{{ migrationResult.succeeded }}</span>
          </div>
          <div class="flex flex-col gap-[0.35rem] text-center">
            <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">SKIPPED</span>
            <span class="text-xl font-semibold text-text-secondary">{{ migrationResult.skipped }}</span>
          </div>
          <div class="flex flex-col gap-[0.35rem] text-center">
            <span class="text-label font-semibold uppercase tracking-[0.08em] text-text-muted">FAILED</span>
            <span class="text-xl font-semibold text-warning">{{ migrationResult.failed }}</span>
          </div>
        </div>
        <p v-if="migrationResult.message" class="mt-3 border-t border-border-subtle pt-3 text-center text-body text-text-secondary">
          {{ migrationResult.message }}
        </p>
      </div>

      <p v-if="migrationError" class="mt-1 text-body text-[var(--cat-byzantine)]">{{ migrationError }}</p>
    </section>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Database, RefreshCw } from 'lucide-vue-next'
import {
  getTags, createTag, updateTag as updateTagApi, deleteTag,
  getStorageLocations, createStorageLocation, updateStorageLocation, deleteStorageLocation,
  migrateLegacyReferences,
} from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import type { Tag, StorageLocation, LegacyMigrationResult } from '@/types'

const { showConfirm } = useDialog()
// Tag management
const tagList = ref<Tag[]>([])
const newTagName = ref('')
const newTagColor = ref('#6b7280')
const editingTag = ref<Tag | null>(null)
const editTagName = ref('')
const editTagColor = ref('')
const tagError = ref('')

const TAG_COLORS = ['#6b7280', '#ef4444', '#f59e0b', '#10b981', '#3b82f6', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#6366f1']

async function loadTags() {
  try {
    const res = await getTags()
    tagList.value = res.data?.tags ?? []
  } catch { tagList.value = [] }
}

async function handleCreateTag() {
  tagError.value = ''
  const name = newTagName.value.trim()
  if (!name) return
  try {
    await createTag({ name, color: newTagColor.value })
    newTagName.value = ''
    newTagColor.value = '#6b7280'
    await loadTags()
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'response' in e) {
      const axiosErr = e as { response?: { data?: { error?: string } } }
      tagError.value = axiosErr.response?.data?.error ?? 'Failed to create tag'
    } else {
      tagError.value = 'Failed to create tag'
    }
  }
}

function startEditTag(tag: Tag) {
  editingTag.value = tag
  editTagName.value = tag.name
  editTagColor.value = tag.color
}

async function handleSaveTag() {
  tagError.value = ''
  if (!editingTag.value) return
  try {
    await updateTagApi(editingTag.value.id, { name: editTagName.value.trim(), color: editTagColor.value })
    editingTag.value = null
    await loadTags()
  } catch (e: unknown) {
    if (typeof e === 'object' && e !== null && 'response' in e) {
      const axiosErr = e as { response?: { data?: { error?: string } } }
      tagError.value = axiosErr.response?.data?.error ?? 'Failed to update tag'
    } else {
      tagError.value = 'Failed to update tag'
    }
  }
}

async function handleDeleteTag(tag: Tag) {
  const confirmed = await showConfirm(`Delete tag "${tag.name}"? It will be removed from all coins.`, { title: 'Delete Tag', variant: 'danger' })
  if (!confirmed) return
  try {
    await deleteTag(tag.id)
    await loadTags()
  } catch { /* ignore */ }
}

// Storage location management
const storageLocationList = ref<StorageLocation[]>([])
const newStorageLocationName = ref('')
const editingStorageLocation = ref<StorageLocation | null>(null)
const editStorageLocationName = ref('')
const storageLocationError = ref('')
const storageLocationsLoading = ref(false)
const storageLocationSaving = ref(false)
const deletingStorageLocationId = ref<number | null>(null)

function apiErrorText(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'response' in error) {
    const axiosErr = error as { response?: { status?: number; data?: { error?: string; message?: string; count?: number } } }
    const message = axiosErr.response?.data?.message ?? axiosErr.response?.data?.error
    if (axiosErr.response?.status === 409) {
      return message ?? "Can't delete — this location is used by coins. Reassign them first."
    }
    return message ?? fallback
  }
  return fallback
}

async function loadStorageLocations() {
  storageLocationsLoading.value = true
  storageLocationError.value = ''
  try {
    const res = await getStorageLocations()
    storageLocationList.value = res.data?.storageLocations ?? []
  } catch {
    storageLocationList.value = []
    storageLocationError.value = 'Failed to load storage locations'
  } finally {
    storageLocationsLoading.value = false
  }
}

async function handleCreateStorageLocation() {
  storageLocationError.value = ''
  const name = newStorageLocationName.value.trim()
  if (!name) return
  storageLocationSaving.value = true
  try {
    await createStorageLocation({ name })
    newStorageLocationName.value = ''
    await loadStorageLocations()
  } catch (error: unknown) {
    storageLocationError.value = apiErrorText(error, 'Failed to create storage location')
  } finally {
    storageLocationSaving.value = false
  }
}

function startEditStorageLocation(location: StorageLocation) {
  editingStorageLocation.value = location
  editStorageLocationName.value = location.name
  storageLocationError.value = ''
}

async function handleSaveStorageLocation() {
  storageLocationError.value = ''
  if (!editingStorageLocation.value) return
  const name = editStorageLocationName.value.trim()
  if (!name) return
  storageLocationSaving.value = true
  try {
    await updateStorageLocation(editingStorageLocation.value.id, { name })
    editingStorageLocation.value = null
    await loadStorageLocations()
  } catch (error: unknown) {
    storageLocationError.value = apiErrorText(error, 'Failed to update storage location')
  } finally {
    storageLocationSaving.value = false
  }
}

async function handleDeleteStorageLocation(location: StorageLocation) {
  storageLocationError.value = ''
  const confirmed = await showConfirm(`Delete storage location "${location.name}"? Coins must be reassigned first if this location is in use.`, { title: 'Delete Storage Location', variant: 'danger' })
  if (!confirmed) return
  deletingStorageLocationId.value = location.id
  try {
    await deleteStorageLocation(location.id)
    await loadStorageLocations()
  } catch (error: unknown) {
    storageLocationError.value = apiErrorText(error, 'Failed to delete storage location')
  } finally {
    deletingStorageLocationId.value = null
  }
}

// Migration
const migrationRunning = ref(false)
const migrationResult = ref<LegacyMigrationResult | null>(null)
const migrationError = ref('')

async function handleMigrate() {
  migrationRunning.value = true
  migrationError.value = ''
  migrationResult.value = null
  
  try {
    const res = await migrateLegacyReferences()
    migrationResult.value = res.data
  } catch (error: unknown) {
    migrationError.value = apiErrorText(error, 'Migration failed. Please try again.')
  } finally {
    migrationRunning.value = false
  }
}

onMounted(() => {
  loadTags()
  loadStorageLocations()
})

defineExpose({ loadTags, loadStorageLocations })
</script>
