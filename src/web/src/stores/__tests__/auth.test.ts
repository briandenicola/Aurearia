import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import type { AxiosResponse } from 'axios'
import type { AuthResponse, User } from '@/types'

// Mock the API client module
vi.mock('@/api/client', () => ({
  login: vi.fn(),
  register: vi.fn(),
  webauthnLoginBegin: vi.fn(),
  webauthnLoginFinish: vi.fn(),
  refreshAccessToken: vi.fn(),
  onTokenRefreshed: vi.fn(),
}))

import * as api from '@/api/client'
import { onTokenRefreshed } from '@/api/client'
import { useAuthStore } from '../auth'

const mockUser: User = {
  id: 1,
  username: 'testuser',
  role: 'user',
  email: 'test@example.com',
  avatarPath: '',
  isPublic: false,
  bio: '',
  zipCode: '',
}

const mockAdminUser: User = {
  ...mockUser,
  id: 2,
  username: 'admin',
  role: 'admin',
}

const mockAuthResponse: AuthResponse = {
  token: 'jwt-token-abc',
  refreshToken: 'refresh-token-xyz',
  user: mockUser,
}

function makeAssertionCredential(): PublicKeyCredential {
  return {
    id: 'credential-id',
    rawId: new Uint8Array([1, 2, 3]).buffer,
    type: 'public-key',
    response: {
      authenticatorData: new Uint8Array([4, 5, 6]).buffer,
      clientDataJSON: new Uint8Array([7, 8, 9]).buffer,
      signature: new Uint8Array([10, 11, 12]).buffer,
      userHandle: null,
    },
  } as PublicKeyCredential
}

function getStorageMock(): Record<string, string> {
  const store: Record<string, string> = {}
  return store
}

describe('Auth Store', () => {
  let storageMock: Record<string, string>
  let credentialsGet: ReturnType<typeof vi.fn>
  let cacheDelete: ReturnType<typeof vi.fn>

  beforeEach(() => {
    storageMock = getStorageMock()
    credentialsGet = vi.fn()
    cacheDelete = vi.fn().mockResolvedValue(true)

    // Mock localStorage
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storageMock[key] ?? null),
      setItem: vi.fn((key: string, value: string) => { storageMock[key] = value }),
      removeItem: vi.fn((key: string) => { delete storageMock[key] }),
    })
    Object.defineProperty(navigator, 'credentials', {
      value: { get: credentialsGet },
      configurable: true,
    })
    vi.stubGlobal('caches', {
      delete: cacheDelete,
    })

    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('starts unauthenticated with no user or token', () => {
      const store = useAuthStore()
      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(store.isAdmin).toBe(false)
    })

    it('restores token and user from localStorage on creation', () => {
      storageMock['token'] = 'persisted-token'
      storageMock['user'] = JSON.stringify(mockUser)

      // Re-create pinia so the store reads from the seeded localStorage
      setActivePinia(createPinia())
      const store = useAuthStore()

      expect(store.token).toBe('persisted-token')
      expect(store.user).toEqual(mockUser)
      expect(store.isAuthenticated).toBe(true)
    })
  })

  describe('computed properties', () => {
    it('isAuthenticated is true when token is present', () => {
      storageMock['token'] = 'some-token'
      storageMock['user'] = JSON.stringify(mockUser)
      setActivePinia(createPinia())
      const store = useAuthStore()
      expect(store.isAuthenticated).toBe(true)
    })

    it('isAdmin is true when user role is admin', () => {
      storageMock['token'] = 'admin-token'
      storageMock['user'] = JSON.stringify(mockAdminUser)
      setActivePinia(createPinia())
      const store = useAuthStore()
      expect(store.isAdmin).toBe(true)
    })

    it('isAdmin is false for regular users', () => {
      storageMock['token'] = 'user-token'
      storageMock['user'] = JSON.stringify(mockUser)
      setActivePinia(createPinia())
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
    })

    it('isAdmin is false when user is null', () => {
      const store = useAuthStore()
      expect(store.isAdmin).toBe(false)
    })
  })

  describe('doLogin', () => {
    it('calls api.login and sets auth state', async () => {
      vi.mocked(api.login).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)
      const store = useAuthStore()

      await store.doLogin('testuser', 'password123')

      expect(api.login).toHaveBeenCalledWith('testuser', 'password123')
      expect(store.token).toBe('jwt-token-abc')
      expect(store.user).toEqual(mockUser)
      expect(store.isAuthenticated).toBe(true)
    })

    describe('applyAuthResponse', () => {
      it('accepts OIDC callback auth responses and persists the session', async () => {
        const store = useAuthStore()

        await store.applyAuthResponse(mockAuthResponse)

        expect(store.token).toBe(mockAuthResponse.token)
        expect(store.user).toEqual(mockUser)
        expect(localStorage.setItem).toHaveBeenCalledWith('refreshToken', mockAuthResponse.refreshToken)
      })

      it('clears private media caches when OIDC callback switches users', async () => {
        const firstAuth: AuthResponse = { ...mockAuthResponse, token: 'first-token' }
        const secondAuth: AuthResponse = {
          token: 'second-token',
          refreshToken: 'second-refresh',
          user: mockAdminUser,
        }
        const store = useAuthStore()

        await store.applyAuthResponse(firstAuth)
        await store.applyAuthResponse(secondAuth)

        expect(store.user).toEqual(mockAdminUser)
        expect(cacheDelete).toHaveBeenCalledWith('coin-images')
      })
    })

    it('persists token, refreshToken, and user to localStorage', async () => {
      vi.mocked(api.login).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)
      const store = useAuthStore()

      await store.doLogin('testuser', 'password123')

      expect(localStorage.setItem).toHaveBeenCalledWith('token', 'jwt-token-abc')
      expect(localStorage.setItem).toHaveBeenCalledWith('refreshToken', 'refresh-token-xyz')
      expect(localStorage.setItem).toHaveBeenCalledWith('user', JSON.stringify(mockUser))
    })

    it('propagates API errors', async () => {
      vi.mocked(api.login).mockRejectedValue(new Error('Invalid credentials'))
      const store = useAuthStore()

      await expect(store.doLogin('bad', 'creds')).rejects.toThrow('Invalid credentials')
      expect(store.isAuthenticated).toBe(false)
    })

    it('overwrites previous session on double login', async () => {
      const firstAuth: AuthResponse = { ...mockAuthResponse, token: 'first-token' }
      const secondAuth: AuthResponse = {
        token: 'second-token',
        refreshToken: 'second-refresh',
        user: mockAdminUser,
      }

      vi.mocked(api.login)
        .mockResolvedValueOnce({ data: firstAuth } as AxiosResponse<AuthResponse>)
        .mockResolvedValueOnce({ data: secondAuth } as AxiosResponse<AuthResponse>)

      const store = useAuthStore()
      await store.doLogin('user1', 'pw')
      expect(store.token).toBe('first-token')

      await store.doLogin('admin', 'pw')
      expect(store.token).toBe('second-token')
      expect(store.user).toEqual(mockAdminUser)
      expect(store.isAdmin).toBe(true)
      expect(cacheDelete).toHaveBeenCalledWith('coin-images')
    })
  })

  describe('doRegister', () => {
    it('calls api.register and sets auth state', async () => {
      vi.mocked(api.register).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)
      const store = useAuthStore()

      await store.doRegister('newuser', 'password123', 'new@example.com')

      expect(api.register).toHaveBeenCalledWith('newuser', 'password123', 'new@example.com')
      expect(store.token).toBe('jwt-token-abc')
      expect(store.isAuthenticated).toBe(true)
    })

    it('works without optional email', async () => {
      vi.mocked(api.register).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)
      const store = useAuthStore()

      await store.doRegister('newuser', 'password123')

      expect(api.register).toHaveBeenCalledWith('newuser', 'password123', undefined)
    })
  })

  describe('doWebAuthnLogin', () => {
    it('accepts legacy nested publicKey challenge data from the login begin response', async () => {
      const credential = makeAssertionCredential()
      credentialsGet.mockResolvedValue(credential)
      vi.mocked(api.webauthnLoginBegin).mockResolvedValue({
        data: {
          options: {
            publicKey: {
              challenge: 'AQIDBA',
              rpId: 'coins.example',
              allowCredentials: [
                { id: 'BQYH', type: 'public-key', transports: ['internal'] },
              ],
              userVerification: 'required',
              timeout: 120000,
            },
          },
          username: ' testuser ',
        },
      } as AxiosResponse)
      vi.mocked(api.webauthnLoginFinish).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)

      const store = useAuthStore()
      await store.doWebAuthnLogin(' testuser ')

      expect(api.webauthnLoginBegin).toHaveBeenCalledWith('testuser')
      expect(credentialsGet).toHaveBeenCalledWith({
        publicKey: {
          challenge: new Uint8Array([1, 2, 3, 4]).buffer,
          rpId: 'coins.example',
          allowCredentials: [
            {
              id: new Uint8Array([5, 6, 7]).buffer,
              type: 'public-key',
              transports: ['internal'],
            },
          ],
          userVerification: 'required',
          timeout: 120000,
        },
      })
      expect(api.webauthnLoginFinish).toHaveBeenCalledWith('testuser', credential)
      expect(store.token).toBe(mockAuthResponse.token)
      expect(store.user).toEqual(mockUser)
    })

    it('uses fixed flat login begin options with challenge data directly under options', async () => {
      const credential = makeAssertionCredential()
      credentialsGet.mockResolvedValue(credential)
      vi.mocked(api.webauthnLoginBegin).mockResolvedValue({
        data: {
          options: {
            challenge: 'AQIDBA',
            rpId: 'coins.example',
            allowCredentials: [],
          },
        },
      } as AxiosResponse)
      vi.mocked(api.webauthnLoginFinish).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)

      const store = useAuthStore()
      await store.doWebAuthnLogin('testuser')

      expect(credentialsGet).toHaveBeenCalledWith(expect.objectContaining({
        publicKey: expect.objectContaining({
          challenge: new Uint8Array([1, 2, 3, 4]).buffer,
          rpId: 'coins.example',
          allowCredentials: [],
        }),
      }))
      expect(api.webauthnLoginFinish).toHaveBeenCalledWith('testuser', credential)
    })

    it('fails before invoking browser biometrics when challenge data is missing', async () => {
      vi.mocked(api.webauthnLoginBegin).mockResolvedValue({
        data: {
          options: { publicKey: { rpId: 'coins.example', allowCredentials: [] } },
          username: 'testuser',
        },
      } as AxiosResponse)

      const store = useAuthStore()
      await expect(store.doWebAuthnLogin('testuser')).rejects.toThrow('Biometric login is temporarily unavailable. Missing challenge data.')

      expect(credentialsGet).not.toHaveBeenCalled()
      expect(api.webauthnLoginFinish).not.toHaveBeenCalled()
    })

    it('fails gracefully when all allowCredentials entries are missing ids', async () => {
      vi.mocked(api.webauthnLoginBegin).mockResolvedValue({
        data: {
          options: {
            challenge: 'Zm9v',
            allowCredentials: [{ type: 'public-key' }],
          },
          username: 'testuser',
        },
      } as AxiosResponse<{
        options: { challenge: string; allowCredentials: Array<{ id?: string; type: string }> }
        username: string
      }>)

      const store = useAuthStore()

      await expect(store.doWebAuthnLogin('testuser')).rejects.toThrow(
        'Biometric login is temporarily unavailable. Please sign in with your password and try again.',
      )
      expect(credentialsGet).not.toHaveBeenCalled()
      expect(api.webauthnLoginFinish).not.toHaveBeenCalled()
    })
  })

  describe('logout', () => {
    it('clears all auth state', async () => {
      vi.mocked(api.login).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)
      const store = useAuthStore()
      await store.doLogin('testuser', 'password123')

      store.logout()

      expect(store.token).toBeNull()
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('removes all auth keys from localStorage', async () => {
      vi.mocked(api.login).mockResolvedValue({ data: mockAuthResponse } as AxiosResponse<AuthResponse>)
      const store = useAuthStore()
      await store.doLogin('testuser', 'password123')

      store.logout()

      expect(localStorage.removeItem).toHaveBeenCalledWith('token')
      expect(localStorage.removeItem).toHaveBeenCalledWith('refreshToken')
      expect(localStorage.removeItem).toHaveBeenCalledWith('user')
      expect(cacheDelete).toHaveBeenCalledWith('coin-images')
    })

    it('is safe to call when already logged out', () => {
      const store = useAuthStore()
      expect(() => store.logout()).not.toThrow()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('token refresh sync', () => {
    it('registers a callback via onTokenRefreshed', () => {
      useAuthStore()
      expect(onTokenRefreshed).toHaveBeenCalledWith(expect.any(Function))
    })

    it('updates store state when the refresh callback fires', () => {
      const store = useAuthStore()

      // Capture the callback that was registered
      const registeredCb = vi.mocked(onTokenRefreshed).mock.calls[0]![0]

      const refreshedData: AuthResponse = {
        token: 'refreshed-token',
        refreshToken: 'new-refresh',
        user: mockAdminUser,
      }
      registeredCb(refreshedData)

      expect(store.token).toBe('refreshed-token')
      expect(store.user).toEqual(mockAdminUser)
      expect(store.isAdmin).toBe(true)
    })
  })
})
