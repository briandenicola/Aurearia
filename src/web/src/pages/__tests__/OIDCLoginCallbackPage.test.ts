import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import OIDCLoginCallbackPage from '@/pages/OIDCLoginCallbackPage.vue'
import type { AuthResponse } from '@/types'

const mockReplace = vi.fn()
const mockCompleteOIDCLoginCallback = vi.fn()
const mockApplyAuthResponse = vi.fn()
const mockRoute = {
  params: { providerId: '7' } as Record<string, string>,
  query: { code: 'auth-code', state: 'opaque-state' } as Record<string, string | string[] | undefined>,
}

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace: mockReplace }),
  useRoute: () => mockRoute,
  RouterLink: { template: '<a><slot /></a>' },
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    applyAuthResponse: mockApplyAuthResponse,
  }),
}))

vi.mock('@/api/client', () => ({
  completeOIDCLoginCallback: (providerId: number, code: string, state: string) =>
    mockCompleteOIDCLoginCallback(providerId, code, state),
  getApiErrorMessage: (error: unknown) => {
    const maybeError = error as { response?: { data?: { error?: string; message?: string } }; message?: string }
    return maybeError.response?.data?.error ?? maybeError.response?.data?.message ?? maybeError.message ?? ''
  },
}))

const authResponse: AuthResponse = {
  token: 'jwt-token',
  refreshToken: 'refresh-token',
  user: {
    id: 1,
    username: 'collector',
    role: 'user',
    email: 'collector@example.com',
    avatarPath: '',
    isPublic: false,
    bio: '',
    zipCode: '',
  },
}

function mountPage() {
  return mount(OIDCLoginCallbackPage, {
    global: {
      stubs: {
        RouterLink: { template: '<a><slot /></a>' },
      },
    },
  })
}

describe('OIDCLoginCallbackPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockRoute.params = { providerId: '7' }
    mockRoute.query = { code: 'auth-code', state: 'opaque-state' }
    mockCompleteOIDCLoginCallback.mockResolvedValue({ data: authResponse })
    mockApplyAuthResponse.mockResolvedValue(undefined)
  })

  it('exchanges the code with the API, stores auth response, and strips query secrets from the URL', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(mockReplace).toHaveBeenCalledWith({ name: 'oidc-login-callback', params: { providerId: '7' } })
    expect(mockCompleteOIDCLoginCallback).toHaveBeenCalledWith(7, 'auth-code', 'opaque-state')
    expect(mockApplyAuthResponse).toHaveBeenCalledWith(authResponse)
    expect(wrapper.text()).toContain('Sign In Complete')
    expect(wrapper.text()).toContain('Continue to Collection')
    expect(wrapper.text()).not.toContain('jwt-token')
    expect(wrapper.text()).not.toContain('refresh-token')
  })

  it('shows a safe account-conflict message without storing tokens', async () => {
    mockCompleteOIDCLoginCallback.mockRejectedValue({
      response: { status: 409, data: { error: 'account conflict' } },
    })

    const wrapper = mountPage()
    await flushPromises()

    expect(mockApplyAuthResponse).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Sign In Failed')
    expect(wrapper.text()).toContain('Sign in locally, then link the provider from Account Settings.')
  })

  it('shows safe provider configuration detail for token exchange failures', async () => {
    mockCompleteOIDCLoginCallback.mockRejectedValue({
      response: {
        status: 400,
        data: {
          error: 'OIDC authorization code was rejected',
          detail: 'provider rejected the client secret; for Entra, paste the client secret Value, not the Secret ID',
        },
      },
    })

    const wrapper = mountPage()
    await flushPromises()

    expect(mockApplyAuthResponse).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Sign In Failed')
    expect(wrapper.text()).toContain('The sign-in provider is not configured correctly: provider rejected the client secret; for Entra, paste the client secret Value, not the Secret ID. Ask an administrator to test the provider settings.')
    expect(wrapper.text()).not.toContain('The provider response could not be validated.')
  })
})
