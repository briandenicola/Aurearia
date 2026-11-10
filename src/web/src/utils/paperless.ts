import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'

export interface InvoiceLink {
  url: string
  label: string
}

const PAPERLESS_DOC_PATH = /\/documents\/(\d+)(?:\/|$)/

/**
 * Resolves a coin's vendor invoice to a link into paperless-ngx.
 *
 * - If the invoice field already holds an http(s) URL (e.g. a pasted
 *   paperless document link), link to it directly.
 * - Otherwise, when the user configured a paperless-ngx base URL, link to a
 *   paperless full-text search for the invoice number.
 * - Otherwise there is no link.
 */
export function resolveInvoiceLink(
  invoice: string | null | undefined,
  paperlessBaseUrl: string | null | undefined,
): InvoiceLink | null {
  const value = invoice?.trim()
  if (!value) return null

  if (/^https?:\/\//i.test(value)) {
    const direct = sanitizeExternalUrl(value)
    if (!direct) return null
    const docMatch = new URL(direct).pathname.match(PAPERLESS_DOC_PATH)
    return { url: direct, label: docMatch ? `Document #${docMatch[1]}` : 'View invoice' }
  }

  const base = sanitizeExternalUrl(paperlessBaseUrl)
  if (!base) return null
  const searchUrl = `${base.replace(/\/+$/, '')}/documents?query=${encodeURIComponent(value)}`
  return { url: searchUrl, label: value }
}
