<template>
  <div class="container pb-6">
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

    <div v-else-if="loadError" class="empty-state flex flex-col items-center gap-3 rounded-md border border-border-subtle bg-card text-text-secondary shadow-[var(--shadow-card)]">
      <AlertCircle :size="42" />
      <h3>Unable to load notes</h3>
      <p>{{ loadError }}</p>
      <button class="btn btn-secondary btn-sm" @click="loadNotes">Try Again</button>
    </div>

    <div v-else class="grid items-start gap-4 md:grid-cols-[minmax(240px,0.85fr)_minmax(0,2fr)]">
      <aside class="max-h-[280px] overflow-y-auto rounded-md border border-border-subtle bg-card p-4 shadow-[var(--shadow-card)] md:max-h-none md:min-h-[460px]" aria-label="Notes list">
        <div class="mb-4 flex items-start justify-between gap-4">
          <span class="section-label mb-0">All notes</span>
          <span class="chip-sm">{{ notes.length }} {{ notes.length === 1 ? 'note' : 'notes' }}</span>
        </div>

        <div v-if="notes.length === 0 && !editMode" class="flex min-h-[320px] flex-col items-center justify-center gap-3 text-center text-text-muted">
          <StickyNote :size="36" />
          <h3 class="m-0 text-lg text-text-primary">No notes yet</h3>
          <p class="max-w-[34rem] text-text-secondary">Capture ideas, links, and research threads that do not belong to a coin.</p>
          <button class="btn btn-primary btn-sm" @click="startNewNote">Create Note</button>
        </div>

        <button
          v-for="note in notes"
          :key="note.id"
          class="flex w-full flex-col gap-1 rounded-sm border border-transparent px-3 py-3 text-left text-text-secondary transition-all hover:border-border-accent hover:bg-gold-glow"
          :class="selectedId === note.id ? 'border-border-accent bg-gold-glow' : ''"
          @click="selectNote(note.id)"
        >
          <span class="flex min-w-0 items-center justify-between gap-2">
            <span class="truncate text-base font-semibold text-text-primary">{{ note.title || 'Untitled Note' }}</span>
            <span v-if="note.body" class="chip-sm">Markdown</span>
          </span>
          <span class="truncate text-chip text-text-muted">{{ previewText(note.body) }}</span>
          <span class="text-chip text-text-muted">Updated {{ formatDate(note.updatedAt) }}</span>
        </button>
      </aside>

      <section class="rounded-md border border-border-subtle bg-card p-6 shadow-[var(--shadow-card)] md:min-h-[460px]">
        <div v-if="!editMode && !selectedNote" class="flex min-h-[320px] flex-col items-center justify-center gap-3 text-center text-text-muted">
          <BookOpen :size="44" />
          <h2 class="m-0 text-xl text-text-primary">Select or create a note</h2>
          <p class="max-w-[34rem] text-text-secondary">Use notes for research leads, dealer links, show ideas, and collection thoughts.</p>
          <button class="btn btn-primary" @click="startNewNote">New Note</button>
        </div>

        <form v-else-if="editMode" @submit.prevent="saveNote">
          <div class="mb-4 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div>
              <p class="section-label">{{ selectedId === null ? 'New note' : 'Editing note' }}</p>
              <h2 class="m-0 text-xl text-text-primary">{{ selectedId === null ? 'Draft Note' : draftTitle || 'Untitled Note' }}</h2>
              <p class="text-chip text-text-muted">Write Markdown here; links and formatting render after save.</p>
            </div>
            <div class="flex flex-wrap gap-2 md:justify-end">
              <button type="button" class="btn btn-ghost btn-sm" @click="cancelEdit">Cancel</button>
              <button type="submit" class="btn btn-primary btn-sm" :disabled="saving">
                <Save :size="16" />
                {{ saving ? 'Saving...' : 'Save' }}
              </button>
            </div>
          </div>

          <p v-if="formError" class="mb-4 rounded-sm border border-error-bg bg-overlay p-3 text-base text-text-primary">{{ formError }}</p>

          <label class="form-group">
            <span class="form-label">Title</span>
            <input v-model="draftTitle" class="form-input" type="text" maxlength="200" placeholder="Idea, link, or research thread" />
          </label>

          <label class="form-group mb-0 block">
            <span class="form-label">Markdown</span>
            <textarea
              v-model="draftBody"
              class="form-textarea min-h-[300px] leading-[1.6] md:min-h-[360px]"
              maxlength="20000"
              rows="16"
              placeholder="Write plain Markdown here. Use links, lists, headings, and emphasis."
            ></textarea>
          </label>
        </form>

        <article v-else-if="selectedNote">
          <div class="mb-4 flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div>
              <p class="section-label">Display mode</p>
              <h2 class="m-0 text-xl text-text-primary">{{ selectedNote.title || 'Untitled Note' }}</h2>
              <p class="text-chip text-text-muted">Updated {{ formatDate(selectedNote.updatedAt) }}</p>
            </div>
            <div class="flex flex-wrap gap-2 md:justify-end">
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

          <div
            v-if="selectedNote.body"
            class="markdown-rendered rounded-sm border border-border-subtle bg-input p-4 text-body leading-[1.7] text-text-secondary [&_a]:break-words [&_a]:text-gold [&_blockquote]:mb-3 [&_h1]:mb-2 [&_h1]:mt-4 [&_h1]:text-gold [&_h2]:mb-2 [&_h2]:mt-4 [&_h2]:text-gold [&_h3]:mb-2 [&_h3]:mt-4 [&_h3]:text-gold [&_ol]:mb-3 [&_ol]:pl-5 [&_p]:mb-3 [&_strong]:text-text-primary [&_ul]:mb-3 [&_ul]:pl-5"
            v-html="renderedSelectedBody"
          ></div>
          <div v-else class="empty-state flex min-h-[320px] flex-col items-center justify-center gap-3 text-center text-text-muted">
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
/*
 * :deep() audit — markdown-rendered content
 * Target: HTML elements emitted by markdown-it inside .markdown-rendered.
 * Notes are stored as Markdown and rendered at runtime; the resulting HTML
 * nodes are not authored in the Vue template and cannot be styled by scope hash
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
</style>
