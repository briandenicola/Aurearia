<template>
  <section class="settings-section card">
    <h2>Data Management</h2>
    <div class="lookup-manager-grid">
      <section class="lookup-manager" aria-labelledby="tags-heading">
        <h3 id="tags-heading">Tags</h3>
        <p class="setting-desc">Create custom tags to organize and filter your coins.</p>

        <div class="tag-create-form">
          <input
            v-model="newTagName"
            type="text"
            class="form-input"
            placeholder="New tag name..."
            maxlength="50"
            @keydown.enter="handleCreateTag"
          />
          <div class="tag-color-picker">
            <button
              v-for="c in TAG_COLORS"
              :key="c"
              class="color-swatch"
              :class="{ active: newTagColor === c }"
              :style="{ backgroundColor: c }"
              @click="newTagColor = c"
            ></button>
          </div>
          <button class="btn btn-primary btn-sm" @click="handleCreateTag" :disabled="!newTagName.trim()">Create Tag</button>
        </div>
        <p v-if="tagError" class="tag-error">{{ tagError }}</p>

        <div v-if="tagList.length" class="tag-list">
          <div v-for="tag in tagList" :key="tag.id" class="tag-list-item">
            <template v-if="editingTag?.id === tag.id">
              <input v-model="editTagName" class="form-input tag-edit-input" maxlength="50" @keydown.enter="handleSaveTag" />
              <div class="tag-color-picker">
                <button
                  v-for="c in TAG_COLORS"
                  :key="c"
                  class="color-swatch sm"
                  :class="{ active: editTagColor === c }"
                  :style="{ backgroundColor: c }"
                  @click="editTagColor = c"
                ></button>
              </div>
              <button class="btn btn-primary btn-sm" @click="handleSaveTag">Save</button>
              <button class="btn btn-secondary btn-sm" @click="editingTag = null">Cancel</button>
            </template>
            <template v-else>
              <span class="tag-preview" :style="{ backgroundColor: tag.color + '22', color: tag.color, borderColor: tag.color + '44' }">{{ tag.name }}</span>
              <div class="tag-actions">
                <button class="btn btn-secondary btn-sm" @click="startEditTag(tag)">Edit</button>
                <button class="btn btn-danger btn-sm" @click="handleDeleteTag(tag)">Delete</button>
              </div>
            </template>
          </div>
        </div>
        <p v-else class="empty-tags">No tags created yet. Create your first tag above.</p>
      </section>

      <section class="lookup-manager" aria-labelledby="storage-locations-heading">
        <h3 id="storage-locations-heading">Storage Locations</h3>
        <p class="setting-desc">Create shelf, tray, safe, or box locations for the coin form dropdown.</p>

        <div class="tag-create-form">
          <input
            v-model="newStorageLocationName"
            type="text"
            class="form-input"
            placeholder="New storage location..."
            maxlength="100"
            :disabled="storageLocationSaving"
            @keydown.enter="handleCreateStorageLocation"
          />
          <button class="btn btn-primary btn-sm" @click="handleCreateStorageLocation" :disabled="!newStorageLocationName.trim() || storageLocationSaving">
            {{ storageLocationSaving ? 'Saving...' : 'Create Location' }}
          </button>
        </div>
        <p v-if="storageLocationError" class="tag-error">{{ storageLocationError }}</p>
        <p v-if="storageLocationsLoading" class="empty-tags">Loading storage locations...</p>

        <div v-else-if="storageLocationList.length" class="tag-list">
          <div v-for="location in storageLocationList" :key="location.id" class="tag-list-item">
            <template v-if="editingStorageLocation?.id === location.id">
              <input v-model="editStorageLocationName" class="form-input tag-edit-input" maxlength="100" @keydown.enter="handleSaveStorageLocation" />
              <button class="btn btn-primary btn-sm" @click="handleSaveStorageLocation" :disabled="storageLocationSaving">Save</button>
              <button class="btn btn-secondary btn-sm" @click="editingStorageLocation = null" :disabled="storageLocationSaving">Cancel</button>
            </template>
            <template v-else>
              <span class="chip-sm storage-location-preview">{{ location.name }}</span>
              <div class="tag-actions">
                <button class="btn btn-secondary btn-sm" @click="startEditStorageLocation(location)">Edit</button>
                <button class="btn btn-danger btn-sm" :disabled="deletingStorageLocationId === location.id" @click="handleDeleteStorageLocation(location)">
                  {{ deletingStorageLocationId === location.id ? 'Deleting...' : 'Delete' }}
                </button>
              </div>
            </template>
          </div>
        </div>
        <p v-else class="empty-tags">No storage locations created yet. Create your first location above.</p>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  getTags, createTag, updateTag as updateTagApi, deleteTag,
  getStorageLocations, createStorageLocation, updateStorageLocation, deleteStorageLocation,
} from '@/api/client'
import { useDialog } from '@/composables/useDialog'
import type { Tag, StorageLocation } from '@/types'

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

onMounted(() => {
  loadTags()
  loadStorageLocations()
})

defineExpose({ loadTags, loadStorageLocations })
</script>

<style scoped>
.settings-section h2 {
  font-size: 1.1rem;
  margin-bottom: 1.25rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.settings-section h3 {
  font-size: 0.95rem;
  margin-top: 1.25rem;
  margin-bottom: 0.75rem;
  color: var(--text-secondary);
}

.setting-desc {
  font-size: 0.75rem;
  color: var(--text-muted);
}

/* Lookup Managers */
.lookup-manager-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-top: 2rem;
}

.lookup-manager {
  min-width: 0;
}

.lookup-manager h3 {
  margin-top: 0;
}

.tag-create-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  margin: 1rem 0;
}

.tag-create-form .form-input {
  flex: 1;
  min-width: 150px;
}

.tag-color-picker {
  display: flex;
  gap: 0.3rem;
  align-items: center;
}

.color-swatch {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid transparent;
  cursor: pointer;
  padding: 0;
}

.color-swatch.active {
  border-color: var(--text-primary);
  box-shadow: 0 0 0 2px var(--bg-card);
}

.color-swatch.sm {
  width: 18px;
  height: 18px;
}

.tag-error {
  color: #ef4444;
  font-size: 0.85rem;
  margin-top: 0.25rem;
}

.tag-list {
  margin-top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.tag-list-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  flex-wrap: wrap;
}

.tag-preview {
  font-size: 0.8rem;
  padding: 0.2rem 0.6rem;
  border-radius: var(--radius-full);
  border: 1px solid;
  flex-shrink: 0;
}

.storage-location-preview {
  color: var(--text-primary);
  background: var(--bg-input);
  flex-shrink: 0;
}

.tag-edit-input {
  flex: 1;
  min-width: 120px;
}

.tag-actions {
  margin-left: auto;
  display: flex;
  gap: 0.25rem;
}

.empty-tags {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-top: 1rem;
}

@media (max-width: 768px) {
  .lookup-manager-grid {
    grid-template-columns: 1fr;
  }
}
</style>
