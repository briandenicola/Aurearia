<template>
  <div class="container notes-page">
    <div class="page-header">
      <h1>Notes</h1>
      <div v-if="isPwa" class="pwa-actions">
        <button class="pwa-icon-btn" @click="startNewNote" title="New Note">
          <CirclePlus :size="22" />
        </button>
      </div>
      <div v-else class="header-actions">
        <button class="btn btn-primary" @click="startNewNote">
          <Plus :size="16" /> New Note
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-overlay">
      <div class="spinner" />
      <p>Loading notes...</p>
    </div>

    <div v-else-if="loadError" class="empty-state card error-state">
      <AlertCircle :size="42" />
      <h3>Unable to load notes</h3>
      <p>{{ loadError }}</p>
      <button class="btn btn-secondary btn-sm" @click="loadNotes">Try Again</button>
    </div>

    <div v-else class="notes-layout">
      <aside class="notes-list card" aria-label="Notes list">
        <div class="notes-list-header">
          <span class="section-label">All notes</span>
          <span class="chip-sm">{{ notes.length }} {{ notes.length === 1 ? 'note' : 'notes' }}</span>
        </div>

        <div v-if="notes.length === 0 && !editMode" class="notes-empty-list">
          <StickyNote :size="36" />
          <h3>No notes yet</h3>
          <p>Capture ideas, links, and research threads that do not belong to a coin.</p>
          <button class="btn btn-primary btn-sm" @click="startNewNote">Create Note</button>
        </div>

        <button
          v-for="note in notes"
          :key="note.id"
          class="note-list-item"
          :class="{ active: selectedId === note.id }"
          @click="selectNote(note.id)"
        >
          <span class="note-list-topline">
            <span class="note-list-title">{{ note.title || 'Untitled Note' }}</span>
            <span v-if="note.body" class="chip-sm">Markdown</span>
          </span>
          <span class="note-list-preview">{{ previewText(note.body) }}</span>
          <span class="note-list-date">Updated {{ formatDate(note.updatedAt) }}</span>
        </button>
      </aside>

      <section class="note-editor card">
        <div v-if="!editMode && !selectedNote" class="note-placeholder">
          <BookOpen :size="44" />
          <h2>Select or create a note</h2>
          <p>Use notes for research leads, dealer links, show ideas, and collection thoughts.</p>
          <button class="btn btn-primary" @click="startNewNote">New Note</button>
        </div>

        <form v-else-if="editMode" class="note-form" @submit.prevent="saveNote">
          <div class="note-panel-header">
            <div>
              <p class="section-label">{{ selectedId === null ? 'New note' : 'Editing note' }}</p>
              <h2>{{ selectedId === null ? 'Draft Note' : draftTitle || 'Untitled Note' }}</h2>
              <p class="note-updated">Write Markdown here; links and formatting render after save.</p>
            </div>
            <div class="note-actions">
              <button type="button" class="btn btn-ghost btn-sm" @click="cancelEdit">Cancel</button>
              <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">
                <Save :size="16" />
                {{ saving ? 'Saving...' : 'Save' }}
              </button>
            </div>
          </div>

          <p v-if="formError" class="form-error">{{ formError }}</p>

          <label class="form-group">
            <span class="form-label">Title</span>
            <input v-model="draftTitle" class="form-input" type="text" maxlength="200" placeholder="Idea, link, or research thread" />
          </label>

          <label class="form-group note-body-field">
            <span class="form-label">Markdown</span>
            <textarea
              v-model="draftBody"
              class="form-textarea markdown-source"
              maxlength="20000"
              rows="16"
              placeholder="Write plain Markdown here. Use links, lists, headings, and emphasis."
            ></textarea>
          </label>
        </form>

        <article v-else-if="selectedNote" class="note-display">
          <div class="note-panel-header">
            <div>
              <p class="section-label">Display mode</p>
              <h2>{{ selectedNote.title || 'Untitled Note' }}</h2>
              <p class="note-updated">Updated {{ formatDate(selectedNote.updatedAt) }}</p>
            </div>
            <div class="note-actions">
              <button class="btn btn-secondary btn-sm" @click="editSelected">
                <Pencil :size="16" />
                Edit
              </button>
              <button class="btn btn-danger btn-sm" :disabled="deleting" @click="deleteSelected">
                <Trash2 :size="16" />
                {{ deleting ? 'Deleting...' : 'Delete' }}
              </button>
            </div>
          </div>

          <div v-if="selectedNote.body" class="section-content-card markdown-rendered" v-html="renderedSelectedBody"></div>
          <div v-else class="empty-state note-empty-body">
            <p>This note has no body yet.</p>
            <button class="btn btn-secondary btn-sm" @click="editSelected">Add Markdown</button>
          </div>
        </article>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { AlertCircle, BookOpen, CirclePlus, Pencil, Plus, Save, StickyNote, Trash2 } from 'lucide-vue-next'
import { createNote, deleteNote, getNotes, updateNote } from '@/api/client'
import type { UserNote } from '@/types'
import { useDialog } from '@/composables/useDialog'
import { renderSafeMarkdown } from '@/composables/useMarkdown'
import { usePwa } from '@/composables/usePwa'

const { showConfirm, showAlert } = useDialog()
const { isPwa } = usePwa()

const notes = ref<UserNote[]>([])
const selectedId = ref<number | null>(null)
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)
const editMode = ref(false)
const loadError = ref('')
const formError = ref('')
const draftTitle = ref('')
const draftBody = ref('')

const selectedNote = computed(() => notes.value.find(note => note.id === selectedId.value) ?? null)
const renderedSelectedBody = computed(() => renderSafeMarkdown(selectedNote.value?.body))

onMounted(loadNotes)

async function loadNotes() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await getNotes()
    notes.value = [...(res.data.notes ?? [])].sort(sortByUpdatedDesc)
    selectedId.value = notes.value[0]?.id ?? null
  } catch {
    loadError.value = 'Notes are not available right now. Check your connection and try again.'
  } finally {
    loading.value = false
  }
}

function sortByUpdatedDesc(a: UserNote, b: UserNote): number {
  return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
}

function startNewNote() {
  selectedId.value = null
  draftTitle.value = ''
  draftBody.value = ''
  formError.value = ''
  editMode.value = true
}

function selectNote(id: number) {
  selectedId.value = id
  formError.value = ''
  editMode.value = false
}

function editSelected() {
  if (!selectedNote.value) return
  draftTitle.value = selectedNote.value.title
  draftBody.value = selectedNote.value.body
  formError.value = ''
  editMode.value = true
}

function cancelEdit() {
  formError.value = ''
  if (selectedId.value === null) {
    editMode.value = false
    selectedId.value = notes.value[0]?.id ?? null
    return
  }
  editMode.value = false
}

async function saveNote() {
  const body = draftBody.value
  const title = deriveTitle(draftTitle.value, body)
  if (!title && !body.trim()) {
    formError.value = 'Add a title or note body before saving.'
    return
  }

  saving.value = true
  formError.value = ''
  try {
    if (selectedId.value === null) {
      const res = await createNote({ title: title || 'Untitled Note', body })
      notes.value = [res.data, ...notes.value].sort(sortByUpdatedDesc)
      selectedId.value = res.data.id
    } else {
      const res = await updateNote(selectedId.value, { title: title || 'Untitled Note', body })
      notes.value = notes.value.map(note => note.id === res.data.id ? res.data : note).sort(sortByUpdatedDesc)
      selectedId.value = res.data.id
    }
    editMode.value = false
  } catch {
    formError.value = 'Unable to save this note. Try again before leaving the page.'
  } finally {
    saving.value = false
  }
}

function deriveTitle(rawTitle: string, body: string): string {
  const title = rawTitle.trim()
  if (title) return title
  const firstLine = body.split('\n').find(line => line.trim())
  return firstLine?.replace(/^#+\s*/, '').trim().slice(0, 80) ?? ''
}

async function deleteSelected() {
  if (!selectedNote.value) return
  const confirmed = await showConfirm('Delete this note? This cannot be undone.', {
    title: 'Delete Note',
    confirmLabel: 'Delete',
    variant: 'danger',
  })
  if (!confirmed) return

  deleting.value = true
  try {
    const id = selectedNote.value.id
    await deleteNote(id)
    notes.value = notes.value.filter(note => note.id !== id)
    selectedId.value = notes.value[0]?.id ?? null
    editMode.value = false
  } catch {
    await showAlert('Unable to delete this note. Try again.', { title: 'Delete Failed' })
  } finally {
    deleting.value = false
  }
}

function previewText(body: string): string {
  const compact = body.replace(/[#*_`>\-[\]()]/g, ' ').replace(/\s+/g, ' ').trim()
  return compact || 'No body yet'
}

function formatDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Unknown date'
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })
}
</script>

<style scoped>
.notes-page {
  padding-bottom: 1.5rem;
}

.note-panel-header h2 {
  margin: 0;
}

.notes-layout {
  display: grid;
  grid-template-columns: minmax(240px, 0.85fr) minmax(0, 2fr);
  gap: 1rem;
  align-items: start;
}

.notes-list,
.note-editor {
  min-height: 460px;
}

.notes-list {
  padding: 1rem;
  border-color: var(--border-subtle);
}

.notes-list-header,
.note-panel-header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.note-list-item {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.75rem;
  border: 1px solid transparent;
  background: transparent;
  color: var(--text-secondary);
  text-align: left;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.note-list-item:hover,
.note-list-item.active {
  border-color: var(--border-accent);
  background: var(--accent-gold-glow);
}

.note-list-topline {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  justify-content: space-between;
  min-width: 0;
}

.note-list-title {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 0.9rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.note-list-preview,
.note-list-date,
.note-updated {
  color: var(--text-muted);
  font-size: 0.8rem;
}

.note-list-preview {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notes-empty-list,
.note-placeholder,
.note-empty-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  text-align: center;
  color: var(--text-muted);
  min-height: 320px;
}

.notes-empty-list h3,
.note-placeholder h2 {
  margin: 0;
}

.notes-empty-list p,
.note-placeholder p {
  max-width: 34rem;
  color: var(--text-secondary);
}

.note-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  justify-content: flex-end;
}

.form-error {
  padding: 0.75rem;
  margin-bottom: 1rem;
  border: 1px solid var(--error-bg);
  color: var(--text-primary);
  background: var(--overlay-dark);
}

.note-editor {
  padding: 1.5rem;
  border-color: var(--border-subtle);
}

.note-body-field {
  margin-bottom: 0;
}

.markdown-source {
  min-height: 360px;
  line-height: 1.6;
}

.markdown-rendered {
  padding: 1rem;
  background: var(--bg-input);
  border-color: var(--border-subtle);
  color: var(--text-secondary);
  line-height: 1.7;
}

/*
 * :deep() audit — markdown-rendered content
 * Target: HTML elements emitted by markdown-it inside .markdown-rendered.
 * Notes are stored as Markdown and rendered at runtime; the resulting HTML
 * elements carry no Vue-scope hash and cannot be reached by scoped selectors
 * or Tailwind utilities.
 */
.markdown-rendered :deep(h1),
.markdown-rendered :deep(h2),
.markdown-rendered :deep(h3) {
  margin: 1rem 0 0.5rem;
  color: var(--accent-gold);
}

.markdown-rendered :deep(p),
.markdown-rendered :deep(ul),
.markdown-rendered :deep(ol),
.markdown-rendered :deep(blockquote) {
  margin-bottom: 0.75rem;
}

.markdown-rendered :deep(ul),
.markdown-rendered :deep(ol) {
  padding-left: 1.25rem;
}

.markdown-rendered :deep(strong) {
  color: var(--text-primary);
}

.markdown-rendered :deep(a) {
  color: var(--accent-gold);
  word-break: break-word;
}

.error-state {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  align-items: center;
}

@media (max-width: 768px) {
  .notes-layout {
    grid-template-columns: 1fr;
  }

  .notes-list,
  .note-editor {
    min-height: auto;
  }

  .notes-list {
    max-height: 280px;
    overflow-y: auto;
  }

  .note-panel-header {
    flex-direction: column;
    align-items: stretch;
  }

  .note-actions {
    justify-content: flex-start;
  }

  .markdown-source {
    min-height: 300px;
  }
}
</style>
