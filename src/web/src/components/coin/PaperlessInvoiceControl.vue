<template>
  <div class="mt-2 flex flex-wrap items-center gap-2" data-testid="paperless-invoice-control">
    <SafeExternalLink
      v-if="invoiceLink"
      :href="invoiceLink.url"
      class="inline-flex min-h-9 items-center gap-1 text-sm text-gold hover:text-bronze"
      data-testid="paperless-invoice-link"
    >
      <FileText :size="14" />
      Open in Paperless
    </SafeExternalLink>

    <template v-if="coinId">
      <button
        v-if="paperlessConfigured"
        type="button"
        class="btn btn-secondary btn-sm inline-flex min-h-9 items-center gap-1"
        :disabled="busy"
        data-testid="paperless-invoice-upload"
        @click="fileInput?.click()"
      >
        <Upload :size="14" />
        {{ busy ? statusLabel : linkedDocumentId ? 'Replace in Paperless' : 'Upload to Paperless' }}
      </button>
      <input
        ref="fileInput"
        type="file"
        class="hidden"
        accept="application/pdf,image/*"
        data-testid="paperless-invoice-file"
        @change="onFileSelected"
      />
      <router-link
        v-if="!paperlessConfigured"
        to="/settings"
        class="text-chip text-text-muted underline"
      >
        Connect paperless-ngx to upload invoices
      </router-link>
    </template>
    <span v-else-if="paperlessConfigured" class="text-chip text-text-muted">Save the coin to upload an invoice to Paperless.</span>

    <p v-if="message" class="m-0 w-full text-chip" :class="error ? 'text-loss' : 'text-text-muted'" role="status">{{ message }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { FileText, Upload } from 'lucide-vue-next'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import { resolvePaperlessInvoiceTask, uploadInvoiceToPaperless, getApiErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { resolveInvoiceLink } from '@/utils/paperless'

const props = defineProps<{
  coinId?: number
  invoice?: string | null
  documentId?: number | null
}>()

const emit = defineEmits<{ linked: [documentId: number] }>()

const RESOLVE_ATTEMPTS = 20
const RESOLVE_INTERVAL_MS = 3000

const auth = useAuthStore()
const fileInput = ref<HTMLInputElement | null>(null)
const busy = ref(false)
const statusLabel = ref('Uploading...')
const message = ref('')
const error = ref(false)
const linkedDocumentId = ref<number | null>(props.documentId ?? null)
let cancelled = false

const paperlessConfigured = computed(() => Boolean(auth.user?.paperlessConfigured))
const invoiceLink = computed(() =>
  resolveInvoiceLink(props.invoice, auth.user?.paperlessUrl, linkedDocumentId.value ?? props.documentId),
)

onBeforeUnmount(() => {
  cancelled = true
})

function wait(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function markLinked(documentId: number) {
  linkedDocumentId.value = documentId
  message.value = `Linked to Paperless document #${documentId}.`
  emit('linked', documentId)
}

async function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !props.coinId) return

  busy.value = true
  error.value = false
  message.value = ''
  statusLabel.value = 'Uploading...'
  try {
    let result = (await uploadInvoiceToPaperless(props.coinId, file)).data
    statusLabel.value = 'Processing...'
    for (let attempt = 0; result.status === 'pending' && attempt < RESOLVE_ATTEMPTS && !cancelled; attempt++) {
      message.value = 'Paperless is processing the invoice...'
      await wait(RESOLVE_INTERVAL_MS)
      result = (await resolvePaperlessInvoiceTask(props.coinId, result.taskId)).data
    }
    if (result.status === 'linked' && result.documentId) {
      markLinked(result.documentId)
    } else {
      message.value = 'Uploaded. Paperless is still processing; the link appears once it finishes and you reopen this coin.'
    }
  } catch (err) {
    error.value = true
    message.value = getApiErrorMessage(err) || 'Upload to Paperless failed'
  } finally {
    busy.value = false
  }
}
</script>
