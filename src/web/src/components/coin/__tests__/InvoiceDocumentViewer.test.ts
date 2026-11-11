import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import InvoiceDocumentViewer from '../InvoiceDocumentViewer.vue'
import { getInvoiceDocument } from '@/api/client'

vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { paperlessUrl: 'https://paperless.local', paperlessConfigured: true } }) }))
vi.mock('@/api/client', () => ({
  getInvoiceDocument: vi.fn(),
  getApiErrorMessage: () => '',
}))

const pdfRender = vi.fn(() => ({ promise: Promise.resolve() }))
vi.mock('pdfjs-dist/legacy/build/pdf.mjs', () => ({
  GlobalWorkerOptions: {},
  getDocument: () => ({
    promise: Promise.resolve({
      numPages: 2,
      destroy: () => Promise.resolve(),
      getPage: () => Promise.resolve({
        getViewport: ({ scale }: { scale: number }) => ({ width: 600 * scale, height: 800 * scale }),
        render: pdfRender,
      }),
    }),
  }),
}))
vi.mock('pdfjs-dist/legacy/build/pdf.worker.min.mjs?url', () => ({ default: '/worker.mjs' }))

describe('InvoiceDocumentViewer', () => {
  beforeEach(() => {
    vi.mocked(getInvoiceDocument).mockReset()
    pdfRender.mockClear()
    globalThis.URL.createObjectURL = vi.fn(() => 'blob:invoice')
    globalThis.URL.revokeObjectURL = vi.fn()
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({})) as never
  })

  it('renders every PDF page in the app with a page indicator', async () => {
    vi.mocked(getInvoiceDocument).mockResolvedValue({ data: new Blob(['%PDF'], { type: 'application/pdf' }) } as never)
    const wrapper = mount(InvoiceDocumentViewer, { props: { coinId: 3, documentId: 482, invoiceNumber: '7NK85364NU046780H' }, attachTo: document.body })
    await flushPromises(); await flushPromises()
    expect(getInvoiceDocument).toHaveBeenCalledWith(3)
    expect(document.querySelectorAll('[data-testid="invoice-viewer"] canvas')).toHaveLength(2)
    expect(pdfRender).toHaveBeenCalledTimes(2)
    expect(document.body.textContent).toContain('Page 1 of 2')
    expect(document.body.textContent).toContain('7NK85364NU046780H · Paperless #482')
    const link = document.querySelector('[data-testid="invoice-viewer"] a') as HTMLAnchorElement
    expect(link.href).toBe('https://paperless.local/documents/482/details')
    wrapper.unmount()
  })

  it('shows a photo invoice as an image', async () => {
    vi.mocked(getInvoiceDocument).mockResolvedValue({ data: new Blob(['jpg'], { type: 'image/jpeg' }) } as never)
    const wrapper = mount(InvoiceDocumentViewer, { props: { coinId: 3, documentId: 511 }, attachTo: document.body })
    await flushPromises()
    expect((document.querySelector('[data-testid="invoice-viewer"] img') as HTMLImageElement).src).toBe('blob:invoice')
    expect(document.body.textContent).toContain('Pinch to zoom')
    expect(document.body.textContent).toContain('Paperless #511 · photo')
    wrapper.unmount()
  })

  it('explains a failure instead of showing a blank viewer', async () => {
    const errorBlob = new Blob([JSON.stringify({ error: 'paperless-ngx rejected the API token' })], { type: 'application/json' })
    vi.mocked(getInvoiceDocument).mockRejectedValue({ response: { data: errorBlob } })
    const wrapper = mount(InvoiceDocumentViewer, { props: { coinId: 3, documentId: 5 }, attachTo: document.body })
    await flushPromises(); await flushPromises()
    expect(document.body.textContent).toContain("Couldn't load the invoice")
    expect(document.body.textContent).toContain('paperless-ngx rejected the API token')
    wrapper.unmount()
  })

  it('closes on Escape and on the close button', async () => {
    vi.mocked(getInvoiceDocument).mockResolvedValue({ data: new Blob(['jpg'], { type: 'image/jpeg' }) } as never)
    const wrapper = mount(InvoiceDocumentViewer, { props: { coinId: 3, documentId: 5 }, attachTo: document.body })
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    ;(document.querySelector('[aria-label="Close"]') as HTMLButtonElement).click()
    expect(wrapper.emitted('close')).toHaveLength(2)
    wrapper.unmount()
  })
})
