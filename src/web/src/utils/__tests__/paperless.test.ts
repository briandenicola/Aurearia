import { describe, it, expect } from 'vitest'
import { externalInvoiceUrl, paperlessDocumentUrl } from '@/utils/paperless'

describe('paperlessDocumentUrl', () => {
  it('builds the document details URL from the configured base', () => {
    expect(paperlessDocumentUrl('https://paperless.local/', 42)).toBe('https://paperless.local/documents/42/details')
    expect(paperlessDocumentUrl('http://10.0.0.5:8000/paperless', 7)).toBe('http://10.0.0.5:8000/paperless/documents/7/details')
  })

  it('returns null without a usable base or document', () => {
    expect(paperlessDocumentUrl('', 42)).toBeNull()
    expect(paperlessDocumentUrl('javascript:alert(1)', 42)).toBeNull()
    expect(paperlessDocumentUrl('https://paperless.local', null)).toBeNull()
    expect(paperlessDocumentUrl('https://paperless.local', 0)).toBeNull()
  })
})

describe('externalInvoiceUrl', () => {
  it('returns a pasted http(s) URL', () => {
    expect(externalInvoiceUrl(' https://paperless.local/documents/3/details ')).toBe('https://paperless.local/documents/3/details')
  })

  it('never turns a plain invoice number into a link', () => {
    expect(externalInvoiceUrl('INV-2026-0421')).toBeNull()
    expect(externalInvoiceUrl('487323')).toBeNull()
    expect(externalInvoiceUrl('')).toBeNull()
    expect(externalInvoiceUrl('javascript:alert(1)')).toBeNull()
  })
})
