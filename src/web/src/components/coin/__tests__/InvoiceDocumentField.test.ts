import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import InvoiceDocumentField from '../InvoiceDocumentField.vue'
import {
  getInvoiceThumbnail,
  resolvePaperlessInvoiceTask,
  unlinkInvoiceDocument,
  uploadInvoiceToPaperless,
} from '@/api/client'

const authUser: { paperlessConfigured?: boolean; paperlessUrl?: string } = {}
const showConfirm = vi.fn()

vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: authUser }) }))
vi.mock('@/composables/useDialog', () => ({ useDialog: () => ({ showConfirm }) }))
vi.mock('@/components/coin/InvoiceDocumentViewer.vue', () => ({
  default: { name: 'InvoiceDocumentViewer', props: ['coinId', 'documentId', 'invoiceNumber'], template: '<div data-testid="viewer-stub" />' },
}))
vi.mock('@/api/client', () => ({
  uploadInvoiceToPaperless: vi.fn(),
  resolvePaperlessInvoiceTask: vi.fn(),
  unlinkInvoiceDocument: vi.fn(),
  getInvoiceThumbnail: vi.fn(),
  getApiErrorMessage: (err: { response?: { data?: { error?: string } } }) => err?.response?.data?.error ?? '',
}))

async function chooseFile(wrapper: ReturnType<typeof mount>, name = 'invoice.pdf', type = 'application/pdf') {
  const input = wrapper.get('[data-testid="invoice-document-file"]')
  const file = new File(['%PDF'], name, { type })
  Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
  await input.trigger('change')
  return file
}

describe('InvoiceDocumentField', () => {
  beforeEach(() => {
    vi.mocked(uploadInvoiceToPaperless).mockReset()
    vi.mocked(resolvePaperlessInvoiceTask).mockReset()
    vi.mocked(unlinkInvoiceDocument).mockReset()
    vi.mocked(getInvoiceThumbnail).mockReset().mockRejectedValue(new Error('no thumb'))
    showConfirm.mockReset()
    authUser.paperlessConfigured = true
    authUser.paperlessUrl = 'https://paperless.local'
  })

  it('renders nothing when paperless is not connected', () => {
    authUser.paperlessConfigured = false
    const wrapper = mount(InvoiceDocumentField, { props: { coinId: 1, documentId: 5 } })
    expect(wrapper.html()).not.toContain('invoice-document-field')
    expect(wrapper.text()).toBe('')
  })

  it('shows the dashed attach tile when nothing is attached', () => {
    const wrapper = mount(InvoiceDocumentField, { props: { coinId: 1 } })
    expect(wrapper.get('[data-testid="invoice-document-attach"]').text()).toContain('Attach Invoice')
    expect(wrapper.text()).toContain('PDF or photo')
  })

  it('uploads on an existing coin and reports the linked document', async () => {
    vi.mocked(uploadInvoiceToPaperless).mockResolvedValue({ data: { status: 'linked', taskId: 't', documentId: 77 } } as never)
    const wrapper = mount(InvoiceDocumentField, { props: { coinId: 4 } })
    const file = await chooseFile(wrapper)
    await flushPromises()
    expect(uploadInvoiceToPaperless).toHaveBeenCalledWith(4, file)
    expect(wrapper.emitted('update:documentId')).toEqual([[77]])
  })

  it('stores a pending task and polls it until linked', async () => {
    vi.useFakeTimers()
    try {
      vi.mocked(uploadInvoiceToPaperless).mockResolvedValue({ data: { status: 'pending', taskId: 'abc' } } as never)
      vi.mocked(resolvePaperlessInvoiceTask).mockResolvedValue({ data: { status: 'linked', taskId: 'abc', documentId: 12 } } as never)
      const wrapper = mount(InvoiceDocumentField, { props: { coinId: 4 } })
      await chooseFile(wrapper)
      await flushPromises()
      expect(wrapper.emitted('update:taskId')).toEqual([['abc']])
      await wrapper.setProps({ taskId: 'abc' })
      expect(wrapper.get('[data-testid="invoice-document-busy"]').text()).toContain('Processing in Paperless')
      await vi.advanceTimersByTimeAsync(4000)
      await flushPromises()
      expect(resolvePaperlessInvoiceTask).toHaveBeenCalledWith(4, 'abc')
      expect(wrapper.emitted('update:documentId')).toEqual([[12]])
    } finally {
      vi.useRealTimers()
    }
  })

  it('holds the file on a new coin instead of uploading', async () => {
    const wrapper = mount(InvoiceDocumentField, { props: {} })
    const file = await chooseFile(wrapper)
    expect(uploadInvoiceToPaperless).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="invoice-document-pending"]').text()).toContain('Uploads when you save')
    expect((wrapper.vm as unknown as { pendingFile: File | null }).pendingFile).toBe(file)
    await wrapper.get('[aria-label="Remove file"]').trigger('click')
    expect((wrapper.vm as unknown as { pendingFile: File | null }).pendingFile).toBeNull()
  })

  it('shows the error inside the tile and retries the same file', async () => {
    vi.mocked(uploadInvoiceToPaperless)
      .mockRejectedValueOnce({ response: { data: { error: 'paperless-ngx rejected the API token' } } })
      .mockResolvedValueOnce({ data: { status: 'linked', taskId: 't', documentId: 3 } } as never)
    const wrapper = mount(InvoiceDocumentField, { props: { coinId: 4 } })
    const file = await chooseFile(wrapper)
    await flushPromises()
    expect(wrapper.get('[data-testid="invoice-document-error"]').text()).toContain('paperless-ngx rejected the API token')
    await wrapper.get('[data-testid="invoice-document-error"] button').trigger('click')
    await flushPromises()
    expect(uploadInvoiceToPaperless).toHaveBeenLastCalledWith(4, file)
    expect(wrapper.emitted('update:documentId')).toEqual([[3]])
  })

  it('opens the viewer from the attached tile and unlinks after confirmation', async () => {
    vi.mocked(unlinkInvoiceDocument).mockResolvedValue({} as never)
    showConfirm.mockResolvedValue(true)
    const wrapper = mount(InvoiceDocumentField, { props: { coinId: 4, documentId: 9, invoiceNumber: 'INV-9' } })
    expect(wrapper.text()).toContain('Document #9 · tap to view')

    await wrapper.get('[aria-label="View invoice"]').trigger('click')
    expect(wrapper.find('[data-testid="viewer-stub"]').exists()).toBe(true)

    await wrapper.get('[aria-label="Invoice options"]').trigger('click')
    const unlink = wrapper.findAll('button').find((b) => b.text().includes('Unlink from coin'))!
    await unlink.trigger('click')
    await flushPromises()
    expect(showConfirm).toHaveBeenCalledWith(expect.stringContaining('stays in Paperless'), expect.objectContaining({ title: 'Unlink invoice?' }))
    expect(unlinkInvoiceDocument).toHaveBeenCalledWith(4)
    expect(wrapper.emitted('update:documentId')).toEqual([[null]])
  })

  it('does not unlink when the confirmation is cancelled', async () => {
    showConfirm.mockResolvedValue(false)
    const wrapper = mount(InvoiceDocumentField, { props: { coinId: 4, documentId: 9 } })
    await wrapper.get('[aria-label="Invoice options"]').trigger('click')
    await wrapper.findAll('button').find((b) => b.text().includes('Unlink from coin'))!.trigger('click')
    await flushPromises()
    expect(unlinkInvoiceDocument).not.toHaveBeenCalled()
  })
})
