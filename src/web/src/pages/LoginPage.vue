<template>
  <div class="auth-page">
    <div class="auth-card">
      <img :src="coinLogoSrc" alt="Ancient Coins" class="auth-logo" />
      <h1>Ancient Coins</h1>
      <p class="auth-subtitle">Sign in to your collection</p>
      <form @submit.prevent="handleLogin" class="auth-form">
        <div class="form-group">
          <label class="form-label">Username</label>
          <input v-model="username" class="form-input" required autocomplete="username" @blur="checkBiometric" />
        </div>
        <div class="form-group">
          <label class="form-label">Password</label>
          <input v-model="password" type="password" class="form-input" required autocomplete="current-password" />
        </div>
        <p v-if="error" class="auth-error">{{ error }}</p>
        <button type="submit" class="btn btn-primary auth-btn" :disabled="loading">
          {{ loading ? 'Signing in...' : 'Sign In' }}
        </button>
      </form>
      <button
        v-if="biometricAvailable"
        class="btn btn-secondary auth-btn biometric-btn"
        :disabled="loading"
        @click="handleBiometricLogin"
      >
        <LockKeyhole :size="18" aria-hidden="true" />
        Sign in with Biometrics
      </button>
      <p class="auth-footer">
        Don't have an account? <router-link to="/register">Create one</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { webauthnCheck } from '@/api/client'
import { LockKeyhole } from 'lucide-vue-next'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const biometricAvailable = ref(false)
const coinLogoSrc = '/coin-logo.jpg'
let retryTimer: ReturnType<typeof setInterval> | null = null

const supportsWebAuthn = !!window.PublicKeyCredential

onMounted(() => {
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

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background: radial-gradient(ellipse at top, var(--bg-secondary) 0%, var(--bg-primary) 70%);
}

.auth-card {
  width: 100%;
  max-width: 400px;
  text-align: center;
}

.auth-logo {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid var(--accent-gold-dim);
  margin-bottom: 1.5rem;
  box-shadow: 0 0 30px var(--accent-gold-glow);
}

.auth-card h1 {
  margin-bottom: 0.25rem;
}

.auth-subtitle {
  color: var(--text-secondary);
  margin-bottom: 2rem;
  font-size: 0.9rem;
}

.auth-form {
  text-align: left;
}

.auth-error {
  color: var(--color-negative);
  font-size: 0.85rem;
  margin-bottom: 0.5rem;
}

.auth-btn {
  width: 100%;
  justify-content: center;
  padding: 0.75rem;
  margin-top: 0.5rem;
}

.biometric-btn {
  margin-top: 0.75rem;
  font-size: 0.95rem;
  gap: 0.5rem;
}

.auth-footer {
  margin-top: 1.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}
</style>
