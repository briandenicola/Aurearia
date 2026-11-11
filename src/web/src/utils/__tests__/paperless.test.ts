import { describe, it, expect } from 'vitest'
import { resolveInvoiceLink } from '@/utils/paperless'

describe('resolveInvoiceLink', () => {
  it('returns null for an empty invoice', () => {
    expect(resolveInvoiceLink('', 'https://paperless.local')).toBeNull()
    expect(resolveInvoiceLink(undefined, 'https://paperless.local')).toBeNull()
  })

  it('returns null when no paperless URL is configured and the invoice is plain text', () => {
    expect(resolveInvoiceLink('INV-2026-0421', '')).toBeNull()
  })

  it('builds a paperless search link for a plain invoice number', () => {
    expect(resolveInvoiceLink('INV 2026/0421', 'https://paperless.local/')).toEqual({
      url: 'https://paperless.local/documents?query=INV%202026%2F0421',
      label: 'INV 2026/0421',
    })
  })

  it('keeps a subpath on the configured base URL', () => {
    expect(resolveInvoiceLink('411', 'http://10.0.0.5:8000/paperless')?.url).toBe(
      'http://10.0.0.5:8000/paperless/documents?query=411',
    )
  })

  it('links directly to a pasted paperless document URL', () => {
    expect(resolveInvoiceLink('https://paperless.local/documents/123/details', '')).toEqual({
      url: 'https://paperless.local/documents/123/details',
      label: 'Document #123',
    })
  })

  it('links directly to other http(s) URLs with a generic label', () => {
    expect(resolveInvoiceLink('https://vendor.example.com/inv.pdf', '')?.label).toBe('View invoice')
  })

  it('rejects non-http configured base URLs', () => {
    expect(resolveInvoiceLink('INV-1', 'javascript:alert(1)')).toBeNull()
  })
})

describe('resolveInvoiceLink with an uploaded paperless document', () => {
  it('prefers the linked document over a search', () => {
    expect(resolveInvoiceLink('INV-1', 'https://paperless.local/', 42)).toEqual({
      url: 'https://paperless.local/documents/42/details',
      label: 'INV-1',
    })
  })

  it('links the document even when the invoice field is empty', () => {
    expect(resolveInvoiceLink('', 'https://paperless.local', 42)).toEqual({
      url: 'https://paperless.local/documents/42/details',
      label: 'Document #42',
    })
  })

  it('falls back when no paperless URL is configured', () => {
    expect(resolveInvoiceLink('INV-1', '', 42)).toBeNull()
    expect(resolveInvoiceLink('', '', 42)).toBeNull()
  })
})
