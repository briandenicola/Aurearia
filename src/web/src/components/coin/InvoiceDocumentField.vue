<template>
  <div v-if="paperlessConfigured" class="mt-2" data-testid="invoice-document-field">
    <input
      ref="fileInput"
      type="file"
      class="hidden"
      accept="application/pdf,image/*"
      data-testid="invoice-document-file"
      @change="onFileSelected"
    />

    <!-- Error -->
    <div v-if="errorMessage" :class="[tileClass, 'border-[color-mix(in_srgb,var(--color-negative)_55%,transparent)] pr-1.5']" role="alert" data-testid="invoice-document-error">
      <span class="grid h-9 w-9 flex-none place-items-center rounded-sm bg-[color-mix(in_srgb,var(--color-negative)_12%,transparent)] text-loss">
        <CircleAlert :size="16" />
      </span>
      <span class="min-w-0 flex-1">
        <span class="block text-base leading-tight text-text-primary">Upload failed</span>
        <span class="mt-0.5 block text-xs leading-snug text-loss">{{ errorMessage }}</span>
      </span>
      <button type="button" class="btn btn-ghost btn-sm flex-none" @click="retry">Retry</button>
    </div>

    <!-- Uploading / processing -->
    <div v-else-if="busy || processing" :class="tileClass" aria-live="polite" data-testid="invoice-document-busy">
      <span class="grid h-9 w-9 flex-none place-items-center rounded-sm bg-gold-glow">
        <span class="h-[18px] w-[18px] animate-spin rounded-full border-2 border-border-subtle border-t-gold"></span>
      </span>
      <span class="min-w-0 flex-1">
        <span class="block truncate text-base leading-tight text-text-primary">{{ activeFileName || 'Invoice' }}</span>
        <span class="mt-0.5 block truncate text-xs text-text-muted">{{ busy && uploadingStage ? 'Uploading to Paperless…' : 'Processing in Paperless…' }}</span>
      </span>
    </div>

    <!-- Picked before the coin exists -->
    <div v-else-if="pendingFile" :class="[tileClass, 'pr-1.5']" data-testid="invoice-document-pending">
      <span class="grid h-[52px] w-10 flex-none place-items-center overflow-hidden rounded-[3px] bg-input text-gold">
        <img v-if="pendingPreviewUrl" :src="pendingPreviewUrl" alt="" class="h-full w-full object-cover" />
        <FileText v-else :size="18" />
      </span>
      <span class="min-w-0 flex-1">
        <span class="block truncate text-base leading-tight text-text-primary">{{ pendingFile.name }}</span>
        <span class="mt-0.5 block truncate text-xs text-text-muted">Uploads when you save</span>
      </span>
      <button type="button" class="inline-flex h-11 w-11 flex-none items-center justify-center rounded-sm text-text-secondary hover:text-text-primary" aria-label="Remove file" @click="clearPending">
        <X :size="18" />
      </button>
    </div>

    <!-- Attached -->
    <div v-else-if="documentId" :class="[tileClass, 'pr-1.5']" data-testid="invoice-document-attached">
      <button type="button" class="flex min-w-0 flex-1 items-center gap-3 bg-transparent p-0 text-left" aria-label="View invoice" @click="viewerOpen = true">
        <span class="grid h-[52px] w-10 flex-none place-items-center overflow-hidden rounded-[3px] bg-white shadow-[0_1px_4px_var(--overlay-45)]">
          <img v-if="thumbnailUrl" :src="thumbnailUrl" alt="" class="h-full w-full object-cover object-top" data-testid="invoice-document-thumbnail" />
          <FileText v-else :size="18" class="text-gold" />
        </span>
        <span class="min-w-0 flex-1">
          <span class="block truncate text-base leading-tight text-text-primary">Invoice in Paperless</span>
          <span class="mt-0.5 block truncate text-xs text-text-muted">Document #{{ documentId }} · tap to view</span>
        </span>
      </button>
      <AppOverflowMenu panel-width-class="w-[220px]" backdrop-aria-label="Close invoice options">
        <template #trigger="{ open, toggle }">
          <AppIconButton title="Invoice options" aria-label="Invoice options" :active="open" @click="toggle">
            <Ellipsis :size="20" />
          </AppIconButton>
        </template>
        <template #default="{ close }">
          <button type="button" :class="menuItemClass" @click="close(); pickFile()">
            <RefreshCw :size="16" /> Replace document
          </button>
          <button type="button" :class="menuItemClass" @click="close(); confirmUnlink()">
            <Unlink :size="16" /> Unlink from coin
          </button>
        </template>
      </AppOverflowMenu>
    </div>

    <!-- Empty -->
    <button
      v-else
      type="button"
      :class="[tileClass, 'border-dashed text-left transition-colors hover:border-gold']"
      data-testid="invoice-document-attach"
      @click="pickFile"
    >
      <span class="grid h-9 w-9 flex-none place-items-center rounded-sm border border-dashed border-border-subtle text-text-muted">
        <Paperclip :size="16" />
      </span>
      <span class="min-w-0 flex-1">
        <span class="block text-base leading-tight text-text-secondary">Attach invoice document</span>
        <span class="mt-0.5 block truncate text-xs text-text-muted">PDF or photo · stored in Paperless</span>
      </span>
    </button>

    <InvoiceDocumentViewer
      v-if="viewerOpen && coinId && documentId"
      :coin-id="coinId"
      :document-id="documentId"
      :invoice-number="invoiceNumber"
      @close="viewerOpen = false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { CircleAlert, Ellipsis, FileText, Paperclip, RefreshCw, Unlink, X } from 'lucide-vue-next'
import AppOverflowMenu from '@/components/ui/AppOverflowMenu.vue'
import AppIconButton from '@/components/ui/AppIconButton.vue'
import InvoiceDocumentViewer from '@/components/coin/InvoiceDocumentViewer.vue'
import {
  getApiErrorMessage,
  getInvoiceThumbnail,
  resolvePaperlessInvoiceTask,
  unlinkInvoiceDocument,
  uploadInvoiceToPaperless,
} from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useDialog } from '@/composables/useDialog'

const props = defineProps<{
  /** Undefined while adding a coin: the file is held and uploaded after save. */
  coinId?: number
  invoiceNumber?: string | null
  documentId?: number | null
  taskId?: string | null
}>()

const emit = defineEmits<{
  'update:documentId': [value: number | null]
  'update:taskId': [value: string]
}>()

const RESOLVE_INTERVAL_MS = 4000
const RESOLVE_ATTEMPTS = 45

const tileClass =
  'flex w-full min-h-16 items-center gap-3 rounded-sm border border-border-subtle bg-input py-2.5 pl-2.5 pr-3 text-text-primary'
const menuItemClass =
  'inline-flex w-full items-center gap-2 whitespace-nowrap rounded-sm px-3 py-2 text-left text-body text-text-secondary transition-all hover:bg-card-hover hover:text-text-primary'

const auth = useAuthStore()
const { showConfirm } = useDialog()

const fileInput = ref<HTMLInputElement | null>(null)
const pendingFile = ref<File | null>(null)
const pendingPreviewUrl = ref('')
const lastFile = ref<File | null>(null)
const activeFileName = ref('')
const busy = ref(false)
const uploadingStage = ref(false)
const errorMessage = ref('')
const thumbnailUrl = ref('')
const viewerOpen = ref(false)
let disposed = false
let pollRun = 0

const paperlessConfigured = computed(() => Boolean(auth.user?.paperlessConfigured))
const processing = computed(() => Boolean(props.taskId) && !props.documentId)

function pickFile() {
  errorMessage.value = ''
  fileInput.value?.click()
}

function setPending(file: File | null) {
  if (pendingPreviewUrl.value) URL.revokeObjectURL(pendingPreviewUrl.value)
  pendingPreviewUrl.value = file && file.type.startsWith('image/') ? URL.createObjectURL(file) : ''
  pendingFile.value = file
}

function clearPending() {
  setPending(null)
}

async function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!props.coinId) {
    setPending(file)
    return
  }
  await upload(file)
}

async function upload(file: File) {
  if (!props.coinId) return
  lastFile.value = file
  activeFileName.value = file.name
  errorMessage.value = ''
  busy.value = true
  uploadingStage.value = true
  try {
    const res = await uploadInvoiceToPaperless(props.coinId, file)
    uploadingStage.value = false
    if (res.data.status === 'linked' && res.data.documentId) {
      emit('update:taskId', '')
      emit('update:documentId', res.data.documentId)
    } else {
      emit('update:taskId', res.data.taskId)
    }
  } catch (err) {
    errorMessage.value = getApiErrorMessage(err) || 'Paperless could not be reached.'
  } finally {
    busy.value = false
    uploadingStage.value = false
  }
}

function retry() {
  errorMessage.value = ''
  if (lastFile.value) void upload(lastFile.value)
  else pickFile()
}

async function pollTask(taskId: string) {
  const run = ++pollRun
  for (let attempt = 0; attempt < RESOLVE_ATTEMPTS; attempt++) {
    await new Promise((resolve) => setTimeout(resolve, RESOLVE_INTERVAL_MS))
    if (disposed || run !== pollRun || !props.coinId || props.taskId !== taskId) return
    try {
      const res = await resolvePaperlessInvoiceTask(props.coinId, taskId)
      if (res.data.status === 'linked' && res.data.documentId) {
        emit('update:taskId', '')
        emit('update:documentId', res.data.documentId)
        return
      }
    } catch (err) {
      emit('update:taskId', '')
      errorMessage.value = getApiErrorMessage(err) || 'Paperless could not process the document.'
      return
    }
  }
}

async function confirmUnlink() {
  if (!props.coinId || !props.documentId) return
  const ok = await showConfirm(
    `This coin will no longer point to Paperless document #${props.documentId}. The document itself stays in Paperless.`,
    { title: 'Unlink invoice?', confirmLabel: 'Unlink' },
  )
  if (!ok) return
  try {
    await unlinkInvoiceDocument(props.coinId)
    emit('update:taskId', '')
    emit('update:documentId', null)
  } catch (err) {
    errorMessage.value = getApiErrorMessage(err) || 'Could not unlink the invoice.'
  }
}

async function loadThumbnail() {
  if (thumbnailUrl.value) URL.revokeObjectURL(thumbnailUrl.value)
  thumbnailUrl.value = ''
  if (!props.coinId || !props.documentId || !paperlessConfigured.value) return
  try {
    const res = await getInvoiceThumbnail(props.coinId)
    if (disposed) return
    thumbnailUrl.value = URL.createObjectURL(res.data as Blob)
  } catch {
    // Fall back to the document icon.
  }
}

watch(() => [props.coinId, props.documentId], loadThumbnail, { immediate: true })
watch(
  () => props.taskId,
  (taskId) => {
    if (taskId && !props.documentId && paperlessConfigured.value) void pollTask(taskId)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  disposed = true
  if (thumbnailUrl.value) URL.revokeObjectURL(thumbnailUrl.value)
  if (pendingPreviewUrl.value) URL.revokeObjectURL(pendingPreviewUrl.value)
})

defineExpose({ pendingFile })
</script>
