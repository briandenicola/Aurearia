import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'

export interface InvoiceLink {
  url: string
  label: string
}

const PAPERLESS_DOC_PATH = /\/documents\/(\d+)(?:\/|$)/

function paperlessBase(paperlessBaseUrl: string | null | undefined): string | null {
  const base = sanitizeExternalUrl(paperlessBaseUrl)
  return base ? base.replace(/\/+$/, '') : null
}

/**
 * Resolves a coin's vendor invoice to a link into paperless-ngx, in order:
 *
 * 1. A paperless document linked by an invoice upload (`paperlessDocumentId`)
 *    opens that document, when the paperless-ngx URL is configured.
 * 2. An http(s) URL typed into the invoice field (e.g. a pasted paperless
 *    document link) opens directly.
 * 3. A plain invoice number opens a paperless full-text search for it.
 *
 * Returns null when none applies.
 */
export function resolveInvoiceLink(
  invoice: string | null | undefined,
  paperlessBaseUrl: string | null | undefined,
  paperlessDocumentId?: number | null,
): InvoiceLink | null {
  const value = invoice?.trim() ?? ''
  const base = paperlessBase(paperlessBaseUrl)

  if (base && paperlessDocumentId != null && paperlessDocumentId > 0) {
    return {
      url: `${base}/documents/${paperlessDocumentId}/details`,
      label: value && !/^https?:\/\//i.test(value) ? value : `Document #${paperlessDocumentId}`,
    }
  }

  if (!value) return null

  if (/^https?:\/\//i.test(value)) {
    const direct = sanitizeExternalUrl(value)
    if (!direct) return null
    const docMatch = new URL(direct).pathname.match(PAPERLESS_DOC_PATH)
    return { url: direct, label: docMatch ? `Document #${docMatch[1]}` : 'View invoice' }
  }

  if (!base) return null
  return { url: `${base}/documents?query=${encodeURIComponent(value)}`, label: value }
}
