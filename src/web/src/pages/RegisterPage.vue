<template>
  <div class="auth-page">
    <div class="auth-card">
      <img src="/coin-logo.jpg" alt="Aurearia - Coin Collection" class="auth-logo" />
      <h1>Create Account</h1>
      <p class="auth-subtitle">Start tracking your collection</p>
      <form @submit.prevent="handleRegister" class="auth-form">
        <div class="form-group">
          <label class="form-label">Username</label>
          <input v-model="username" class="form-input" required minlength="3" autocomplete="username" />
        </div>
        <div class="form-group">
          <label class="form-label">Email</label>
          <input v-model="email" type="email" class="form-input" required autocomplete="email" placeholder="you@example.com" />
        </div>
        <div class="form-group">
          <label class="form-label">Password</label>
          <input v-model="password" type="password" class="form-input" required minlength="6" autocomplete="new-password" />
        </div>
        <div class="form-group">
          <label class="form-label">Confirm Password</label>
          <input v-model="confirmPassword" type="password" class="form-input" required autocomplete="new-password" />
        </div>
        <p v-if="error" class="auth-error">{{ error }}</p>
        <button type="submit" class="btn btn-primary auth-btn" :disabled="loading">
          {{ loading ? 'Creating...' : 'Create Account' }}
        </button>
      </form>
      <p class="auth-footer">
        Already have an account? <router-link to="/login">Sign in</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { getApiErrorMessage } from '@/api/client'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

async function handleRegister() {
  error.value = ''
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }
  loading.value = true
  try {
    await auth.doRegister(username.value, password.value, email.value)
    router.push('/')
  } catch (e) {
    error.value = getApiErrorMessage(e) || 'Registration failed — username may already exist'
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

.auth-footer {
  margin-top: 1.5rem;
  font-size: 0.85rem;
  color: var(--text-secondary);
}
</style>
