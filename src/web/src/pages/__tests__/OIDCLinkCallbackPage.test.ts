import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OIDCLinkCallbackPage from '@/pages/OIDCLinkCallbackPage.vue'

const mockReplace = vi.fn()
const mockCompleteOIDCLinkCallback = vi.fn()
const mockRoute = {
  params: { providerId: '7' } as Record<string, string>,
  query: { code: 'auth-code', state: 'opaque-state' } as Record<string, string | string[] | undefined>,
}

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: mockReplace }),
  useRoute: () => mockRoute,
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/api/client', () => ({
  completeOIDCLinkCallback: (providerId: number, code: string, state: string) =>
    mockCompleteOIDCLinkCallback(providerId, code, state),
  getApiErrorMessage: (error: unknown) => {
    const maybeError = error as { response?: { data?: { error?: string; message?: string } }; message?: string }
    return maybeError.response?.data?.error ?? maybeError.response?.data?.message ?? maybeError.message ?? ''
  },
}))

function mountPage() {
  return mount(OIDCLinkCallbackPage, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('OIDCLinkCallbackPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockRoute.params = { providerId: '7' }
    mockRoute.query = { code: 'auth-code', state: 'opaque-state' }
    mockCompleteOIDCLinkCallback.mockResolvedValue({
      data: {
        message: 'OIDC identity linked',
        identity: {
          id: 1,
          providerId: 7,
          providerDisplayName: 'Microsoft',
          issuer: 'https://login.microsoftonline.com/tenant/v2.0',
          subjectPreview: 'subject...',
          email: 'collector@example.com',
          emailVerified: true,
          createdAt: '2026-06-24T17:00:00Z',
          lastLoginAt: null,
        },
      },
    })
  })

  it('exchanges the link code with the API and strips query secrets from the URL', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(mockReplace).toHaveBeenCalledWith({ name: 'oidc-link-callback', params: { providerId: '7' } })
    expect(mockCompleteOIDCLinkCallback).toHaveBeenCalledWith(7, 'auth-code', 'opaque-state')
    expect(wrapper.text()).toContain('Provider Linked')
    expect(wrapper.text()).toContain('collector@example.com')
    expect(wrapper.text()).not.toContain('auth-code')
    expect(wrapper.text()).not.toContain('opaque-state')
  })

  it('maps redirect URI token-exchange failures to provider configuration guidance', async () => {
    mockCompleteOIDCLinkCallback.mockRejectedValue({
      response: {
        status: 400,
        data: {
          error: 'OIDC authorization code was rejected',
          detail: 'provider rejected the redirect URI; confirm the exact callback URL is registered',
        },
      },
    })

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('Linking Failed')
    expect(wrapper.text()).toContain('The sign-in provider is not configured correctly: provider rejected the redirect URI; confirm the exact callback URL is registered. Ask an administrator to review the provider settings.')
    expect(wrapper.text()).not.toContain('The provider response could not be validated.')
  })
})
