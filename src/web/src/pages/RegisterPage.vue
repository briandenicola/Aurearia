<template>
  <div class="flex min-h-screen items-center justify-center bg-[radial-gradient(ellipse_at_top,var(--bg-secondary)_0%,var(--bg-primary)_70%)] p-8">
    <div class="card w-full max-w-[400px] text-center">
      <img src="/coin-logo.jpg" alt="Aurearia - Coin Collection" class="mx-auto mb-6 h-20 w-20 rounded-full border-[3px] border-[var(--accent-gold-dim)] object-cover shadow-[0_0_30px_var(--accent-gold-glow)]" />
      <h1 class="mb-1">Create Account</h1>
      <p class="mb-8 text-base text-text-secondary">Start tracking your collection</p>
      <form @submit.prevent="handleRegister" class="text-left">
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
        <p v-if="error" class="mb-2 text-body text-warning">{{ error }}</p>
        <button type="submit" class="btn btn-primary mt-2 w-full justify-center py-3" :disabled="loading">
          {{ loading ? 'Creating...' : 'Create Account' }}
        </button>
      </form>
      <p class="mt-6 text-body text-text-secondary">
        Already have an account? <router-link to="/login" class="text-gold transition-colors hover:text-heading">Sign in</router-link>
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
