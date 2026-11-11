import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises, RouterLinkStub } from '@vue/test-utils'
import PaperlessInvoiceControl from '../PaperlessInvoiceControl.vue'
import { uploadInvoiceToPaperless, resolvePaperlessInvoiceTask } from '@/api/client'

const authUser: { paperlessUrl?: string; paperlessConfigured?: boolean } = {}

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ user: authUser }),
}))

vi.mock('@/api/client', () => ({
  uploadInvoiceToPaperless: vi.fn(),
  resolvePaperlessInvoiceTask: vi.fn(),
  getApiErrorMessage: (err: { response?: { data?: { error?: string } } }) => err?.response?.data?.error ?? '',
}))

function mountControl(props: Record<string, unknown>) {
  return mount(PaperlessInvoiceControl, { props, global: { stubs: { RouterLink: RouterLinkStub } } })
}

async function chooseFile(wrapper: ReturnType<typeof mountControl>) {
  const input = wrapper.get('[data-testid="paperless-invoice-file"]')
  const file = new File(['%PDF'], 'invoice.pdf', { type: 'application/pdf' })
  Object.defineProperty(input.element, 'files', { value: [file], configurable: true })
  await input.trigger('change')
  return file
}

describe('PaperlessInvoiceControl', () => {
  beforeEach(() => {
    vi.mocked(uploadInvoiceToPaperless).mockReset()
    vi.mocked(resolvePaperlessInvoiceTask).mockReset()
    authUser.paperlessUrl = 'https://paperless.local'
    authUser.paperlessConfigured = true
  })

  it('shows an Open in Paperless link for a linked document', () => {
    const wrapper = mountControl({ coinId: 1, invoice: 'INV-1', documentId: 9 })
    expect(wrapper.get('[data-testid="paperless-invoice-link"]').attributes('href')).toBe('https://paperless.local/documents/9/details')
    expect(wrapper.get('[data-testid="paperless-invoice-upload"]').text()).toContain('Replace in Paperless')
  })

  it('uploads the chosen file and emits the linked document id', async () => {
    vi.mocked(uploadInvoiceToPaperless).mockResolvedValue({ data: { status: 'linked', taskId: 't', documentId: 77 } } as never)
    const wrapper = mountControl({ coinId: 5, invoice: '' })

    const file = await chooseFile(wrapper)
    await flushPromises()

    expect(uploadInvoiceToPaperless).toHaveBeenCalledWith(5, file)
    expect(wrapper.emitted('linked')).toEqual([[77]])
    expect(wrapper.get('[data-testid="paperless-invoice-link"]').attributes('href')).toBe('https://paperless.local/documents/77/details')
  })

  it('polls a pending task until it links', async () => {
    vi.useFakeTimers()
    try {
      vi.mocked(uploadInvoiceToPaperless).mockResolvedValue({ data: { status: 'pending', taskId: 'abc' } } as never)
      vi.mocked(resolvePaperlessInvoiceTask).mockResolvedValue({ data: { status: 'linked', taskId: 'abc', documentId: 12 } } as never)
      const wrapper = mountControl({ coinId: 5 })

      await chooseFile(wrapper)
      await flushPromises()
      await vi.advanceTimersByTimeAsync(3000)
      await flushPromises()

      expect(resolvePaperlessInvoiceTask).toHaveBeenCalledWith(5, 'abc')
      expect(wrapper.emitted('linked')).toEqual([[12]])
    } finally {
      vi.useRealTimers()
    }
  })

  it('shows the server error when the upload fails', async () => {
    vi.mocked(uploadInvoiceToPaperless).mockRejectedValue({ response: { data: { error: 'paperless-ngx rejected the API token' } } })
    const wrapper = mountControl({ coinId: 5 })

    await chooseFile(wrapper)
    await flushPromises()

    expect(wrapper.text()).toContain('paperless-ngx rejected the API token')
  })

  it('renders nothing when paperless is not set up at all', () => {
    authUser.paperlessUrl = ''
    authUser.paperlessConfigured = false
    const wrapper = mountControl({ coinId: 5, invoice: 'INV-1' })
    expect(wrapper.find('[data-testid="paperless-invoice-control"]').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('shows only the search link, no upload or prompt, when only the URL is saved', () => {
    authUser.paperlessConfigured = false
    const wrapper = mountControl({ coinId: 5, invoice: 'INV-1' })
    expect(wrapper.find('[data-testid="paperless-invoice-upload"]').exists()).toBe(false)
    expect(wrapper.text()).not.toMatch(/connect|settings/i)
    expect(wrapper.get('[data-testid="paperless-invoice-link"]').attributes('href')).toBe('https://paperless.local/documents?query=INV-1')
  })

  it('renders nothing on an unsaved coin without an invoice link', () => {
    const wrapper = mountControl({ invoice: '' })
    expect(wrapper.find('[data-testid="paperless-invoice-control"]').exists()).toBe(false)
  })
})

describe('PaperlessInvoiceControl on the detail page', () => {
  it('keeps the upload button but hides its own link when hideLink is set', () => {
    const wrapper = mount(PaperlessInvoiceControl, {
      props: { coinId: 3, invoice: 'INV-1', documentId: 4, hideLink: true },
      global: { stubs: { RouterLink: RouterLinkStub } },
    })
    expect(wrapper.find('[data-testid="paperless-invoice-link"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="paperless-invoice-upload"]').exists()).toBe(true)
  })
})
