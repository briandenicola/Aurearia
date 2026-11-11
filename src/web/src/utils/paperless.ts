import { sanitizeExternalUrl } from '@/composables/useSafeExternalLink'

/** Web UI address of a paperless-ngx document, or null without a valid base URL. */
export function paperlessDocumentUrl(
  paperlessBaseUrl: string | null | undefined,
  documentId: number | null | undefined,
): string | null {
  const base = sanitizeExternalUrl(paperlessBaseUrl)
  if (!base || documentId == null || documentId <= 0) return null
  return `${base.replace(/\/+$/, '')}/documents/${documentId}/details`
}

/**
 * When the Vendor Invoice field itself holds an http(s) URL (for example a
 * pasted paperless document link), that URL; otherwise null. Plain invoice
 * numbers never become links.
 */
export function externalInvoiceUrl(invoice: string | null | undefined): string | null {
  const value = invoice?.trim() ?? ''
  if (!/^https?:\/\//i.test(value)) return null
  return sanitizeExternalUrl(value)
}
