import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AdminOIDCSection from '@/components/admin/AdminOIDCSection.vue'
import type { OIDCAdminProvider } from '@/types'

const mockGetAdminOIDCProviders = vi.fn()
const mockCreateAdminOIDCProvider = vi.fn()
const mockUpdateAdminOIDCProvider = vi.fn()
const mockDeleteAdminOIDCProvider = vi.fn()
const mockTestAdminOIDCProvider = vi.fn()
const mockShowAlert = vi.fn()
const mockShowConfirm = vi.fn()
const mockRouterPush = vi.fn()

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockRouterPush }),
}))

vi.mock('@/api/client', () => ({
  getAdminOIDCProviders: () => mockGetAdminOIDCProviders(),
  createAdminOIDCProvider: (payload: Record<string, unknown>) => mockCreateAdminOIDCProvider(payload),
  updateAdminOIDCProvider: (id: number, payload: Record<string, unknown>) => mockUpdateAdminOIDCProvider(id, payload),
  deleteAdminOIDCProvider: (id: number) => mockDeleteAdminOIDCProvider(id),
  testAdminOIDCProvider: (id: number) => mockTestAdminOIDCProvider(id),
  getApiErrorMessage: (error: unknown) => {
    const maybeError = error as { response?: { data?: { error?: string; message?: string } }; message?: string }
    return maybeError.response?.data?.error ?? maybeError.response?.data?.message ?? maybeError.message ?? ''
  },
}))

vi.mock('@/composables/useDialog', () => ({
  useDialog: () => ({
    showAlert: mockShowAlert,
    showConfirm: mockShowConfirm,
  }),
}))

const provider: OIDCAdminProvider = {
  id: 7,
  name: 'entra-work',
  displayName: 'Microsoft',
  providerType: 'entra',
  enabled: true,
  issuerUrl: 'https://login.microsoftonline.com/tenant/v2.0',
  clientId: 'client-id',
  clientSecretConfigured: true,
  scopes: ['openid', 'profile', 'email'],
  callbackPath: '/api/auth/oidc/7/callback',
  requireVerifiedEmail: true,
  lastTestStatus: 'ok',
  lastTestMessage: 'Discovery succeeded',
}

function providerResponse(providers: OIDCAdminProvider[] = [provider]) {
  return { data: { providers } }
}

function buttonByText(wrapper: ReturnType<typeof mount>, text: string) {
  const button = wrapper.findAll('button').find((candidate) => candidate.text() === text)
  expect(button, `button ${text} should exist`).toBeTruthy()
  return button!
}

describe('AdminOIDCSection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetAdminOIDCProviders.mockResolvedValue(providerResponse())
    mockCreateAdminOIDCProvider.mockResolvedValue({ data: provider })
    mockUpdateAdminOIDCProvider.mockResolvedValue({ data: provider })
    mockDeleteAdminOIDCProvider.mockResolvedValue({ data: {} })
    mockTestAdminOIDCProvider.mockResolvedValue({
      data: {
        available: true,
        message: 'Discovery succeeded',
        issuer: provider.issuerUrl,
        authorizationEndpoint: `${provider.issuerUrl}/authorize`,
        tokenEndpoint: `${provider.issuerUrl}/token`,
      },
    })
    mockShowConfirm.mockResolvedValue(true)
    mockShowAlert.mockResolvedValue(true)
    mockRouterPush.mockResolvedValue(undefined)
  })

  it('preserves a configured secret on edit unless a new secret is entered', async () => {
    const wrapper = mount(AdminOIDCSection)
    await flushPromises()

    await buttonByText(wrapper, 'Edit').trigger('click')
    expect(wrapper.find('#oidc-client-secret').attributes('placeholder')).toContain('leave blank to preserve')

    await wrapper.find('#oidc-display-name').setValue('Microsoft Entra')
    await wrapper.find('form.modal-body').trigger('submit.prevent')
    await flushPromises()

    expect(mockUpdateAdminOIDCProvider).toHaveBeenCalledWith(7, expect.not.objectContaining({
      clientSecret: expect.anything(),
    }))

    await buttonByText(wrapper, 'Edit').trigger('click')
    await wrapper.find('#oidc-client-secret').setValue('Configured')
    await wrapper.find('form.modal-body').trigger('submit.prevent')
    await flushPromises()

    expect(mockUpdateAdminOIDCProvider).toHaveBeenLastCalledWith(7, expect.not.objectContaining({
      clientSecret: expect.anything(),
    }))

    await buttonByText(wrapper, 'Edit').trigger('click')
    await wrapper.find('#oidc-client-secret').setValue('new-secret')
    await wrapper.find('form.modal-body').trigger('submit.prevent')
    await flushPromises()

    expect(mockUpdateAdminOIDCProvider).toHaveBeenLastCalledWith(7, expect.objectContaining({
      clientSecret: 'new-secret',
    }))
  })

  it('derives the Entra issuer URL from the tenant ID on create', async () => {
    mockGetAdminOIDCProviders.mockResolvedValue(providerResponse([]))
    const wrapper = mount(AdminOIDCSection)
    await flushPromises()

    await buttonByText(wrapper, 'Add Provider').trigger('click')
    await wrapper.find('#oidc-name').setValue('entra-work')
    await wrapper.find('#oidc-display-name').setValue('Microsoft Entra')
    await wrapper.find('#oidc-tenant-id').setValue('new-tenant')
    await wrapper.find('#oidc-client-id').setValue('client-id')
    await wrapper.find('#oidc-client-secret').setValue('client-secret')

    expect(wrapper.text()).toContain('https://login.microsoftonline.com/new-tenant/v2.0')

    await wrapper.find('form.modal-body').trigger('submit.prevent')
    await flushPromises()

    expect(mockCreateAdminOIDCProvider).toHaveBeenCalledWith(expect.objectContaining({
      providerType: 'entra',
      issuerUrl: 'https://login.microsoftonline.com/new-tenant/v2.0',
      clientSecret: 'client-secret',
    }))
  })

  it('infers the Entra tenant ID from an existing issuer URL on edit', async () => {
    const wrapper = mount(AdminOIDCSection)
    await flushPromises()

    await buttonByText(wrapper, 'Edit').trigger('click')

    const tenantInput = wrapper.find<HTMLInputElement>('#oidc-tenant-id')
    expect(tenantInput.element.value).toBe('tenant')
    expect(wrapper.text()).toContain('https://login.microsoftonline.com/tenant/v2.0')

    await tenantInput.setValue('rotated-tenant')
    await wrapper.find('form.modal-body').trigger('submit.prevent')
    await flushPromises()

    expect(mockUpdateAdminOIDCProvider).toHaveBeenCalledWith(7, expect.objectContaining({
      issuerUrl: 'https://login.microsoftonline.com/rotated-tenant/v2.0',
    }))
  })

  it('does not render or reuse an accidental secret value returned by the API', async () => {
    mockGetAdminOIDCProviders.mockResolvedValue(providerResponse([
      {
        ...provider,
        clientSecret: 'server-secret-should-not-render',
      } as OIDCAdminProvider & { clientSecret: string },
    ]))

    const wrapper = mount(AdminOIDCSection)
    await flushPromises()

    expect(wrapper.text()).not.toContain('server-secret-should-not-render')
    await buttonByText(wrapper, 'Edit').trigger('click')
    const secretInput = wrapper.find<HTMLInputElement>('#oidc-client-secret')
    expect(secretInput.element.value).toBe('')

    await wrapper.find('form.modal-body').trigger('submit.prevent')
    await flushPromises()

    expect(mockUpdateAdminOIDCProvider).toHaveBeenCalledWith(7, expect.not.objectContaining({
      clientSecret: expect.anything(),
    }))
  })

  it('renders provider test status and shows distinct test failure messages', async () => {
    mockGetAdminOIDCProviders.mockResolvedValue(providerResponse([
      {
        ...provider,
        lastTestStatus: 'failed',
        lastTestMessage: 'Issuer metadata is unreachable',
      },
    ]))
    mockTestAdminOIDCProvider.mockResolvedValue({
      data: {
        available: false,
        message: 'Discovery endpoint returned 404',
        issuer: provider.issuerUrl,
        authorizationEndpoint: '',
        tokenEndpoint: '',
      },
    })

    const wrapper = mount(AdminOIDCSection)
    await flushPromises()

    expect(wrapper.text()).toContain('Discovery failed')
    expect(wrapper.text()).toContain('Issuer metadata is unreachable')

    await buttonByText(wrapper, 'Test Discovery').trigger('click')
    await flushPromises()

    expect(mockTestAdminOIDCProvider).toHaveBeenCalledWith(7)
    expect(wrapper.text()).toContain('Discovery failed')
    expect(wrapper.text()).toContain('Discovery endpoint returned 404')
    expect(wrapper.text()).toContain('Discovery tests do not validate the client secret.')
  })

  it('shows save errors from the admin provider API', async () => {
    mockGetAdminOIDCProviders.mockResolvedValue(providerResponse([]))
    mockCreateAdminOIDCProvider.mockRejectedValue({
      response: { data: { error: 'Issuer URL is invalid' } },
    })

    const wrapper = mount(AdminOIDCSection)
    await flushPromises()

    await buttonByText(wrapper, 'Add Provider').trigger('click')
    await wrapper.find('#oidc-name').setValue('pocket-home')
    await wrapper.find('#oidc-display-name').setValue('Pocket ID')
    await wrapper.find('#oidc-provider-type').setValue('pocket_id')
    await wrapper.find('#oidc-issuer-url').setValue('https://id.example.com')
    await wrapper.find('#oidc-client-id').setValue('client-id')
    await wrapper.find('#oidc-client-secret').setValue('client-secret')
    await wrapper.find('form.modal-body').trigger('submit.prevent')
    await flushPromises()

    expect(mockCreateAdminOIDCProvider).toHaveBeenCalledWith(expect.objectContaining({
      name: 'pocket-home',
      providerType: 'pocket_id',
      clientSecret: 'client-secret',
      scopes: ['openid', 'profile', 'email'],
    }))
    expect(wrapper.text()).toContain('Issuer URL is invalid')
  })

  it('links to the OIDC setup guide in Settings Help', async () => {
    const wrapper = mount(AdminOIDCSection)
    await flushPromises()

    await buttonByText(wrapper, 'Setup Guide').trigger('click')

    expect(mockRouterPush).toHaveBeenCalledWith({
      path: '/settings',
      query: { tab: 'help', section: 'oidc' },
    })
  })
})
