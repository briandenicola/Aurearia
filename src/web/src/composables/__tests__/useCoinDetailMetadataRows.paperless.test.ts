import { describe, expect, it } from 'vitest'
import { useCoinDetailMetadataRows } from '@/composables/useCoinDetailMetadataRows'
import type { Coin } from '@/types'

const invoiceRow = (coin: Partial<Coin>, options: { paperlessConfigured?: boolean; paperlessError?: string }) =>
  useCoinDetailMetadataRows(coin as Coin, options).rows.value.find((row) => row.key === 'vendorInvoice')

describe('vendor invoice row and paperless uploads', () => {
  it('shows a pending upload as processing', () => {
    expect(invoiceRow({ paperlessTaskId: 't-1', paperlessDocumentId: null }, { paperlessConfigured: true })?.value)
      .toBe('Processing in paperless…')
    expect(invoiceRow({ vendorInvoice: 'INV-9', paperlessTaskId: 't-1' }, { paperlessConfigured: true })?.value)
      .toBe('INV-9 (processing in paperless…)')
  })

  it('shows the linked invoice once the document exists', () => {
    const row = invoiceRow({ paperlessTaskId: '', paperlessDocumentId: 4950 }, { paperlessConfigured: true })
    expect(row?.value).toBe('View invoice')
    expect(row?.activatable).toBe(true)
  })

  it('shows a failed upload with the reason', () => {
    const row = invoiceRow({ paperlessTaskId: '' }, { paperlessConfigured: true, paperlessError: 'duplicate' })
    expect(row?.value).toBe('Upload failed: duplicate')
    expect(row?.valueClass).toBe('text-loss')
  })

  it('ignores paperless state when paperless is not configured', () => {
    expect(invoiceRow({ paperlessTaskId: 't-1' }, { paperlessConfigured: false })).toBeUndefined()
  })
})
