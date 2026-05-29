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

function getStorageMock(): Record<string, string> {
  const store: Record<string, string> = {}
  return store
}

describe('Auth Store', () => {
  let storageMock: Record<string, string>

  beforeEach(() => {
    storageMock = getStorageMock()

    // Mock localStorage
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storageMock[key] ?? null),
      setItem: vi.fn((key: string, value: string) => { storageMock[key] = value }),
      removeItem: vi.fn((key: string) => { delete storageMock[key] }),
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
