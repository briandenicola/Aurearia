<template>
  <div class="min-h-screen flex items-center justify-center p-8 bg-[radial-gradient(ellipse_at_top,var(--bg-secondary)_0%,var(--bg-primary)_70%)]">
    <div class="w-full max-w-[400px] text-center">
      <img
        :src="coinLogoSrc"
        alt="Aurearia - Coin Collection"
        class="w-20 h-20 rounded-full object-cover border-[3px] border-gold-dim mb-6 shadow-[0_0_30px_var(--accent-gold-glow)] mx-auto block"
      />
      <h1 class="mb-1">Aurearia - Coin Collection</h1>
      <p class="text-text-secondary mb-8 text-base">Sign in to your collection</p>
      <form @submit.prevent="handleLogin" class="text-left">
        <div class="form-group">
          <label class="form-label">Username</label>
          <input v-model="username" class="form-input" required autocomplete="username" @blur="checkBiometric" />
        </div>
        <div class="form-group">
          <label class="form-label">Password</label>
          <input v-model="password" type="password" class="form-input" required autocomplete="current-password" />
        </div>
        <p v-if="error" class="text-[var(--color-negative)] text-body mb-2">{{ error }}</p>
        <button
          type="submit"
          class="btn btn-primary w-full justify-center py-3 mt-2"
          :disabled="loading"
        >
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
      </form>
      <button
        v-if="biometricAvailable"
        class="btn btn-secondary w-full justify-center py-3 mt-3 gap-2 text-base"
        :disabled="loading"
        @click="handleBiometricLogin"
      >
        <LockKeyhole :size="18" aria-hidden="true" />
        Sign in with Biometrics
      </button>
      <section
        v-if="oidcProviders.length || oidcLoading || oidcError"
        class="mt-4"
        aria-label="Alternate sign in"
      >
        <div class="flex items-center gap-3 my-4 mb-2 text-text-muted text-chip uppercase tracking-[0.08em] before:content-[''] before:flex-1 before:border-t before:border-border-subtle after:content-[''] after:flex-1 after:border-t after:border-border-subtle">
          <span>or</span>
        </div>
        <p v-if="oidcError" class="text-[var(--color-negative)] text-body mb-2 text-left">{{ oidcError }}</p>
        <button
          v-for="provider in oidcProviders"
          :key="provider.id"
          type="button"
          class="btn btn-secondary w-full justify-center py-3 mt-2 gap-2"
          :disabled="loading || oidcLoading || startingProviderId === provider.id"
          @click="handleOIDCLogin(provider)"
        >
          <LogIn :size="18" aria-hidden="true" />
          {{ oidcButtonLabel(provider.displayName) }}
        </button>
        <p v-if="oidcLoading" class="text-text-muted text-chip mt-2">Loading sign-in providers...</p>
      </section>
      <p class="mt-6 text-body text-text-secondary">
        Don't have an account? <router-link to="/register">Create one</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { getApiErrorMessage, getOIDCPublicProviders, startOIDCLogin, webauthnCheck } from '@/api/client'
import type { OIDCPublicProvider } from '@/types'
import { LockKeyhole, LogIn } from 'lucide-vue-next'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const biometricAvailable = ref(false)
const oidcProviders = ref<OIDCPublicProvider[]>([])
const oidcLoading = ref(false)
const oidcError = ref('')
const startingProviderId = ref<number | null>(null)
const coinLogoSrc = '/coin-logo.jpg'
let retryTimer: ReturnType<typeof setInterval> | null = null

const supportsWebAuthn = !!window.PublicKeyCredential

onMounted(() => {
  oidcError.value = getOIDCCallbackErrorMessage()
  void loadOIDCProviders()

  // Check if we have a remembered username with biometrics
  const lastUser = localStorage.getItem('lastUsername')
  if (lastUser && supportsWebAuthn) {
    username.value = lastUser
    checkBiometric()
  }
})

onUnmounted(() => {
  clearRetryTimer()
})

async function checkBiometric() {
  if (!supportsWebAuthn || !username.value.trim()) {
    biometricAvailable.value = false
    return
  }
  try {
    const res = await webauthnCheck(username.value.trim())
    biometricAvailable.value = res.data.available
  } catch {
    biometricAvailable.value = false
  }
}

async function loadOIDCProviders() {
  oidcLoading.value = true
  try {
    const res = await getOIDCPublicProviders()
    oidcProviders.value = res.data.providers ?? []
  } catch {
    oidcProviders.value = []
  } finally {
    oidcLoading.value = false
  }
}

async function handleLogin() {
  error.value = ''
  loading.value = true
  const trimmedUsername = username.value.trim()
  try {
    await auth.doLogin(trimmedUsername, password.value)
    localStorage.setItem('lastUsername', trimmedUsername)
    router.push('/')
  } catch (err: unknown) {
    if (!handleRateLimitError(err)) {
      error.value = 'Invalid username or password'
    }
  } finally {
    loading.value = false
  }
}

async function handleOIDCLogin(provider: OIDCPublicProvider) {
  oidcError.value = ''
  startingProviderId.value = provider.id
  try {
    const res = await startOIDCLogin(provider.id, {
      redirectPath: '/',
      callbackPath: `/auth/oidc/callback/${provider.id}`,
    })
    const authorizationUrl = res.data.authorizationUrl
    if (!authorizationUrl) {
      oidcError.value = 'The sign-in provider did not return an authorization URL. Ask an administrator to check provider configuration.'
      return
    }
    window.location.assign(authorizationUrl)
  } catch (err: unknown) {
    oidcError.value = getOIDCStartErrorMessage(err)
  } finally {
    startingProviderId.value = null
  }
}

function oidcButtonLabel(displayName: string) {
  const label = displayName.trim() || 'OIDC'
  return `Sign in with ${label}`
}

function getOIDCCallbackErrorMessage() {
  const category = firstQueryValue('oidc_error') ?? firstQueryValue('error')
  const status = firstQueryValue('status')
  const message = firstQueryValue('message')

  if (category) {
    return mapOIDCErrorCategory(category, status, message)
  }

  if (status && ['400', '401', '409', '500'].includes(status)) {
    return mapOIDCErrorCategory('', status, message)
  }

  return ''
}

function firstQueryValue(name: string) {
  const value = route.query[name]
  if (Array.isArray(value)) return value[0] ?? ''
  return value ?? ''
}

function getOIDCStartErrorMessage(err: unknown) {
  const response = getErrorResponse(err)
  const message = getApiErrorMessage(err)
  if (response?.status === 409) {
    return 'This sign-in provider is currently disabled. Ask an administrator to check provider settings.'
  }
  return mapOIDCErrorCategory(message, String(response?.status ?? ''), message)
}

function mapOIDCErrorCategory(category: string, status?: string, message?: string) {
  const normalized = category.toLowerCase()
  if (normalized.includes('access_denied') || normalized.includes('denied') || normalized.includes('cancel')) {
    return 'Sign-in was cancelled or denied at the provider. You can try again or use your local password.'
  }
  if (normalized.includes('conflict') || status === '409') {
    return 'This provider account matches an existing local account. Sign in locally, then link the provider from Account Settings.'
  }
  if (normalized.includes('misconfig') || normalized.includes('configuration') || normalized.includes('discovery') || status === '500') {
    return 'The sign-in provider is not configured correctly. Ask an administrator to test the provider settings.'
  }
  if (normalized.includes('validation') || normalized.includes('state') || normalized.includes('nonce') || normalized.includes('issuer') || normalized.includes('audience') || normalized.includes('signature') || status === '400' || status === '401') {
    return 'The provider response could not be validated. Try again, or ask an administrator to review the provider setup.'
  }
  if (message?.trim()) {
    return message.trim()
  }
  return 'OIDC sign-in failed. Try again or use your local password.'
}

function handleRateLimitError(err: unknown) {
  const response = getErrorResponse(err)
  if (response?.status !== 429) return false

  const retryAfter = getRetryAfterSeconds(response.headers)
  if (retryAfter > 0) {
    startRetryCountdown(retryAfter)
  } else {
    error.value = 'Too many attempts. Try again later.'
  }
  return true
}

function getErrorResponse(err: unknown): { status?: number; headers?: Record<string, unknown> } | null {
  if (typeof err !== 'object' || err === null || !('response' in err)) return null
  const response = (err as { response?: unknown }).response
  if (typeof response !== 'object' || response === null) return null
  return response as { status?: number; headers?: Record<string, unknown> }
}

function getRetryAfterSeconds(headers: Record<string, unknown> | undefined) {
  const raw = headers?.['retry-after'] ?? headers?.['Retry-After']
  if (typeof raw !== 'string') return 0
  const seconds = Number(raw)
  if (Number.isFinite(seconds)) return Math.max(0, Math.ceil(seconds))
  const retryAt = new Date(raw).getTime()
  if (Number.isNaN(retryAt)) return 0
  return Math.max(0, Math.ceil((retryAt - Date.now()) / 1000))
}

function startRetryCountdown(seconds: number) {
  clearRetryTimer()
  let remaining = seconds
  error.value = formatRateLimitMessage(remaining)
  retryTimer = setInterval(() => {
    remaining -= 1
    if (remaining <= 0) {
      clearRetryTimer()
      error.value = 'Too many attempts. Try again later.'
      return
    }
    error.value = formatRateLimitMessage(remaining)
  }, 1000)
}

function formatRateLimitMessage(seconds: number) {
  return `Too many attempts. Try again later. Retry in ${seconds} second${seconds === 1 ? '' : 's'}.`
}

function clearRetryTimer() {
  if (!retryTimer) return
  clearInterval(retryTimer)
  retryTimer = null
}

async function handleBiometricLogin() {
  error.value = ''
  loading.value = true
  const trimmedUsername = username.value.trim()
  try {
    await auth.doWebAuthnLogin(trimmedUsername)
    localStorage.setItem('lastUsername', trimmedUsername)
    router.push('/')
  } catch (e: unknown) {
    // Handle different error types appropriately
    if (e instanceof Error) {
      error.value = e.message
    } else if (typeof e === 'object' && e !== null && 'response' in e) {
      // Axios error - extract server error message if available
      const axiosError = e as { response?: { data?: { error?: string } } }
      error.value = axiosError.response?.data?.error || 'Biometric authentication failed'
    } else {
      error.value = 'Biometric authentication failed'
    }
  } finally {
    loading.value = false
  }
}
</script>

