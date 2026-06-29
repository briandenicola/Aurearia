<template>
  <div class="auth-page">
    <div class="auth-card oidc-result-card">
      <img :src="coinLogoSrc" alt="Aurearia - Coin Collection" class="auth-logo" />
      <div class="status-icon" :class="status">
        <LoaderCircle v-if="status === 'loading'" :size="28" aria-hidden="true" class="spin" />
        <CheckCircle v-else-if="status === 'success'" :size="28" aria-hidden="true" />
        <AlertTriangle v-else :size="28" aria-hidden="true" />
      </div>

      <h1>{{ title }}</h1>
      <p class="auth-subtitle">{{ subtitle }}</p>

      <div v-if="identity" class="identity-summary">
        <div class="summary-row">
          <span class="section-label">Provider</span>
          <strong>{{ identity.providerDisplayName }}</strong>
        </div>
        <div v-if="identity.email" class="summary-row">
          <span class="section-label">Email</span>
          <strong>{{ identity.email }}</strong>
        </div>
      </div>

      <p v-if="message" class="result-message" :class="{ error: status === 'error' }">
        {{ message }}
      </p>

      <router-link class="btn btn-primary auth-btn" to="/settings?tab=account">
        <ArrowLeft :size="18" aria-hidden="true" />
        Back to Account Settings
      </router-link>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { completeOIDCLinkCallback, getApiErrorMessage } from '@/api/client'
import type { OIDCLinkedIdentity } from '@/types'
import { AlertTriangle, ArrowLeft, CheckCircle, LoaderCircle } from 'lucide-vue-next'

type CallbackStatus = 'loading' | 'success' | 'error'

const route = useRoute()
const router = useRouter()

const status = ref<CallbackStatus>('loading')
const identity = ref<OIDCLinkedIdentity | null>(null)
const message = ref('')
const coinLogoSrc = '/coin-logo.jpg'

const title = computed(() => {
  if (status.value === 'loading') return 'Linking Sign-in Provider'
  if (status.value === 'success') return 'Provider Linked'
  return 'Linking Failed'
})

const subtitle = computed(() => {
  if (status.value === 'loading') return 'Finishing the secure provider handshake...'
  if (status.value === 'success') return 'Your external sign-in provider is now connected to this account.'
  return 'The provider could not be linked to your account.'
})

onMounted(() => {
  void completeCallback()
})

async function completeCallback() {
  const providerId = Number(firstParamValue(route.params.providerId))
  const code = firstQueryValue('code')
  const state = firstQueryValue('state')
  const providerError = firstQueryValue('error') || firstQueryValue('error_description')

  void router.replace({ name: 'oidc-link-callback', params: { providerId: route.params.providerId } })

  if (!Number.isInteger(providerId) || providerId <= 0) {
    setError('The provider callback was missing a valid provider. Start linking again from Account Settings.')
    return
  }

  if (providerError) {
    setError(mapProviderError(providerError))
    return
  }

  if (!code || !state) {
    setError('The provider callback was incomplete. Start linking again from Account Settings.')
    return
  }

  try {
    const response = await completeOIDCLinkCallback(providerId, code, state)
    identity.value = response.data.identity
    message.value = response.data.message || 'OIDC identity linked.'
    status.value = 'success'
  } catch (error: unknown) {
    setError(mapCallbackError(error))
  }
}

function setError(text: string) {
  status.value = 'error'
  message.value = text
}

function firstQueryValue(name: string) {
  const value = route.query[name]
  if (Array.isArray(value)) return value[0] ?? ''
  return value ?? ''
}

function firstParamValue(value: string | string[] | undefined) {
  if (Array.isArray(value)) return value[0] ?? ''
  return value
}

function mapProviderError(error: string) {
  const normalized = error.toLowerCase()
  if (normalized.includes('access_denied') || normalized.includes('cancel') || normalized.includes('denied')) {
    return 'Linking was cancelled or denied at the provider. You can try again from Account Settings.'
  }
  return 'The provider returned an error before linking completed. Try again or ask an administrator to review the provider setup.'
}

function mapCallbackError(error: unknown) {
  const response = getErrorResponse(error)
  const messageText = getApiErrorMessage(error)
  const detailText = getErrorDetail(error)
  const normalized = `${messageText} ${detailText}`.toLowerCase()

  if (response?.status === 409) {
    if (normalized.includes('another user') || normalized.includes('already linked')) {
      return 'This provider account is already linked to another user. Sign in with a different provider account or ask an administrator for help.'
    }
    return 'This provider account cannot be linked automatically. Sign in locally with the intended account, then try linking again.'
  }

  if (normalized.includes('redirect uri') || normalized.includes('client secret') || normalized.includes('configuration') || normalized.includes('discovery') || response?.status === 500) {
    return providerConfigurationMessage(detailText)
  }

  if (normalized.includes('state') || normalized.includes('claims') || response?.status === 400 || response?.status === 401) {
    return 'The provider response could not be validated. Start the linking flow again from Account Settings.'
  }

  return messageText || 'The provider could not be linked. Start the linking flow again from Account Settings.'
}

function getErrorResponse(error: unknown): { status?: number } | null {
  if (typeof error !== 'object' || error === null || !('response' in error)) return null
  const response = (error as { response?: unknown }).response
  if (typeof response !== 'object' || response === null) return null
  return response as { status?: number }
}

function getErrorDetail(error: unknown) {
  if (typeof error !== 'object' || error === null || !('response' in error)) return ''
  const response = (error as { response?: { data?: { detail?: unknown } } }).response
  const detail = response?.data?.detail
  return typeof detail === 'string' ? detail : ''
}

function providerConfigurationMessage(detail: string) {
  const safeDetail = detail.trim()
  if (safeDetail) {
    return `The sign-in provider is not configured correctly: ${safeDetail}. Ask an administrator to review the provider settings.`
  }
  return 'The sign-in provider is not configured correctly. Ask an administrator to review the provider settings.'
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: radial-gradient(ellipse at top, var(--bg-secondary) 0%, var(--bg-primary) 70%);
}

.auth-card {
  width: 100%;
  max-width: 420px;
  text-align: center;
}

.oidc-result-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  box-shadow: var(--shadow-card);
}

.auth-logo {
  width: 80px;
  height: 80px;
  border-radius: var(--radius-full);
  object-fit: cover;
  border: 1px solid var(--accent-gold-dim);
  margin-bottom: 1rem;
  box-shadow: var(--shadow-glow);
}

.status-icon {
  width: 48px;
  height: 48px;
  margin: 0 auto 1rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-full);
  border: 1px solid var(--border-subtle);
  background: var(--bg-input);
  color: var(--accent-gold);
}

.status-icon.success {
  color: var(--color-positive);
  border-color: var(--color-positive);
}

.status-icon.error {
  color: var(--color-negative);
  border-color: var(--color-negative);
}

.spin {
  animation: spin 1s linear infinite;
}

.auth-card h1 {
  margin-bottom: 0.25rem;
}

.auth-subtitle {
  color: var(--text-secondary);
  margin-bottom: 1.5rem;
  font-size: 0.9rem;
}

.identity-summary {
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
  margin-bottom: 1rem;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  text-align: left;
}

.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.summary-row strong {
  color: var(--accent-gold);
  font-size: 0.9rem;
  overflow-wrap: anywhere;
  text-align: right;
}

.result-message {
  color: var(--text-secondary);
  font-size: 0.85rem;
  margin-bottom: 1rem;
}

.result-message.error {
  color: var(--color-negative);
}

.auth-btn {
  width: 100%;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
