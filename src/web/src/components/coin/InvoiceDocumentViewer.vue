<template>
  <Teleport to="body">
    <div
      class="fixed inset-0 z-[1000] flex items-center justify-center bg-black/85 p-4 backdrop-blur-sm max-md:p-0"
      data-testid="invoice-viewer"
      @click.self="emit('close')"
    >
      <div
        class="flex h-[90vh] w-full max-w-[900px] flex-col overflow-hidden rounded-md border border-border-accent bg-card shadow-glow max-md:h-full max-md:max-w-full max-md:rounded-none max-md:border-0"
        role="dialog"
        aria-modal="true"
        aria-label="Invoice viewer"
      >
        <div class="flex items-center justify-between gap-3 border-b border-border-subtle bg-card px-5 py-4 max-md:pt-[max(1rem,env(safe-area-inset-top))]">
          <div class="min-w-0">
            <h2 class="m-0 font-display text-lg text-heading">Invoice</h2>
            <p class="m-0 mt-0.5 truncate text-chip text-text-muted" data-testid="invoice-viewer-subtitle">{{ subtitle }}</p>
          </div>
          <button
            ref="closeButton"
            class="flex items-center justify-center rounded-sm p-1 text-text-muted transition-colors hover:bg-white/5 hover:text-text-primary"
            title="Close (Esc)"
            aria-label="Close"
            @click="emit('close')"
          >
            <X :size="20" />
          </button>
        </div>

        <div ref="scrollArea" class="relative min-h-0 flex-1 overflow-hidden bg-input">
          <div v-if="state === 'loading'" class="flex h-full flex-col items-center justify-center gap-4 text-center">
            <div class="h-10 w-10 animate-spin rounded-full border-[3px] border-border-subtle border-t-gold"></div>
            <p class="m-0 text-base text-gold">Loading invoice from Paperless…</p>
          </div>

          <div v-else-if="state === 'error'" class="flex h-full flex-col items-center justify-center gap-2 px-8 text-center" role="alert">
            <p class="m-0 text-base text-text-primary">Couldn't load the invoice</p>
            <p class="m-0 text-chip text-text-muted">{{ errorMessage }}</p>
          </div>

          <ZoomableSurface
            v-show="state === 'ready'"
            bare
            wheel-pans
            bound-pan
            :min-scale="1"
            :max-scale="4"
            aria-label="Invoice document"
          >
            <div class="flex flex-col items-center gap-4 p-4">
              <img
                v-if="kind === 'image' && imageUrl"
                :src="imageUrl"
                alt="Invoice"
                class="w-full rounded-sm shadow-[0_4px_12px_var(--overlay-30)]"
                draggable="false"
              />
              <canvas
                v-for="n in pageCount"
                v-else
                :key="n"
                :ref="(el) => setPageCanvas(n, el)"
                :data-page="n"
                class="block w-full rounded-[2px] bg-white shadow-[0_4px_14px_var(--overlay-45)]"
              ></canvas>
            </div>
          </ZoomableSurface>
        </div>

        <div class="flex items-center justify-between gap-3 border-t border-border-subtle bg-card px-5 py-4 max-md:pb-[max(1rem,env(safe-area-inset-bottom))]">
          <span class="text-sm text-text-muted" data-testid="invoice-viewer-footer-status">{{ footerStatus }}</span>
          <SafeExternalLink
            v-if="paperlessDocumentUrl"
            :href="paperlessDocumentUrl"
            class="btn btn-secondary btn-sm inline-flex items-center gap-1.5 no-underline"
          >
            Open in Paperless
            <ExternalLink :size="14" />
          </SafeExternalLink>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { ExternalLink, X } from 'lucide-vue-next'
import SafeExternalLink from '@/components/SafeExternalLink.vue'
import ZoomableSurface from '@/components/ZoomableSurface.vue'
import { getInvoiceDocument, getApiErrorMessage } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { paperlessDocumentUrl as buildPaperlessDocumentUrl } from '@/utils/paperless'

const props = defineProps<{
  coinId: number
  documentId: number
  invoiceNumber?: string | null
}>()

const emit = defineEmits<{ close: [] }>()

const auth = useAuthStore()
const state = ref<'loading' | 'ready' | 'error'>('loading')
const errorMessage = ref('')
const kind = ref<'pdf' | 'image'>('pdf')
const imageUrl = ref('')
const pageCount = ref(0)
const currentPage = ref(1)
const scrollArea = ref<HTMLElement | null>(null)
const closeButton = ref<HTMLElement | null>(null)
const canvases = new Map<number, HTMLCanvasElement>()
let observer: IntersectionObserver | null = null
let destroyed = false
let pdfDocument: { destroy: () => Promise<void> } | null = null

const paperlessDocumentUrl = computed(() => buildPaperlessDocumentUrl(auth.user?.paperlessUrl, props.documentId))

const subtitle = computed(() => {
  const parts = [props.invoiceNumber?.trim(), `Paperless #${props.documentId}`]
  if (kind.value === 'image' && state.value === 'ready') parts.push('photo')
  return parts.filter(Boolean).join(' · ')
})

const footerStatus = computed(() => {
  if (state.value !== 'ready') return ''
  if (kind.value === 'image') return 'Pinch to zoom'
  return `Page ${currentPage.value} of ${pageCount.value}`
})

function setPageCanvas(page: number, el: unknown) {
  if (el instanceof HTMLCanvasElement) canvases.set(page, el)
}

async function renderPdf(blob: Blob) {
  const pdfjs = await import('pdfjs-dist/legacy/build/pdf.mjs')
  const { default: workerUrl } = await import('pdfjs-dist/legacy/build/pdf.worker.min.mjs?url')
  pdfjs.GlobalWorkerOptions.workerSrc = workerUrl
  const doc = await pdfjs.getDocument({ data: new Uint8Array(await blob.arrayBuffer()) }).promise
  pdfDocument = doc
  if (destroyed) return
  pageCount.value = doc.numPages
  state.value = 'ready'
  await nextTick()

  const available = (scrollArea.value?.clientWidth ?? 360) - 32
  const pixelRatio = Math.min(window.devicePixelRatio || 1, 3)
  for (let n = 1; n <= doc.numPages && !destroyed; n++) {
    const page = await doc.getPage(n)
    const base = page.getViewport({ scale: 1 })
    // Render sharp enough for 2x zoom without re-rendering.
    const scale = (available / base.width) * pixelRatio * 2
    const viewport = page.getViewport({ scale })
    const canvas = canvases.get(n)
    if (!canvas) continue
    canvas.width = Math.floor(viewport.width)
    canvas.height = Math.floor(viewport.height)
    canvas.style.aspectRatio = `${base.width} / ${base.height}`
    const context = canvas.getContext('2d')
    if (!context) continue
    await page.render({ canvasContext: context, viewport }).promise
  }
  observePages()
}

function observePages() {
  if (typeof IntersectionObserver === 'undefined' || !scrollArea.value) return
  observer = new IntersectionObserver((entries) => {
    const visible = entries.filter((e) => e.isIntersecting).sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0]
    if (visible) currentPage.value = Number((visible.target as HTMLElement).dataset.page) || 1
  }, { root: scrollArea.value, threshold: [0.25, 0.5, 0.75] })
  canvases.forEach((canvas) => observer?.observe(canvas))
}

async function load() {
  try {
    const res = await getInvoiceDocument(props.coinId)
    if (destroyed) return
    const blob = res.data as Blob
    if (blob.type.startsWith('image/')) {
      kind.value = 'image'
      imageUrl.value = URL.createObjectURL(blob)
      state.value = 'ready'
    } else {
      kind.value = 'pdf'
      await renderPdf(blob)
    }
  } catch (err) {
    if (destroyed) return
    let message = getApiErrorMessage(err)
    const data = (err as { response?: { data?: unknown } })?.response?.data
    if (data instanceof Blob) {
      try {
        message = JSON.parse(await data.text()).error ?? message
      } catch { /* keep generic message */ }
    }
    errorMessage.value = message || 'Paperless could not be reached.'
    state.value = 'error'
  }
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
  document.body.style.overflow = 'hidden'
  closeButton.value?.focus()
  load()
})

onBeforeUnmount(() => {
  destroyed = true
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
  observer?.disconnect()
  if (imageUrl.value) URL.revokeObjectURL(imageUrl.value)
  pdfDocument?.destroy().catch(() => {})
})
</script>
