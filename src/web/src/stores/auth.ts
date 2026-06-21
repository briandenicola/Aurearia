import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { User, AuthResponse } from '@/types'
import * as api from '@/api/client'
import { onTokenRefreshed } from '@/api/client'
import { clearPrivateMediaBlobCache } from '@/utils/media'

const PRIVATE_MEDIA_CACHE_NAMES = ['coin-images']

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const user = ref<User | null>(JSON.parse(localStorage.getItem('user') || 'null'))

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  function setTokens(data: AuthResponse) {
    token.value = data.token
    user.value = data.user
    localStorage.setItem('token', data.token)
    localStorage.setItem('refreshToken', data.refreshToken)
    localStorage.setItem('user', JSON.stringify(data.user))
  }

  function isUserSwitch(nextUser: User) {
    return user.value !== null && user.value.id !== nextUser.id
  }

  async function clearPrivateMediaCaches() {
    clearPrivateMediaBlobCache()
    if (typeof caches === 'undefined') return

    await Promise.allSettled(
      PRIVATE_MEDIA_CACHE_NAMES.map((cacheName) => caches.delete(cacheName)),
    )
  }

  // Keep Pinia store in sync after silent token refresh
  onTokenRefreshed((data) => {
    token.value = data.token
    user.value = data.user
  })

  async function doLogin(username: string, password: string) {
    const res = await api.login(username, password)
    if (isUserSwitch(res.data.user)) {
      await clearPrivateMediaCaches()
    }
    setTokens(res.data)
  }

  async function doRegister(username: string, password: string, email?: string) {
    const res = await api.register(username, password, email)
    if (isUserSwitch(res.data.user)) {
      await clearPrivateMediaCaches()
    }
    setTokens(res.data)
  }

  async function doWebAuthnLogin(username: string) {
    try {
      const ceremonyUsername = username.trim()
      if (!ceremonyUsername) {
        throw new Error('Enter your username before using biometric login')
      }

      // Begin ceremony — get challenge from server
      const beginRes = await api.webauthnLoginBegin(ceremonyUsername)
      const publicKeyOptions = unwrapPublicKeyRequestOptions(beginRes.data.options)
      const finishUsername = beginRes.data.username?.trim() || ceremonyUsername

      // Convert base64url challenge to ArrayBuffer
      const challenge = requireBase64url(publicKeyOptions.challenge, 'challenge')
      const allowCredentials = publicKeyOptions.allowCredentials?.flatMap((c) => {
        if (typeof c?.id !== 'string' || !c.id) {
          return []
        }

        return [{
          id: base64urlToBuffer(c.id),
          type: c.type as PublicKeyCredentialType,
          transports: c.transports as AuthenticatorTransport[] | undefined,
        }]
      })

      if (publicKeyOptions.allowCredentials?.length && !allowCredentials?.length) {
        throw new Error('Biometric login is temporarily unavailable. Please sign in with your password and try again.')
      }

      // Call browser WebAuthn API (triggers Face ID / fingerprint)
      const credential = await navigator.credentials.get({
        publicKey: {
          challenge: base64urlToBuffer(challenge),
          rpId: publicKeyOptions.rpId,
          allowCredentials,
          userVerification: (publicKeyOptions.userVerification as UserVerificationRequirement) || 'preferred',
          timeout: publicKeyOptions.timeout || 60000,
        },
      })

      // Handle null return (user cancelled or no matching credential)
      if (!credential) {
        throw new Error('Biometric authentication was cancelled')
      }

      // Verify it's a PublicKeyCredential
      if (!isPublicKeyCredential(credential)) {
        throw new Error('Invalid credential type returned')
      }

      // Finish ceremony — send assertion to server, get tokens
      const finishRes = await api.webauthnLoginFinish(finishUsername, credential)
      if (isUserSwitch(finishRes.data.user)) {
        await clearPrivateMediaCaches()
      }
      setTokens(finishRes.data)
    } catch (error) {
      // Handle WebAuthn-specific errors
      if (error instanceof DOMException) {
        switch (error.name) {
          case 'NotAllowedError':
            throw new Error('Biometric authentication was cancelled or timed out', { cause: error })
          case 'InvalidStateError':
            throw new Error('No matching biometric credential found', { cause: error })
          case 'SecurityError':
            throw new Error('Biometric authentication is not available on this device', { cause: error })
          default:
            throw new Error(`Biometric authentication failed: ${error.message}`, { cause: error })
        }
      }
      // Re-throw other errors (axios errors, etc.)
      throw error
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('user')
    void clearPrivateMediaCaches()
  }

  return { token, user, isAuthenticated, isAdmin, doLogin, doRegister, doWebAuthnLogin, logout }
})

interface PublicKeyCredentialDescriptorJSON {
  id?: string
  type: string
  transports?: string[]
}

interface PublicKeyCredentialRequestOptionsJSON {
  challenge?: string
  timeout?: number
  rpId?: string
  allowCredentials?: PublicKeyCredentialDescriptorJSON[]
  userVerification?: string
  publicKey?: PublicKeyCredentialRequestOptionsJSON
}

function unwrapPublicKeyRequestOptions(options: PublicKeyCredentialRequestOptionsJSON): PublicKeyCredentialRequestOptionsJSON & { challenge: string } {
  const publicKeyOptions = options.publicKey ?? options
  if (!publicKeyOptions.challenge) {
    throw new Error('Biometric login is temporarily unavailable. Missing challenge data.')
  }
  return publicKeyOptions as PublicKeyCredentialRequestOptionsJSON & { challenge: string }
}

function isPublicKeyCredential(credential: Credential): credential is PublicKeyCredential {
  return credential.type === 'public-key' && 'rawId' in credential && 'response' in credential
}

function base64urlToBuffer(base64url: string): ArrayBuffer {
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
  const pad = base64.length % 4 === 0 ? '' : '='.repeat(4 - (base64.length % 4))
  const binary = atob(base64 + pad)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return bytes.buffer
}

function requireBase64url(value: string | undefined, field: string): string {
  if (typeof value !== 'string' || !value) {
    throw new Error(`Biometric login is temporarily unavailable. Missing ${field} data.`)
  }

  return value
}
