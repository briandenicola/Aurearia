<template>
  <div class="auth-page">
    <div class="auth-card">
      <img src="/coin-logo.jpg" alt="Ancient Coins" class="auth-logo" />
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
        🔐 Sign in with Biometrics
      </button>
      <p class="auth-footer">
        Don't have an account? <router-link to="/register">Create one</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { webauthnCheck } from '@/api/client'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const biometricAvailable = ref(false)

const supportsWebAuthn = !!window.PublicKeyCredential

onMounted(() => {
  // Check if we have a remembered username with biometrics
  const lastUser = localStorage.getItem('lastUsername')
  if (lastUser && supportsWebAuthn) {
    username.value = lastUser
    checkBiometric()
  }
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
  try {
    await auth.doLogin(username.value, password.value)
    localStorage.setItem('lastUsername', username.value)
    router.push('/')
  } catch {
    error.value = 'Invalid username or password'
  } finally {
    loading.value = false
  }
}

async function handleBiometricLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.doWebAuthnLogin(username.value)
    localStorage.setItem('lastUsername', username.value)
    router.push('/')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Biometric authentication failed'
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
  color: #e74c3c;
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
}

.auth-footer {
  margin-top: 1.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}
</style>
